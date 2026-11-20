package lkrequest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestTLSSessionCacheResumesSessions checks the session cache against a local
// TLS server that closes every connection, so that each request makes a new
// handshake. By default the second handshake resumes the first one's session;
// with the cache off or a TTL of zero it does not. ResetTLSSessionCacheTTL
// undoes a zero TTL, but does not turn a disabled cache back on.
func TestTLSSessionCacheResumesSessions(t *testing.T) {
	requireSessionCacheAPI(t)
	url := resumptionServer(t)

	for _, tc := range []struct {
		name        string
		builder     *ClientBuilder
		wantResumed bool
	}{
		{"default", NewClientBuilder(), true},
		{"disabled", NewClientBuilder().SetTLSSessionCacheEnabled(false), false},
		{"zero ttl", NewClientBuilder().SetTLSSessionCacheTTL(0), false},
		{"zero ttl reset", NewClientBuilder().SetTLSSessionCacheTTL(0).ResetTLSSessionCacheTTL(), true},
		{"disabled, ttl reset", NewClientBuilder().SetTLSSessionCacheEnabled(false).ResetTLSSessionCacheTTL(), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			session := buildTestSession(t, tc.builder.SetVerify(false))
			if got := handshakeKind(t, session, url); got != "full" {
				t.Fatalf("first handshake = %q, want %q", got, "full")
			}
			want := "full"
			if tc.wantResumed {
				want = "resumed"
			}
			if got := handshakeKind(t, session, url); got != want {
				t.Errorf("second handshake = %q, want %q", got, want)
			}
		})
	}
}

// resumptionServer starts an HTTP/1.1 TLS server that answers whether the
// handshake of a request resumed a session, "resumed" or "full", and closes
// the connection after every response. It returns the URL to request.
func resumptionServer(t *testing.T) string {
	t.Helper()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		if r.TLS.DidResume {
			_, _ = io.WriteString(w, "resumed")
		} else {
			_, _ = io.WriteString(w, "full")
		}
	}))
	t.Cleanup(server.Close)
	return server.URL + "/"
}

// handshakeKind requests url and returns what resumptionServer reports of the
// request's handshake.
func handshakeKind(t *testing.T, session *Session, url string) string {
	t.Helper()

	resp, err := session.Get(url).SetTimeout(5000).Send()
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	defer resp.Close()
	return string(resp.Body())
}

// requireSessionCacheAPI skips the test on a library without session cache
// settings.
func requireSessionCacheAPI(t *testing.T) {
	t.Helper()
	if ffi_lk_client_builder_set_tls_session_cache_enabled == nil {
		t.Skip("loaded lkrequest library predates TLS session cache settings")
	}
}
