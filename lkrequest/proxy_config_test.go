package lkrequest

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// newProxyConfig parses url into a ProxyConfig that is closed when the test
// ends, skipping when the loaded library predates proxy configuration handles.
func newProxyConfig(t *testing.T, url string) *ProxyConfig {
	t.Helper()
	if ffi_lk_proxy_config_parse == nil {
		t.Skip("proxy configuration handles are not supported by the loaded lkrequest library")
	}

	config, err := ParseProxyConfig(url)
	if err != nil {
		t.Fatalf("ParseProxyConfig(%q) error = %v", url, err)
	}
	t.Cleanup(config.Close)
	return config
}

// TestProxyConfigURL checks the URL rendering: username/password credentials
// have a URL form, a static HTTP credential does not.
func TestProxyConfigURL(t *testing.T) {
	config := newProxyConfig(t, "http://proxy.example:8080")

	config.SetUserPass("alice", "s3cret")
	if got, err := config.URL(); err != nil || got != "http://YOUR_PROXY_USER:YOUR_PROXY_PASS@proxy.example.com:1080" {
		t.Fatalf("URL() after SetUserPass = %q, %v; want the credentials in the URL", got, err)
	}

	config.SetHTTPAuth("Preshared", "psk-value")
	if got, err := config.URL(); err != nil || got != "http://proxy.example:8080" {
		t.Fatalf("URL() after SetHTTPAuth = %q, %v; want no credentials in the URL", got, err)
	}
}

// TestProxyConfigIdentity pins what makes Identity usable as a blacklist key:
// proxies sharing an address but not credentials get distinct identities, and
// no secret ever appears in one.
func TestProxyConfigIdentity(t *testing.T) {
	const password = "<REDACTED>"

	identity := func(config *ProxyConfig) string {
		t.Helper()
		id, err := config.Identity()
		if err != nil {
			t.Fatalf("Identity() error = %v", err)
		}
		return id
	}

	anonymous := identity(newProxyConfig(t, "http://proxy.example:8080"))
	user := identity(newProxyConfig(t, "http://proxy.example:8080").SetUserPass("alice", password))
	pskA := identity(newProxyConfig(t, "http://proxy.example:8080").SetHTTPAuth("Preshared", "secret-a"))
	pskB := identity(newProxyConfig(t, "http://proxy.example:8080").SetHTTPAuth("Preshared", "secret-b"))

	seen := map[string]string{}
	for name, id := range map[string]string{"anonymous": anonymous, "user": user, "psk-a": pskA, "psk-b": pskB} {
		if other, dup := seen[id]; dup {
			t.Errorf("%s and %s share identity %q; one failing proxy would blacklist the other", name, other, id)
		}
		seen[id] = name
	}

	if !strings.Contains(user, "user:alice") {
		t.Errorf("username identity = %q, want it to name the user", user)
	}
	for _, id := range []string{user, pskA, pskB} {
		for _, secret := range []string{password, "secret-a", "secret-b"} {
			if strings.Contains(id, secret) {
				t.Errorf("identity %q leaks a secret", id)
			}
		}
	}
}

// TestProxyConfigRejectsHTTPAuthOnSOCKS5 checks that a header credential on a
// SOCKS5 proxy, which has no request headers to carry it, is an error rather
// than a credential silently never sent.
func TestProxyConfigRejectsHTTPAuthOnSOCKS5(t *testing.T) {
	config := newProxyConfig(t, "socks5://proxy.example:1080").SetHTTPAuth("Preshared", "psk-value")
	if _, err := config.URL(); err == nil {
		t.Fatal("SetHTTPAuth on a SOCKS5 proxy succeeded; want an error")
	}
}

// TestProxyConfigRejectsUnknownAuthHeader checks the Go-side guard: the
// library takes the header as a Rust enum by value, where an out-of-range value
// would be undefined behavior, so it must never get that far.
func TestProxyConfigRejectsUnknownAuthHeader(t *testing.T) {
	config := newProxyConfig(t, "http://proxy.example:8080").SetAuthHeader(ProxyAuthHeader(2))
	if _, err := config.URL(); err == nil || !strings.Contains(err.Error(), "invalid proxy auth header") {
		t.Fatalf("SetAuthHeader(2) error = %v, want an invalid proxy auth header error", err)
	}
}

// TestProxyConfigErrorsReachTheBuilder checks that an unusable config fails
// the builder it is handed to, instead of the builder quietly going direct.
func TestProxyConfigErrorsReachTheBuilder(t *testing.T) {
	broken := newProxyConfig(t, "socks5://proxy.example:1080").SetHTTPAuth("Preshared", "psk-value")

	client, err := NewClientBuilder().Build()
	if err != nil {
		t.Fatalf("NewClientBuilder().Build() error = %v", err)
	}
	t.Cleanup(client.Close)

	if session, err := NewSessionBuilder(client).SetProxyConfig(broken).Build(); err == nil {
		session.Close()
		t.Fatal("SessionBuilder with a failed ProxyConfig built; want the config's error")
	}

	if _, err := NewProxyPoolBuilder().AddProxyConfig(nil).Build(); !errors.Is(err, ErrNilHandle) {
		t.Fatalf("ProxyPoolBuilder.AddProxyConfig(nil) Build error = %v, want ErrNilHandle", err)
	}
}

// TestProxyConfigCredentialsOnTheWire checks that a static HTTP credential is
// actually sent, in the header SetAuthHeader selects and only there. A local
// proxy double records the CONNECT and refuses it, so nothing leaves the
// machine.
func TestProxyConfigCredentialsOnTheWire(t *testing.T) {
	for _, tc := range []struct {
		name          string
		header        *ProxyAuthHeader
		sentIn, notIn string
	}{
		{name: "default", sentIn: "Proxy-Authorization", notIn: "Authorization"},
		{name: "authorization", header: ptrTo(ProxyAuthHeaderAuthorization), sentIn: "Authorization", notIn: "Proxy-Authorization"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			addr, captured := captureProxyRequest(t)

			config := newProxyConfig(t, "http://"+addr).SetHTTPAuth("Preshared", "psk-value")
			if tc.header != nil {
				config.SetAuthHeader(*tc.header)
			}

			client, err := NewClientBuilder().SetTimeoutTotal(5000).Build()
			if err != nil {
				t.Fatalf("NewClientBuilder().Build() error = %v", err)
			}
			t.Cleanup(client.Close)
			session, err := NewSessionBuilder(client).SetProxyConfig(config).Build()
			if err != nil {
				t.Fatalf("SessionBuilder.SetProxyConfig Build error = %v", err)
			}
			t.Cleanup(session.Close)

			// The double answers 407, so the request is expected to fail.
			if resp, err := session.Get("https://origin.example/").Send(); err == nil {
				resp.Close()
			}

			select {
			case req := <-captured:
				if req.Method != http.MethodConnect {
					t.Fatalf("proxy saw %s, want CONNECT", req.Method)
				}
				if got := req.Header.Get(tc.sentIn); got != "Preshared psk-value" {
					t.Errorf("%s = %q, want %q", tc.sentIn, got, "Preshared psk-value")
				}
				if got := req.Header.Get(tc.notIn); got != "" {
					t.Errorf("%s = %q, want it absent", tc.notIn, got)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("the proxy double received no request")
			}
		})
	}
}

// TestProxyPoolMarkBadByIdentity checks that ProxyPool.MarkBad takes the key
// ProxyConfig.Identity reports. Before Identity existed, Go callers had no way
// to obtain that key; a proxy URL is not it, and marking one is a no-op.
//
// The marked proxy is added by URL, so this also pins that parsing the same
// URL yields the identity the pool tracks it under — the way to mark proxies
// that were never built as a ProxyConfig.
func TestProxyPoolMarkBadByIdentity(t *testing.T) {
	const firstURL = "http://first.example:8080"
	firstID, err := newProxyConfig(t, firstURL).Identity()
	if err != nil {
		t.Fatalf("Identity() error = %v", err)
	}
	second := newProxyConfig(t, "http://second.example:8080")

	pool, err := NewProxyPoolBuilder().
		AddProxy(firstURL).
		AddProxyConfig(second).
		SetBadProxyConfig(1, 60_000, 60_000, 1).
		Build()
	if err != nil {
		t.Fatalf("ProxyPoolBuilder.Build() error = %v", err)
	}
	t.Cleanup(pool.Close)

	acquired := func() map[string]int {
		counts := map[string]int{}
		for i := 0; i < 4; i++ {
			guard, err := pool.Acquire()
			if err != nil {
				t.Fatalf("Acquire() error = %v", err)
			}
			counts[guard.URL()]++
			guard.Close()
		}
		return counts
	}

	if before := acquired(); before[firstURL] == 0 {
		t.Fatalf("before MarkBad the pool never handed out %s: %v", firstURL, before)
	}
	if err := pool.MarkBad(firstID); err != nil {
		t.Fatalf("MarkBad() error = %v", err)
	}

	// The library records the mark on a background task, so the cooldown
	// lands shortly after MarkBad returns rather than before.
	deadline := time.Now().Add(2 * time.Second)
	for {
		after := acquired()
		if after[firstURL] == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("after MarkBad(identity) the pool still hands out %s: %v", firstURL, after)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// captureProxyRequest runs an HTTP proxy double that hands the first request
// it receives to the returned channel and refuses every request with a 407.
func captureProxyRequest(t *testing.T) (string, <-chan *http.Request) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	t.Cleanup(func() { listener.Close() })

	captured := make(chan *http.Request, 1)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
				req, err := http.ReadRequest(bufio.NewReader(conn))
				if err != nil {
					return
				}
				select {
				case captured <- req:
				default:
				}
				_, _ = conn.Write([]byte("HTTP/1.1 407 Proxy Authentication Required\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"))
			}(conn)
		}
	}()
	return listener.Addr().String(), captured
}

func ptrTo[T any](v T) *T {
	return &v
}
