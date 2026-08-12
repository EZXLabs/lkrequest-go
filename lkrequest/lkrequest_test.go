package lkrequest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	testNetworkAttempts = 3
	testRetryDelay      = 200 * time.Millisecond
)

func TestABIVersion(t *testing.T) {
	if got := ABIVersion(); got == 0 {
		t.Fatalf("ABIVersion() = %d, want > 0", got)
	}
}

func TestLibraryVersion(t *testing.T) {
	if got := LibraryVersion(); got == "" {
		t.Fatal("LibraryVersion() returned empty string")
	}
}

func TestNewDefaultClient(t *testing.T) {
	client, err := NewDefaultClient()
	if err != nil {
		t.Fatalf("NewDefaultClient() error = %v", err)
	}
	client.Close()
}

func TestClientBuilder(t *testing.T) {
	client, err := NewClientBuilder().
		SetVerify(true).
		SetTimeoutDNS(500).
		SetTimeoutTCPConnect(1000).
		SetTimeoutTLSHandshake(1000).
		SetTimeoutTotal(3000).
		SetMaxOutstandingOps(8).
		SetMaxHeaderCount(64).
		AddDefaultHeader("accept", "*/*").
		AddHeaderOrder("accept").
		SetRetryOnConnectionClose(true).
		Build()
	if err != nil {
		t.Fatalf("ClientBuilder.Build() error = %v", err)
	}
	client.Close()
}

func TestSimpleGet(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req, err := NewRequest(session, "GET", "https://httpbin.org/get")
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	resp, err := req.Send()
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	defer resp.Close()

	if got := resp.StatusCode(); got != 200 {
		t.Fatalf("StatusCode() = %d, want 200", got)
	}
}

func TestPostJSON(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req, err := NewRequest(session, "POST", "https://httpbin.org/post")
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	resp, err := req.SetJSONBody(`{"hello":"world"}`).Send()
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	defer resp.Close()

	if got := resp.StatusCode(); got != 200 {
		t.Fatalf("StatusCode() = %d, want 200", got)
	}

	var payload struct {
		JSON map[string]any `json:"json"`
	}
	if err := resp.UnmarshalJSON(&payload); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	if got := payload.JSON["hello"]; got != "world" {
		t.Fatalf("payload.JSON[\"hello\"] = %#v, want %q", got, "world")
	}
}

func TestStreaming(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req, err := NewRequest(session, "GET", "https://httpbin.org/stream/3")
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	stream, err := req.SendStreaming()
	if err != nil {
		t.Fatalf("SendStreaming() error = %v", err)
	}
	defer func() {
		if closeErr := stream.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	}()

	body, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if len(body) == 0 {
		t.Fatal("stream body is empty")
	}
}

func TestContextCancel(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req, err := NewRequest(session, "GET", "https://httpbin.org/delay/3")
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	resp, err := req.SendWithContext(ctx)
	if resp != nil {
		resp.Close()
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("SendWithContext() error = %v, want %v", err, context.DeadlineExceeded)
	}
}

func TestRequestConsumed(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req, err := NewRequest(session, "GET", "https://httpbin.org/get")
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	resp, err := req.Send()
	if err != nil {
		t.Fatalf("first Send() error = %v", err)
	}
	resp.Close()

	if _, err := req.Send(); !errors.Is(err, ErrRequestConsumed) {
		t.Fatalf("second Send() error = %v, want %v", err, ErrRequestConsumed)
	}
}

func TestSugarGet(t *testing.T) {
	requireNetworkTests(t)

	resp, err := Get("https://httpbin.org/get")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Close()

	if got := resp.StatusCode(); got != 200 {
		t.Fatalf("StatusCode() = %d, want 200", got)
	}
}

func TestResponseHelpers(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req, err := NewRequest(session, "GET", "https://httpbin.org/json")
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	resp, err := req.Send()
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	defer resp.Close()

	if ct := resp.Header("content-type"); !strings.Contains(strings.ToLower(ct), "application/json") {
		t.Fatalf("Header(content-type) = %q, want application/json", ct)
	}

	var doc map[string]any
	if err := json.Unmarshal(resp.Bytes(), &doc); err != nil {
		t.Fatalf("json.Unmarshal(Bytes()) error = %v", err)
	}
}

func TestErrorCodeString(t *testing.T) {
	cases := []struct {
		code ErrorCode
		want string
	}{
		{ErrUnknown, "ErrUnknown"},
		{ErrInvalidArgument, "ErrInvalidArgument"},
		{ErrInternalPanic, "ErrInternalPanic"},
		{ErrStreamClosed, "ErrStreamClosed"},
		{ErrInvalidHandle, "ErrInvalidHandle"},
		{ErrBusy, "ErrBusy"},
		{ErrResourceLimitExceeded, "ErrResourceLimitExceeded"},
		{ErrInvalidConfig, "ErrInvalidConfig"},
		{ErrNotFound, "ErrNotFound"},
		{ErrDecompressionFailed, "ErrDecompressionFailed"},
		{ErrTLS, "ErrTLS"},
		{ErrHTTP, "ErrHTTP"},
		{ErrH2, "ErrH2"},
		{ErrIO, "ErrIO"},
		{ErrProxy, "ErrProxy"},
		{ErrConnection, "ErrConnection"},
		{ErrTimeout, "ErrTimeout"},
		{ErrTooManyRedirects, "ErrTooManyRedirects"},
		{ErrPool, "ErrPool"},
		{ErrURLParse, "ErrURLParse"},
		{ErrStatus, "ErrStatus"},
		{ErrQUIC, "ErrQUIC"},
		{ErrH3, "ErrH3"},
		{ErrorCode(99), "ErrorCode(99)"},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.code.String(); got != tc.want {
				t.Fatalf("ErrorCode(%d).String() = %q, want %q", tc.code, got, tc.want)
			}
		})
	}
}

func TestPhaseString(t *testing.T) {
	cases := []struct {
		phase Phase
		want  string
	}{
		{PhaseNone, "PhaseNone"},
		{PhaseDNSResolution, "PhaseDNSResolution"},
		{PhaseTCPConnect, "PhaseTCPConnect"},
		{PhaseProxyTunnel, "PhaseProxyTunnel"},
		{PhaseTLSHandshake, "PhaseTLSHandshake"},
		{PhaseH2Negotiation, "PhaseH2Negotiation"},
		{PhaseH2CUpgrade, "PhaseH2CUpgrade"},
		{PhaseHTTPRequest, "PhaseHTTPRequest"},
		{PhaseQUICHandshake, "PhaseQUICHandshake"},
		{PhaseH3Negotiation, "PhaseH3Negotiation"},
		{PhaseQUICFallback, "PhaseQUICFallback"},
		{Phase(99), "Phase(99)"},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.phase.String(); got != tc.want {
				t.Fatalf("Phase(%d).String() = %q, want %q", tc.phase, got, tc.want)
			}
		})
	}
}

func TestHttpVersionString(t *testing.T) {
	cases := []struct {
		version HttpVersion
		want    string
	}{
		{HttpVersionUnknown, "HttpVersionUnknown"},
		{HttpVersion10, "HttpVersion10"},
		{HttpVersion11, "HttpVersion11"},
		{HttpVersion2, "HttpVersion2"},
		{HttpVersion3, "HttpVersion3"},
		{HttpVersion(99), "HttpVersion(99)"},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.version.String(); got != tc.want {
				t.Fatalf("HttpVersion(%d).String() = %q, want %q", tc.version, got, tc.want)
			}
		})
	}
}

func TestOpStateString(t *testing.T) {
	cases := []struct {
		state OpState
		want  string
	}{
		{OpInProgress, "OpInProgress"},
		{OpCompletedOK, "OpCompletedOK"},
		{OpCompletedErr, "OpCompletedErr"},
		{OpCancelled, "OpCancelled"},
		{OpConsumed, "OpConsumed"},
		{OpState(99), "OpState(99)"},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.state.String(); got != tc.want {
				t.Fatalf("OpState(%d).String() = %q, want %q", tc.state, got, tc.want)
			}
		})
	}
}

func TestAcceptEncodingString(t *testing.T) {
	cases := []struct {
		name string
		bits AcceptEncoding
		want string
	}{
		{"zero", 0, "AcceptEncoding(0)"},
		{"gzip", AcceptEncodingGzip, "AcceptEncodingGzip"},
		{"br", AcceptEncodingBr, "AcceptEncodingBr"},
		{"deflate", AcceptEncodingDeflate, "AcceptEncodingDeflate"},
		{"zstd", AcceptEncodingZstd, "AcceptEncodingZstd"},
		{
			"combo",
			AcceptEncodingGzip | AcceptEncodingBr | AcceptEncodingZstd,
			"AcceptEncodingGzip|AcceptEncodingBr|AcceptEncodingZstd",
		},
		{
			"unknown-bit",
			AcceptEncodingGzip | AcceptEncoding(0x40),
			"AcceptEncodingGzip|AcceptEncoding(0x40)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.bits.String(); got != tc.want {
				t.Fatalf("AcceptEncoding(%d).String() = %q, want %q", tc.bits, got, tc.want)
			}
		})
	}
}

func TestClientClone(t *testing.T) {
	client, err := NewDefaultClient()
	if err != nil {
		t.Fatalf("NewDefaultClient() error = %v", err)
	}
	t.Cleanup(client.Close)

	clone := client.Clone()
	if clone == nil {
		t.Fatal("Client.Clone() returned nil")
	}
	t.Cleanup(clone.Close)
}

func TestClientFingerprintInfoJSON(t *testing.T) {
	client, err := NewDefaultClient()
	if err != nil {
		t.Fatalf("NewDefaultClient() error = %v", err)
	}
	t.Cleanup(client.Close)

	info, err := client.FingerprintInfoJSON()
	if err != nil {
		t.Fatalf("FingerprintInfoJSON() error = %v", err)
	}
	if info == "" {
		t.Fatal("FingerprintInfoJSON() returned empty string")
	}

	payload := decodeJSONText[map[string]any](t, info)
	for _, key := range []string{"tls", "h2", "quic", "tcp"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("FingerprintInfoJSON() missing %q key in %s", key, info)
		}
	}
}

func TestClientDoubleClose(t *testing.T) {
	client, err := NewDefaultClient()
	if err != nil {
		t.Fatalf("NewDefaultClient() error = %v", err)
	}

	client.Close()
	client.Close()
}

func TestSessionBuilder(t *testing.T) {
	client, err := NewDefaultClient()
	if err != nil {
		t.Fatalf("NewDefaultClient() error = %v", err)
	}
	t.Cleanup(client.Close)

	t.Run("fixed retry", func(t *testing.T) {
		session, err := NewSessionBuilder(client).
			SetProxy("http://127.0.0.1:8888").
			SetMaxRedirects(5).
			SetHTTP2Only().
			SetDefaultAcceptEncoding(AcceptEncodingGzip|AcceptEncodingBr).
			SetMaxConnections(4).
			SetIdleTimeout(1000).
			SetRetryFixed(2, 100).
			Build()
		if err != nil {
			t.Fatalf("SessionBuilder.Build() error = %v", err)
		}
		t.Cleanup(session.Close)
	})

	t.Run("exponential retry", func(t *testing.T) {
		session, err := NewSessionBuilder(client).
			SetMaxRedirects(3).
			SetHTTP1Only().
			SetDefaultAcceptEncoding(AcceptEncodingDeflate|AcceptEncodingZstd).
			SetMaxConnections(2).
			SetIdleTimeout(1500).
			SetRetryExponential(3, 50, 500, true).
			Build()
		if err != nil {
			t.Fatalf("SessionBuilder.Build() error = %v", err)
		}
		t.Cleanup(session.Close)
	})
}

func TestSessionClone(t *testing.T) {
	client, session := newTestSession(t)
	t.Cleanup(client.Close)
	t.Cleanup(session.Close)

	clone := session.Clone()
	if clone == nil {
		t.Fatal("Session.Clone() returned nil")
	}
	t.Cleanup(clone.Close)
}

func TestSessionDoubleClose(t *testing.T) {
	client, session := newTestSession(t)
	t.Cleanup(client.Close)

	session.Close()
	session.Close()
}

func TestRequestAddQuery(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req := mustNewRequest(t, session, "GET", "https://httpbin.org/get")

	resp := mustSendRequest(t, req.AddQuery("alpha", "one").AddQuery("beta", "two"))
	cleanupResponse(t, resp)

	if got := resp.URL(); !strings.Contains(got, "alpha=one") || !strings.Contains(got, "beta=two") {
		t.Fatalf("URL() = %q, want query params", got)
	}

	payload := decodeResponseJSON[httpbinAnythingResponse](t, resp)
	if got := payload.Args["alpha"]; got != "one" {
		t.Fatalf("payload.Args[alpha] = %q, want %q", got, "one")
	}
	if got := payload.Args["beta"]; got != "two" {
		t.Fatalf("payload.Args[beta] = %q, want %q", got, "two")
	}
}

func TestRequestSetBodyBytes(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req := mustNewRequest(t, session, "POST", "https://httpbin.org/post")

	resp := mustSendRequest(t, req.SetBodyBytes([]byte("raw-bytes")))
	cleanupResponse(t, resp)

	payload := decodeResponseJSON[httpbinAnythingResponse](t, resp)
	if got := payload.Data; got != "raw-bytes" {
		t.Fatalf("payload.Data = %q, want %q", got, "raw-bytes")
	}
}

func TestRequestSetTextBody(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req := mustNewRequest(t, session, "POST", "https://httpbin.org/post")

	resp := mustSendRequest(t, req.SetTextBody("plain-text-body"))
	cleanupResponse(t, resp)

	payload := decodeResponseJSON[httpbinAnythingResponse](t, resp)
	if got := payload.Data; got != "plain-text-body" {
		t.Fatalf("payload.Data = %q, want %q", got, "plain-text-body")
	}
}

func TestRequestSetForm(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req := mustNewRequest(t, session, "POST", "https://httpbin.org/post")

	resp := mustSendRequest(t, req.SetForm(map[string]string{
		"alpha": "one",
		"beta":  "two",
	}))
	cleanupResponse(t, resp)

	payload := decodeResponseJSON[httpbinAnythingResponse](t, resp)
	if got := payload.Form["alpha"]; got != "one" {
		t.Fatalf("payload.Form[alpha] = %q, want %q", got, "one")
	}
	if got := payload.Form["beta"]; got != "two" {
		t.Fatalf("payload.Form[beta] = %q, want %q", got, "two")
	}
}

func TestRequestSetCookie(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req := mustNewRequest(t, session, "GET", "https://httpbin.org/cookies")

	resp := mustSendRequest(t, req.SetCookie("session", "cookie-value"))
	cleanupResponse(t, resp)

	payload := decodeResponseJSON[httpbinCookiesResponse](t, resp)
	if got := payload.Cookies["session"]; got != "cookie-value" {
		t.Fatalf("payload.Cookies[session] = %q, want %q", got, "cookie-value")
	}
}

func TestRequestSetBasicAuth(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req := mustNewRequest(t, session, "GET", "https://httpbin.org/basic-auth/user/passwd")

	resp := mustSendRequest(t, req.SetBasicAuth("user", "passwd"))
	cleanupResponse(t, resp)

	payload := decodeResponseJSON[httpbinBasicAuthResponse](t, resp)
	if !payload.Authenticated {
		t.Fatal("payload.Authenticated = false, want true")
	}
	if got := payload.User; got != "user" {
		t.Fatalf("payload.User = %q, want %q", got, "user")
	}
}

func TestRequestSetBearerAuth(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req := mustNewRequest(t, session, "GET", "https://httpbin.org/bearer")

	resp := mustSendRequest(t, req.SetBearerAuth("token-123"))
	cleanupResponse(t, resp)

	payload := decodeResponseJSON[httpbinBearerResponse](t, resp)
	if !payload.Authenticated {
		t.Fatal("payload.Authenticated = false, want true")
	}
	if got := payload.Token; got != "token-123" {
		t.Fatalf("payload.Token = %q, want %q", got, "token-123")
	}
}

func TestRequestSetCustomHeaders(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req := mustNewRequest(t, session, "GET", "https://httpbin.org/anything")

	resp := mustSendRequest(t, req.
		AddHeader("X-Test-One", "one").
		AddHeader("X-Test-Two", "two"))
	cleanupResponse(t, resp)

	payload := decodeResponseJSON[httpbinAnythingResponse](t, resp)
	if got := payload.Headers["X-Test-One"]; got != "one" {
		t.Fatalf("payload.Headers[X-Test-One] = %q, want %q", got, "one")
	}
	if got := payload.Headers["X-Test-Two"]; got != "two" {
		t.Fatalf("payload.Headers[X-Test-Two] = %q, want %q", got, "two")
	}
}

func TestRequestClone(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req := mustNewRequest(t, session, "POST", "https://httpbin.org/anything/clone")
	req = req.
		AddHeader("X-Clone", "true").
		AddQuery("copy", "yes").
		SetTextBody("clone-body")

	clone, err := req.Clone()
	if err != nil {
		t.Fatalf("Clone() error = %v", err)
	}

	resp := mustSendRequest(t, req)
	cleanupResponse(t, resp)

	cloneResp := mustSendRequest(t, clone)
	cleanupResponse(t, cloneResp)

	for _, payload := range []httpbinAnythingResponse{
		decodeResponseJSON[httpbinAnythingResponse](t, resp),
		decodeResponseJSON[httpbinAnythingResponse](t, cloneResp),
	} {
		if got := payload.Data; got != "clone-body" {
			t.Fatalf("payload.Data = %q, want %q", got, "clone-body")
		}
		if got := payload.Headers["X-Clone"]; got != "true" {
			t.Fatalf("payload.Headers[X-Clone] = %q, want %q", got, "true")
		}
		if !strings.Contains(payload.URL, "copy=yes") {
			t.Fatalf("payload.URL = %q, want copy=yes", payload.URL)
		}
	}
}

func TestSendAsync(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req := mustNewRequest(t, session, "GET", "https://httpbin.org/get")

	respCh, errCh := req.SendAsync(context.Background())
	resp := awaitAsyncResponse(t, respCh, errCh)
	cleanupResponse(t, resp)

	if got := resp.StatusCode(); got != 200 {
		t.Fatalf("StatusCode() = %d, want 200", got)
	}
}

func TestSendAsyncCancel(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req := mustNewRequest(t, session, "GET", "https://httpbin.org/delay/3")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	respCh, errCh := req.SendAsync(ctx)
	err := awaitAsyncError[*Response](t, respCh, errCh)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SendAsync() error = %v, want %v", err, context.Canceled)
	}
}

func TestSendStreamingAsync(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req := mustNewRequest(t, session, "GET", "https://httpbin.org/stream/3")

	streamCh, errCh := req.SendStreamingAsync(context.Background())
	stream := awaitAsyncStream(t, streamCh, errCh)
	cleanupStream(t, stream)

	body, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if len(body) == 0 {
		t.Fatal("stream body is empty")
	}
}

func TestResponseURL(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req := mustNewRequest(
		t,
		session,
		"GET",
		"https://httpbin.org/redirect-to?url=https%3A%2F%2Fhttpbin.org%2Fget%3Fsource%3Durl-test",
	)

	resp := mustSendRequest(t, req)
	cleanupResponse(t, resp)

	if got := resp.URL(); got != "https://httpbin.org/get?source=url-test" {
		t.Fatalf("URL() = %q, want %q", got, "https://httpbin.org/get?source=url-test")
	}
}

func TestResponseVersion(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	resp := mustSendRequest(t, mustNewRequest(t, session, "GET", "https://httpbin.org/get"))
	cleanupResponse(t, resp)

	switch got := resp.Version(); got {
	case HttpVersion10, HttpVersion11, HttpVersion2, HttpVersion3:
	default:
		t.Fatalf("Version() = %s, want a known HTTP version", got)
	}
}

func TestResponseHeaders(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	resp := mustSendRequest(t, mustNewRequest(t, session, "GET", "https://httpbin.org/json"))
	cleanupResponse(t, resp)

	headers := resp.Headers()
	if headers.Get("Content-Type") == "" {
		t.Fatal("Headers().Get(Content-Type) returned empty string")
	}
	if want := resp.Header("content-type"); headers.Get("Content-Type") != want {
		t.Fatalf("Headers().Get(Content-Type) = %q, want %q", headers.Get("Content-Type"), want)
	}
}

func TestResponseContentLength(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	resp := mustSendRequest(t, mustNewRequest(t, session, "GET", "https://httpbin.org/bytes/64"))
	cleanupResponse(t, resp)

	if got := resp.ContentLength(); got != 64 {
		t.Fatalf("ContentLength() = %d, want 64", got)
	}
	if got := len(resp.Bytes()); got != 64 {
		t.Fatalf("len(Bytes()) = %d, want 64", got)
	}
}

func TestResponseString(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	resp := mustSendRequest(t, mustNewRequest(t, session, "GET", "https://httpbin.org/base64/aGVsbG8gd29ybGQ="))
	cleanupResponse(t, resp)

	if got := resp.String(); got != "hello world" {
		t.Fatalf("String() = %q, want %q", got, "hello world")
	}
}

func TestStreamingResponseHeaders(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	stream, err := mustNewRequest(t, session, "GET", "https://httpbin.org/stream/3").SendStreaming()
	if err != nil {
		t.Fatalf("SendStreaming() error = %v", err)
	}
	cleanupStream(t, stream)

	if got := stream.StatusCode(); got != 200 {
		t.Fatalf("StatusCode() = %d, want 200", got)
	}
	if headers := stream.Headers(); headers.Get("Content-Type") == "" {
		t.Fatal("Headers().Get(Content-Type) returned empty string")
	}

	buf := make([]byte, 16)
	n, err := stream.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("Read() error = %v", err)
	}
	if n == 0 {
		t.Fatal("Read() returned no data")
	}
}

func TestStreamingResponseSmallReads(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	stream, err := mustNewRequest(t, session, "GET", "https://httpbin.org/stream/5").SendStreaming()
	if err != nil {
		t.Fatalf("SendStreaming() error = %v", err)
	}
	cleanupStream(t, stream)

	buf := make([]byte, 7)
	var out strings.Builder
	reads := 0

	for {
		n, err := stream.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			reads++
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
	}

	if reads < 2 {
		t.Fatalf("Read() iterations = %d, want at least 2", reads)
	}
	if out.Len() == 0 {
		t.Fatal("stream output is empty")
	}
}

func TestListPresetsJSON(t *testing.T) {
	presets, err := ListPresetsJSON()
	if err != nil {
		t.Fatalf("ListPresetsJSON() error = %v", err)
	}
	if presets == "" {
		t.Skip("embedded presets are unavailable in this build")
	}

	names := decodeJSONText[[]string](t, presets)
	if len(names) == 0 {
		t.Skip("embedded presets are unavailable in this build")
	}
}

func TestGetPresetDetailJSON(t *testing.T) {
	preset := firstPresetName(t)

	detailJSON, err := GetPresetDetailJSON(preset)
	if err != nil {
		t.Fatalf("GetPresetDetailJSON() error = %v", err)
	}
	if detailJSON == "" {
		t.Fatal("GetPresetDetailJSON() returned empty string")
	}

	detail := decodeJSONText[presetDetailResponse](t, detailJSON)
	if detail.Name != preset {
		t.Fatalf("detail.Name = %q, want %q", detail.Name, preset)
	}
}

func TestFeatureSupported(t *testing.T) {
	if !FeatureSupported("streaming") {
		t.Fatal("FeatureSupported(streaming) = false, want true")
	}
	if FeatureSupported("__definitely_missing_feature__") {
		t.Fatal("FeatureSupported(__definitely_missing_feature__) = true, want false")
	}

	client, err := NewClient("chrome_144")
	if err != nil {
		t.Fatalf("NewClient(chrome_144) error = %v", err)
	}
	t.Cleanup(client.Close)

	info, err := client.FingerprintInfoJSON()
	if err != nil {
		t.Fatalf("FingerprintInfoJSON() error = %v", err)
	}

	var payload struct {
		Quic json.RawMessage `json:"quic"`
	}
	if err := json.Unmarshal([]byte(info), &payload); err != nil {
		t.Fatalf("json.Unmarshal(FingerprintInfoJSON()) error = %v", err)
	}

	wantQUICH3 := false
	if raw := strings.TrimSpace(string(payload.Quic)); raw != "" && raw != "null" {
		wantQUICH3 = true
	}
	if got := FeatureSupported("quic-h3"); got != wantQUICH3 {
		t.Fatalf("FeatureSupported(quic-h3) = %v, want %v", got, wantQUICH3)
	}
}

func TestNilHandleSafety(t *testing.T) {
	var client *Client
	if clone := client.Clone(); clone != nil {
		t.Fatalf("nil Client.Clone() = %#v, want nil", clone)
	}
	if _, err := client.FingerprintInfoJSON(); !errors.Is(err, ErrNilHandle) {
		t.Fatalf("nil Client.FingerprintInfoJSON() error = %v, want %v", err, ErrNilHandle)
	}
	client.Close()

	var session *Session
	if clone := session.Clone(); clone != nil {
		t.Fatalf("nil Session.Clone() = %#v, want nil", clone)
	}
	session.Close()

	if _, err := NewSession(nil); !errors.Is(err, ErrNilHandle) {
		t.Fatalf("NewSession(nil) error = %v, want %v", err, ErrNilHandle)
	}
	if _, err := NewSessionWithConfig(nil, "", 0); !errors.Is(err, ErrNilHandle) {
		t.Fatalf("NewSessionWithConfig(nil) error = %v, want %v", err, ErrNilHandle)
	}
	if _, err := NewRequest(nil, "GET", "https://example.com"); !errors.Is(err, ErrNilHandle) {
		t.Fatalf("NewRequest(nil) error = %v, want %v", err, ErrNilHandle)
	}

	var req *Request
	if clone, err := req.Clone(); !errors.Is(err, ErrNilHandle) || clone != nil {
		t.Fatalf("nil Request.Clone() = (%v, %v), want (nil, %v)", clone, err, ErrNilHandle)
	}
	if _, err := req.Send(); !errors.Is(err, ErrNilHandle) {
		t.Fatalf("nil Request.Send() error = %v, want %v", err, ErrNilHandle)
	}
	if _, err := req.SendWithContext(context.Background()); !errors.Is(err, ErrNilHandle) {
		t.Fatalf("nil Request.SendWithContext() error = %v, want %v", err, ErrNilHandle)
	}
	if _, err := req.SendStreaming(); !errors.Is(err, ErrNilHandle) {
		t.Fatalf("nil Request.SendStreaming() error = %v, want %v", err, ErrNilHandle)
	}
	if _, err := req.SendStreamingWithContext(context.Background()); !errors.Is(err, ErrNilHandle) {
		t.Fatalf("nil Request.SendStreamingWithContext() error = %v, want %v", err, ErrNilHandle)
	}
	respCh, errCh := req.SendAsync(context.Background())
	if err := awaitAsyncError[*Response](t, respCh, errCh); !errors.Is(err, ErrNilHandle) {
		t.Fatalf("nil Request.SendAsync() error = %v, want %v", err, ErrNilHandle)
	}
	streamCh, streamErrCh := req.SendStreamingAsync(context.Background())
	if err := awaitAsyncError[*StreamingResponse](t, streamCh, streamErrCh); !errors.Is(err, ErrNilHandle) {
		t.Fatalf("nil Request.SendStreamingAsync() error = %v, want %v", err, ErrNilHandle)
	}

	var resp *Response
	if got := resp.StatusCode(); got != 0 {
		t.Fatalf("nil Response.StatusCode() = %d, want 0", got)
	}
	if got := resp.URL(); got != "" {
		t.Fatalf("nil Response.URL() = %q, want empty string", got)
	}
	if got := len(resp.Bytes()); got != 0 {
		t.Fatalf("len(nil Response.Bytes()) = %d, want 0", got)
	}
	resp.Close()

	var stream *StreamingResponse
	buf := make([]byte, 4)
	if n, err := stream.Read(buf); n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("nil StreamingResponse.Read() = (%d, %v), want (0, %v)", n, err, io.EOF)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("nil StreamingResponse.Close() error = %v", err)
	}

	var lkErr *LkError
	if got := lkErr.Error(); got != ErrNilHandle.Error() {
		t.Fatalf("nil LkError.Error() = %q, want %q", got, ErrNilHandle.Error())
	}
}

func TestSugarPostJSON(t *testing.T) {
	requireNetworkTests(t)

	resp, err := PostJSON("https://httpbin.org/post", `{"source":"sugar"}`)
	if err != nil {
		t.Fatalf("PostJSON() error = %v", err)
	}
	cleanupResponse(t, resp)

	payload := decodeResponseJSON[struct {
		JSON map[string]any `json:"json"`
	}](t, resp)
	if got := payload.JSON["source"]; got != "sugar" {
		t.Fatalf("payload.JSON[source] = %#v, want %q", got, "sugar")
	}
}

func requireNetworkTests(t testing.TB) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping network integration tests in short mode")
	}
	if os.Getenv("LKREQUEST_RUN_NETWORK_TESTS") == "" {
		t.Skip("set LKREQUEST_RUN_NETWORK_TESTS=1 to run network integration tests")
	}
}

func newTestSession(t testing.TB) (*Client, *Session) {
	t.Helper()

	var (
		client  *Client
		session *Session
		err     error
	)

	for attempt := 1; attempt <= testNetworkAttempts; attempt++ {
		client, err = NewDefaultClient()
		if err != nil {
			if attempt == testNetworkAttempts {
				t.Fatalf("NewDefaultClient() error = %v", err)
			}
			time.Sleep(testRetryDelay)
			continue
		}

		session, err = NewSession(client)
		if err == nil {
			break
		}

		client.Close()
		if attempt == testNetworkAttempts {
			t.Fatalf("NewSession() error = %v", err)
		}
		time.Sleep(testRetryDelay)
	}

	t.Cleanup(client.Close)
	t.Cleanup(session.Close)
	return client, session
}

type httpbinAnythingResponse struct {
	Args    map[string]string `json:"args"`
	Data    string            `json:"data"`
	Form    map[string]string `json:"form"`
	Headers map[string]string `json:"headers"`
	Method  string            `json:"method"`
	URL     string            `json:"url"`
}

type httpbinCookiesResponse struct {
	Cookies map[string]string `json:"cookies"`
}

type httpbinBasicAuthResponse struct {
	Authenticated bool   `json:"authenticated"`
	User          string `json:"user"`
}

type httpbinBearerResponse struct {
	Authenticated bool   `json:"authenticated"`
	Token         string `json:"token"`
}

type presetDetailResponse struct {
	Name string `json:"name"`
}

func mustNewRequest(t testing.TB, session *Session, method, rawURL string) *Request {
	t.Helper()

	var (
		req *Request
		err error
	)

	for attempt := 1; attempt <= testNetworkAttempts; attempt++ {
		req, err = NewRequest(session, method, rawURL)
		if err == nil {
			return req
		}
		if attempt < testNetworkAttempts {
			time.Sleep(testRetryDelay)
		}
	}

	t.Fatalf("NewRequest() error = %v", err)
	return nil
}

func mustSendRequest(t testing.TB, req *Request) *Response {
	t.Helper()

	var (
		resp       *Response
		err        error
		lastStatus int
	)

	current := req
	for attempt := 1; attempt <= testNetworkAttempts; attempt++ {
		if attempt > 1 {
			current, err = req.Clone()
			if err != nil {
				t.Fatalf("Clone() error = %v", err)
			}
		}

		resp, err = current.Send()
		if err == nil {
			lastStatus = resp.StatusCode()
			if lastStatus == 200 {
				return resp
			}
			resp.Close()
		}
		if attempt < testNetworkAttempts {
			time.Sleep(testRetryDelay)
		}
	}

	if lastStatus != 0 {
		t.Fatalf("Send() status = %d, want 200", lastStatus)
	}
	t.Fatalf("Send() error = %v", err)
	return nil
}

func cleanupResponse(t testing.TB, resp *Response) {
	t.Helper()

	if resp != nil {
		t.Cleanup(resp.Close)
	}
}

func cleanupStream(t testing.TB, stream *StreamingResponse) {
	t.Helper()

	if stream != nil {
		t.Cleanup(func() {
			if err := stream.Close(); err != nil {
				t.Errorf("Close() error = %v", err)
			}
		})
	}
}

func decodeResponseJSON[T any](t testing.TB, resp *Response) T {
	t.Helper()

	var payload T
	if err := resp.UnmarshalJSON(&payload); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	return payload
}

func decodeJSONText[T any](t testing.TB, text string) T {
	t.Helper()

	var payload T
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return payload
}

func firstPresetName(t testing.TB) string {
	t.Helper()

	presets, err := ListPresetsJSON()
	if err != nil {
		t.Fatalf("ListPresetsJSON() error = %v", err)
	}
	if presets == "" {
		t.Skip("embedded presets are unavailable in this build")
	}

	names := decodeJSONText[[]string](t, presets)
	if len(names) == 0 {
		t.Skip("embedded presets are unavailable in this build")
	}

	return names[0]
}

func awaitAsyncResponse(t testing.TB, respCh <-chan *Response, errCh <-chan error) *Response {
	t.Helper()

	timeout := time.After(15 * time.Second)
	for respCh != nil || errCh != nil {
		select {
		case resp, ok := <-respCh:
			if !ok {
				respCh = nil
				continue
			}
			if resp == nil {
				t.Fatal("async response channel returned nil response")
			}
			return resp
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			t.Fatalf("async error = %v", err)
		case <-timeout:
			t.Fatal("timed out waiting for async response")
		}
	}

	t.Fatal("async channels closed without response")
	return nil
}

func awaitAsyncStream(t testing.TB, streamCh <-chan *StreamingResponse, errCh <-chan error) *StreamingResponse {
	t.Helper()

	timeout := time.After(15 * time.Second)
	for streamCh != nil || errCh != nil {
		select {
		case stream, ok := <-streamCh:
			if !ok {
				streamCh = nil
				continue
			}
			if stream == nil {
				t.Fatal("async stream channel returned nil stream")
			}
			return stream
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			t.Fatalf("async error = %v", err)
		case <-timeout:
			t.Fatal("timed out waiting for async stream")
		}
	}

	t.Fatal("async channels closed without stream")
	return nil
}

func awaitAsyncError[T any](t testing.TB, valueCh <-chan T, errCh <-chan error) error {
	t.Helper()

	timeout := time.After(15 * time.Second)
	for valueCh != nil || errCh != nil {
		select {
		case _, ok := <-valueCh:
			if !ok {
				valueCh = nil
				continue
			}
			t.Fatal("async value channel returned a value, want error")
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			if err == nil {
				t.Fatal("async error channel returned nil error")
			}
			return err
		case <-timeout:
			t.Fatal("timed out waiting for async error")
		}
	}

	t.Fatal("async channels closed without error")
	return nil
}

func TestDnsConfigString(t *testing.T) {
	tests := []struct {
		input DnsConfig
		want  string
	}{
		{DnsSystem, "DnsSystem"},
		{DnsGoogle, "DnsGoogle"},
		{DnsGoogleHTTPS, "DnsGoogleHTTPS"},
		{DnsCloudflare, "DnsCloudflare"},
		{DnsCloudflareHTTPS, "DnsCloudflareHTTPS"},
		{DnsQuad9, "DnsQuad9"},
		{DnsQuad9HTTPS, "DnsQuad9HTTPS"},
		{DnsConfig(99), "DnsConfig(99)"},
	}
	for _, tt := range tests {
		if got := tt.input.String(); got != tt.want {
			t.Errorf("DnsConfig(%d).String() = %q, want %q", int32(tt.input), got, tt.want)
		}
	}
}

func TestRotationStrategyString(t *testing.T) {
	tests := []struct {
		input RotationStrategy
		want  string
	}{
		{RotationRoundRobin, "RotationRoundRobin"},
		{RotationRandom, "RotationRandom"},
		{RotationStrategy(99), "RotationStrategy(99)"},
	}
	for _, tt := range tests {
		if got := tt.input.String(); got != tt.want {
			t.Errorf("RotationStrategy(%d).String() = %q, want %q", int32(tt.input), got, tt.want)
		}
	}
}

func TestMultipartNilSafety(t *testing.T) {
	var mp *Multipart
	mp.AddText("key", "value")
	mp.AddFile("file", "test.txt", "text/plain", []byte("hello"))
	mp.Close()
}

func TestClientBuilderDNS(t *testing.T) {
	b := NewClientBuilder()
	b.SetDNS(DnsGoogle).
		SetDNSCustom("8.8.8.8:53").
		SetUseNativeCerts(true)
	client, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	defer client.Close()
}

func TestResponseCookieMethods(t *testing.T) {
	var r *Response
	if r.CookieCount() != 0 {
		t.Error("nil Response CookieCount should return 0")
	}
	n, v := r.CookieAt(0)
	if n != "" || v != "" {
		t.Error("nil Response CookieAt should return empty")
	}
}

func TestResponseRedirectMethods(t *testing.T) {
	var r *Response
	if r.WasRedirected() {
		t.Error("nil Response WasRedirected should return false")
	}
	if r.RedirectCount() != 0 {
		t.Error("nil Response RedirectCount should return 0")
	}
	url, status := r.RedirectAt(0)
	if url != "" || status != 0 {
		t.Error("nil Response RedirectAt should return empty/0")
	}
}

func TestResponseTextNilSafety(t *testing.T) {
	var r *Response
	_, err := r.Text()
	if err == nil {
		t.Error("nil Response Text should return error")
	}
}

func TestResponseErrorForStatusNilSafety(t *testing.T) {
	var r *Response
	err := r.ErrorForStatus()
	if err == nil {
		t.Error("nil Response ErrorForStatus should return error")
	}
}

func TestStreamingResponseDiagnosticsNilSafety(t *testing.T) {
	var s *StreamingResponse
	if got := s.DiagnosticsJSON(); got != "" {
		t.Errorf("nil StreamingResponse DiagnosticsJSON() = %q, want empty", got)
	}
	if got := s.Header("content-type"); got != "" {
		t.Errorf("nil StreamingResponse Header() = %q, want empty", got)
	}
}

func TestSessionCookieMethods(t *testing.T) {
	var s *Session
	if err := s.SetCookie("https://example.com", "name", "value"); err == nil {
		t.Error("nil Session SetCookie should return error")
	}
	if err := s.SetCookieWithAttrs("https://example.com", "name", "value", CookieAttrs{Path: "/app"}); err == nil {
		t.Error("nil Session SetCookieWithAttrs should return error")
	}
	if err := s.RemoveCookie("https://example.com", "name"); err == nil {
		t.Error("nil Session RemoveCookie should return error")
	}
	if _, err := s.GetCookie("https://example.com", "name"); err == nil {
		t.Error("nil Session GetCookie should return error")
	}
	if _, err := s.GetCookiesJSON("https://example.com"); err == nil {
		t.Error("nil Session GetCookiesJSON should return error")
	}
	if err := s.ClearCookies(); err == nil {
		t.Error("nil Session ClearCookies should return error")
	}
}

func TestSessionSetCookieWithAttrs(t *testing.T) {
	client, session := newTestSession(t)
	t.Cleanup(client.Close)
	t.Cleanup(session.Close)

	url := "https://example.com/app/page"
	err := session.SetCookieWithAttrs(url, "scoped", "cookie", CookieAttrs{
		Path:     "/app",
		Domain:   "example.com",
		Secure:   true,
		HTTPOnly: true,
	})
	if err != nil {
		t.Fatalf("SetCookieWithAttrs() error = %v", err)
	}

	got, err := session.GetCookie(url, "scoped")
	if err != nil {
		t.Fatalf("GetCookie() error = %v", err)
	}
	if got != "cookie" {
		t.Fatalf("GetCookie() = %q, want %q", got, "cookie")
	}

	if _, err := session.GetCookie("https://example.com/other", "scoped"); err == nil {
		t.Fatal("GetCookie() on unmatched path returned nil error, want miss")
	}
}

func TestSessionPreconnectNilSafety(t *testing.T) {
	var s *Session
	if err := s.Preconnect("https://example.com"); err == nil {
		t.Error("nil Session Preconnect should return error")
	}
}

func TestSessionConnectionPoolNilSafety(t *testing.T) {
	var s *Session
	if _, err := s.ConnectionPoolStats(); err == nil {
		t.Error("nil Session ConnectionPoolStats should return error")
	}
	if err := s.ConnectionPoolClear(); err == nil {
		t.Error("nil Session ConnectionPoolClear should return error")
	}
}

func TestProxyGuardNilSafety(t *testing.T) {
	var g *ProxyGuard
	if g.URL() != "" {
		t.Error("nil ProxyGuard URL should return empty")
	}
	if err := g.MarkBad(); err == nil {
		t.Error("nil ProxyGuard MarkBad should return error")
	}
	g.Close()
}

func TestSessionPoolGuardNilSafety(t *testing.T) {
	var g *SessionPoolGuard
	if _, err := g.NewRequest("GET", "https://example.com"); err == nil {
		t.Error("nil SessionPoolGuard NewRequest should return error")
	}
	g.Close()
}
