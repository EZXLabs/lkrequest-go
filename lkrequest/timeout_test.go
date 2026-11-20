package lkrequest

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestReadIdleTimeoutStopsAStalledBody checks SetTimeoutReadIdle on a body
// that stalls after its first chunk: buffered and streaming reads time out,
// and DisableTimeoutReadIdle lifts the limit again.
func TestReadIdleTimeoutStopsAStalledBody(t *testing.T) {
	requireTimeoutAPI(t)
	url := stallingServer(t, 0, 600*time.Millisecond)

	stalled := buildTestSession(t, NewClientBuilder().SetTimeoutReadIdle(150))
	if resp, err := stalled.Get(url).Send(); err == nil {
		resp.Close()
		t.Error("buffered Send() succeeded, want a read-idle timeout")
	} else {
		requireTimeoutError(t, "buffered Send()", err)
	}

	stream, err := stalled.Get(url).SendStreaming()
	if err != nil {
		t.Fatalf("SendStreaming() error = %v", err)
	}
	defer stream.Close()
	if _, err := io.ReadAll(stream); err == nil {
		t.Error("reading the streaming body succeeded, want a read-idle timeout")
	} else {
		requireTimeoutError(t, "streaming read", err)
	}

	lifted := buildTestSession(t, NewClientBuilder().SetTimeoutReadIdle(150).DisableTimeoutReadIdle())
	resp, err := lifted.Get(url).Send()
	if err != nil {
		t.Fatalf("Send() with the read-idle timeout disabled error = %v", err)
	}
	defer resp.Close()
	if body := string(resp.Body()); body != stallingBody {
		t.Errorf("body = %q, want %q", body, stallingBody)
	}
}

// TestDeadlinesCanBeDisabled checks the total and the response-header
// deadlines against a server slower than either: each times out, is lifted by
// its Disable setter, and comes back with its Set setter called afterwards.
func TestDeadlinesCanBeDisabled(t *testing.T) {
	requireTimeoutAPI(t)
	url := stallingServer(t, 500*time.Millisecond, 0)

	for _, tc := range []struct {
		name        string
		builder     *ClientBuilder
		wantTimeout bool
	}{
		{"total", NewClientBuilder().SetTimeoutTotal(150), true},
		{"total disabled", NewClientBuilder().SetTimeoutTotal(150).DisableTimeoutTotal(), false},
		{"total set again", NewClientBuilder().DisableTimeoutTotal().SetTimeoutTotal(150), true},
		{"ttfb", NewClientBuilder().SetTimeoutTTFB(150), true},
		{"ttfb disabled", NewClientBuilder().SetTimeoutTTFB(150).DisableTimeoutTTFB(), false},
		{"ttfb set again", NewClientBuilder().DisableTimeoutTTFB().SetTimeoutTTFB(150), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := buildTestSession(t, tc.builder).Get(url).Send()
			if tc.wantTimeout {
				if err == nil {
					resp.Close()
					t.Fatal("Send() succeeded, want a timeout")
				}
				requireTimeoutError(t, "Send()", err)
				return
			}
			if err != nil {
				t.Fatalf("Send() error = %v, want the slow response", err)
			}
			resp.Close()
		})
	}
}

// TestEveryDeadlineCanBeDisabled builds a client with every deadline removed
// and checks it still completes a request.
func TestEveryDeadlineCanBeDisabled(t *testing.T) {
	requireTimeoutAPI(t)
	url := stallingServer(t, 0, 0)

	session := buildTestSession(t, NewClientBuilder().
		DisableTimeoutDNS().
		DisableTimeoutTCPConnect().
		DisableTimeoutTLSHandshake().
		DisableTimeoutTTFB().
		DisableTimeoutTotal().
		DisableTimeoutQUICConnect().
		DisableTimeoutReadIdle())
	resp, err := session.Get(url).Send()
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	defer resp.Close()
	if body := string(resp.Body()); body != stallingBody {
		t.Errorf("body = %q, want %q", body, stallingBody)
	}
}

// stallingBody is what stallingServer sends, in two chunks.
const stallingBody = "first chunk|second chunk"

// stallingServer starts an HTTP server that waits headerDelay before its
// response headers, sends the first chunk of stallingBody, and waits
// bodyStall before the second. It returns the URL to request.
func stallingServer(t *testing.T, headerDelay, bodyStall time.Duration) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sleepUnlessDone(r, headerDelay) {
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, "first chunk|")
		w.(http.Flusher).Flush()
		if !sleepUnlessDone(r, bodyStall) {
			return
		}
		_, _ = io.WriteString(w, "second chunk")
	}))
	t.Cleanup(server.Close)
	return server.URL + "/"
}

// sleepUnlessDone waits d, and reports false when the client goes away first.
func sleepUnlessDone(r *http.Request, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	select {
	case <-time.After(d):
		return true
	case <-r.Context().Done():
		return false
	}
}

// buildTestSession builds builder into a client and returns a session of it,
// both closed when the test ends.
func buildTestSession(t *testing.T, builder *ClientBuilder) *Session {
	t.Helper()

	client, err := builder.Build()
	if err != nil {
		t.Fatalf("ClientBuilder.Build() error = %v", err)
	}
	t.Cleanup(client.Close)
	session, err := NewSessionBuilder(client).Build()
	if err != nil {
		t.Fatalf("SessionBuilder.Build() error = %v", err)
	}
	t.Cleanup(session.Close)
	return session
}

// requireTimeoutError fails the test unless err is an LkError reporting a
// timeout.
func requireTimeoutError(t *testing.T, what string, err error) {
	t.Helper()

	var lkErr *LkError
	if !errors.As(err, &lkErr) || lkErr.Code() != ErrTimeout {
		t.Errorf("%s error = %v (%T), want an LkError with code %v", what, err, err, ErrTimeout)
	}
}

// requireTimeoutAPI skips the test on a library without optional deadlines.
func requireTimeoutAPI(t *testing.T) {
	t.Helper()
	if ffi_lk_client_builder_set_timeout_read_idle == nil || ffi_lk_client_builder_disable_timeout_total == nil {
		t.Skip("loaded lkrequest library predates the read-idle timeout and optional deadlines")
	}
}
