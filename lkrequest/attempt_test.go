package lkrequest

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"unsafe"
)

// TestAttemptLimitViewMatchesTheCLayout pins the layout of
// lk_attempt_limit_view_t that ffiAttemptLimitView mirrors: a bool and a
// uint64 for each of the two values, then the int32 scope, 40 bytes in all.
func TestAttemptLimitViewMatchesTheCLayout(t *testing.T) {
	var view ffiAttemptLimitView
	got := []uintptr{
		unsafe.Offsetof(view.configured),
		unsafe.Offsetof(view.effectivePresent),
		unsafe.Offsetof(view.effective),
		unsafe.Offsetof(view.scope),
		unsafe.Sizeof(view),
	}
	if want := []uintptr{8, 16, 24, 32, 40}; !reflect.DeepEqual(got, want) {
		t.Errorf("offsets and size = %v, want %v", got, want)
	}
}

// TestAttemptLimitsAreInherited checks what each level reports: nothing by
// default, its own limit once set, and its parent's while it sets none, a
// request's clone included, until the parent's is cleared too.
func TestAttemptLimitsAreInherited(t *testing.T) {
	requireAttemptAPI(t)

	builder := NewClientBuilder()
	requireAttemptLimit(t, "default client builder", builder.ConnectAttemptLimit, nil, nil, AttemptLimitUnset)
	builder.SetConnectAttemptLimit(4).SetExtraHTTPAttemptLimit(5).SetExtraHTTPAttemptLimit(6).ClearExtraHTTPAttemptLimit()
	requireAttemptLimit(t, "client builder", builder.ConnectAttemptLimit, ptr(4), ptr(4), AttemptLimitClient)
	requireAttemptLimit(t, "cleared client builder", builder.ExtraHTTPAttemptLimit, nil, nil, AttemptLimitUnset)
	client := buildTrustAnchorClient(t, builder.SetExtraHTTPAttemptLimit(5))
	requireAttemptLimit(t, "client", client.ConnectAttemptLimit, ptr(4), ptr(4), AttemptLimitClient)
	requireAttemptLimit(t, "client", client.ExtraHTTPAttemptLimit, ptr(5), ptr(5), AttemptLimitClient)

	sessionBuilder := NewSessionBuilder(client)
	requireAttemptLimit(t, "session builder", sessionBuilder.ExtraHTTPAttemptLimit, nil, ptr(5), AttemptLimitClient)
	sessionBuilder.SetExtraHTTPAttemptLimit(3).SetConnectAttemptLimit(2).ClearConnectAttemptLimit()
	requireAttemptLimit(t, "session builder", sessionBuilder.ExtraHTTPAttemptLimit, ptr(3), ptr(3), AttemptLimitSession)
	session, err := sessionBuilder.Build()
	if err != nil {
		t.Fatalf("SessionBuilder.Build() error = %v", err)
	}
	t.Cleanup(session.Close)
	requireAttemptLimit(t, "session", session.ExtraHTTPAttemptLimit, ptr(3), ptr(3), AttemptLimitSession)
	requireAttemptLimit(t, "session", session.ConnectAttemptLimit, nil, ptr(4), AttemptLimitClient)

	request := session.Get("http://127.0.0.1:9/")
	requireAttemptLimit(t, "request", request.ExtraHTTPAttemptLimit, nil, ptr(3), AttemptLimitSession)
	request.SetExtraHTTPAttemptLimit(1)
	requireAttemptLimit(t, "request", request.ExtraHTTPAttemptLimit, ptr(1), ptr(1), AttemptLimitRequest)
	clone, err := request.Clone()
	if err != nil {
		t.Fatalf("Request.Clone() error = %v", err)
	}
	requireAttemptLimit(t, "clone", clone.ExtraHTTPAttemptLimit, ptr(1), ptr(1), AttemptLimitRequest)
	request.ClearExtraHTTPAttemptLimit()
	requireAttemptLimit(t, "cleared request", request.ExtraHTTPAttemptLimit, nil, ptr(3), AttemptLimitSession)
}

// TestAttemptLimitsCannotRelaxTheirParent checks that a session or a request
// cannot loosen the limit it inherits: Build and Send report ErrInvalidConfig,
// and the request keeps the limit it had.
func TestAttemptLimitsCannotRelaxTheirParent(t *testing.T) {
	requireAttemptAPI(t)
	client := buildTrustAnchorClient(t, NewClientBuilder().SetExtraHTTPAttemptLimit(5))

	_, err := NewSessionBuilder(client).SetExtraHTTPAttemptLimit(7).Build()
	requireErrorCode(t, "SessionBuilder.Build() with a looser limit", err, ErrInvalidConfig)

	request := newTrustAnchorSession(t, client).Get("http://127.0.0.1:9/").SetExtraHTTPAttemptLimit(2)
	request.SetExtraHTTPAttemptLimit(9)
	requireAttemptLimit(t, "request after the rejected limit", request.ExtraHTTPAttemptLimit, ptr(2), ptr(2), AttemptLimitRequest)
	_, err = request.Send()
	requireErrorCode(t, "Send() with a looser limit", err, ErrInvalidConfig)
}

// TestConnectAttemptLimitZeroStartsNoConnection sends a request allowed no
// connection start: it fails with ErrResourceLimitExceeded, and its attempt
// summary names the connect limit as the cause.
func TestConnectAttemptLimitZeroStartsNoConnection(t *testing.T) {
	requireAttemptAPI(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("the server was reached by a request allowed no connection")
	}))
	t.Cleanup(server.Close)

	session := buildTestSession(t, NewClientBuilder().SetConnectAttemptLimit(0))
	_, err := session.Get(server.URL).EnableAttemptSummary(true).Send()
	requireErrorCode(t, "Send()", err, ErrResourceLimitExceeded)
	var lkErr *LkError
	if errors.As(err, &lkErr) {
		summary := decodeAttemptSummary(t, lkErr.AttemptSummaryJSON())
		requireSummaryFields(t, summary, map[string]any{
			"outcome": "failed", "final": true, "denial_reason": "connect_limit",
			"http_submissions": 0.0, "connect_initiations": 0.0,
		})
	}
}

// TestAttemptSummaryReportsTheAttempts checks the attempt summaries of a
// buffered and a streaming response: present only when the request enables
// one, final, and counting the submission, the connection the first request
// opens and the effective limits with their scopes.
func TestAttemptSummaryReportsTheAttempts(t *testing.T) {
	requireAttemptAPI(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	t.Cleanup(server.Close)
	client := buildTrustAnchorClient(t, NewClientBuilder().SetConnectAttemptLimit(4))
	session, err := NewSessionBuilder(client).SetExtraHTTPAttemptLimit(3).Build()
	if err != nil {
		t.Fatalf("SessionBuilder.Build() error = %v", err)
	}
	t.Cleanup(session.Close)

	resp, err := session.Get(server.URL).EnableAttemptSummary(true).Send()
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	summary := decodeAttemptSummary(t, resp.AttemptSummaryJSON())
	resp.Close()
	requireSummaryFields(t, summary, map[string]any{
		"schema_version": 1.0, "outcome": "succeeded", "final": true,
		"http_submissions": 1.0, "extra_http_submissions": 0.0, "connect_initiations": 1.0,
	})
	if got, want := summary["effective_limits"], map[string]any{
		"connect_attempts": 4.0, "connect_scope": "client", "extra_http_attempts": 3.0, "extra_http_scope": "session",
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("effective_limits = %v, want %v", got, want)
	}

	stream, err := session.Get(server.URL).EnableAttemptSummary(true).SendStreaming()
	if err != nil {
		t.Fatalf("SendStreaming() error = %v", err)
	}
	summary = decodeAttemptSummary(t, stream.AttemptSummaryJSON())
	_ = stream.Close()
	requireSummaryFields(t, summary, map[string]any{"outcome": "succeeded", "final": true, "http_submissions": 1.0})

	resp, err = session.Get(server.URL).Send()
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	defer resp.Close()
	if got := resp.AttemptSummaryJSON(); got != "" {
		t.Errorf("AttemptSummaryJSON() of a request without a summary = %s, want \"\"", got)
	}
}

// TestRequestsAreReplayedOnlyWhenAllowed sends a second request on a pooled
// connection that the server closes after reading it, without an answer. By
// default the request fails; SetRetryOnConnectionClose sends it again on a new
// connection, and an extra HTTP attempt limit of 0 forbids that.
func TestRequestsAreReplayedOnlyWhenAllowed(t *testing.T) {
	requireAttemptAPI(t)

	for _, tc := range []struct {
		name          string
		builder       *ClientBuilder
		wantReplay    bool
		wantSummaries map[string]any
	}{
		{"default", NewClientBuilder(), false, map[string]any{"outcome": "failed", "extra_http_submissions": 0.0}},
		{"replay", NewClientBuilder().SetRetryOnConnectionClose(true), true, map[string]any{"outcome": "succeeded", "extra_http_submissions": 1.0}},
		{"replay, no extra attempt", NewClientBuilder().SetRetryOnConnectionClose(true).SetExtraHTTPAttemptLimit(0), false,
			map[string]any{"outcome": "failed", "extra_http_submissions": 0.0, "denial_reason": "extra_http_limit"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			url, connections := secondRequestDroppingServer(t)
			session := buildTestSession(t, tc.builder)
			first, err := session.Get(url).Send()
			if err != nil {
				t.Fatalf("first Send() error = %v", err)
			}
			first.Close()

			resp, err := session.Get(url).EnableAttemptSummary(true).Send()
			var summaryJSON string
			if tc.wantReplay {
				if err != nil {
					t.Fatalf("second Send() error = %v, want it replayed", err)
				}
				summaryJSON = resp.AttemptSummaryJSON()
				resp.Close()
			} else {
				if err == nil {
					resp.Close()
					t.Fatal("second Send() succeeded, want the dropped request to fail")
				}
				var lkErr *LkError
				if !errors.As(err, &lkErr) {
					t.Fatalf("second Send() error = %v (%T), want an *LkError", err, err)
				}
				summaryJSON = lkErr.AttemptSummaryJSON()
			}
			requireSummaryFields(t, decodeAttemptSummary(t, summaryJSON), tc.wantSummaries)
			if want := map[bool]int{true: 2, false: 1}[tc.wantReplay]; connections() != want {
				t.Errorf("the server accepted %d connections, want %d", connections(), want)
			}
		})
	}
}

// secondRequestDroppingServer starts an HTTP/1.1 server that answers the
// first request on its first connection, keeping the connection open, and
// closes that connection without an answer once it has read a second request
// on it. Every later connection is answered normally. It returns the URL to
// request and a function reporting the connections accepted so far.
func secondRequestDroppingServer(t *testing.T) (string, func() int) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	t.Cleanup(func() { listener.Close() })

	var (
		mu       sync.Mutex
		accepted int
	)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			accepted++
			first := accepted == 1
			mu.Unlock()
			go serveUntilDropped(conn, first)
		}
	}()
	return "http://" + listener.Addr().String() + "/", func() int {
		mu.Lock()
		defer mu.Unlock()
		return accepted
	}
}

// serveUntilDropped answers each request read from conn with "ok"; when drop
// is set, it closes conn instead of answering the second request.
func serveUntilDropped(conn net.Conn, drop bool) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for served := 0; ; served++ {
		request, err := http.ReadRequest(reader)
		if err != nil {
			return
		}
		_, _ = io.Copy(io.Discard, request.Body)
		if drop && served == 1 {
			return
		}
		if _, err := io.WriteString(conn, "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok"); err != nil {
			return
		}
	}
}

// requireAttemptLimit fails the test unless get reports the limit wanted.
func requireAttemptLimit(t *testing.T, what string, get func() (AttemptLimit, error), configured, effective *uint64, scope AttemptLimitScope) {
	t.Helper()

	got, err := get()
	if err != nil {
		t.Fatalf("%s limit error = %v", what, err)
	}
	if want := (AttemptLimit{Configured: configured, Effective: effective, Scope: scope}); !reflect.DeepEqual(got, want) {
		t.Errorf("%s limit = %s, want %s", what, formatAttemptLimit(got), formatAttemptLimit(want))
	}
}

// formatAttemptLimit renders limit with its optional values dereferenced.
func formatAttemptLimit(limit AttemptLimit) string {
	value := func(p *uint64) string {
		if p == nil {
			return "nil"
		}
		return strconv.FormatUint(*p, 10)
	}
	return "{Configured:" + value(limit.Configured) + " Effective:" + value(limit.Effective) + " Scope:" + limit.Scope.String() + "}"
}

// ptr returns a pointer to n.
func ptr(n uint64) *uint64 {
	return &n
}

// decodeAttemptSummary decodes an attempt summary, failing the test when
// there is none.
func decodeAttemptSummary(t *testing.T, raw string) map[string]any {
	t.Helper()

	var summary map[string]any
	if err := json.Unmarshal([]byte(raw), &summary); err != nil {
		t.Fatalf("decode attempt summary %q error = %v", raw, err)
	}
	return summary
}

// requireSummaryFields fails the test unless summary holds each of want's
// fields with its value.
func requireSummaryFields(t *testing.T, summary map[string]any, want map[string]any) {
	t.Helper()

	for field, value := range want {
		if got := summary[field]; got != value {
			t.Errorf("attempt summary %s = %v, want %v; summary = %v", field, got, value, summary)
		}
	}
}

// requireErrorCode fails the test unless err is an LkError with code.
func requireErrorCode(t *testing.T, what string, err error, code ErrorCode) {
	t.Helper()

	var lkErr *LkError
	if !errors.As(err, &lkErr) || lkErr.Code() != code {
		t.Errorf("%s error = %v (%T), want an LkError with code %v", what, err, err, code)
	}
}

// requireAttemptAPI skips the test on a library without attempt limits and
// summaries.
func requireAttemptAPI(t *testing.T) {
	t.Helper()
	if ffi_lk_request_enable_attempt_summary == nil || ffi_lk_client_builder_set_connect_attempt_limit == nil {
		t.Skip("loaded lkrequest library predates attempt limits and summaries")
	}
}
