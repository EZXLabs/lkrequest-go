package lkrequest

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

type localEchoResponse struct {
	Method  string              `json:"method"`
	Body    string              `json:"body"`
	Cookies map[string]string   `json:"cookies"`
	Headers map[string][]string `json:"headers"`
	Proto   string              `json:"proto"`
	URL     string              `json:"url"`
}

type localMultipartFile struct {
	ContentType string `json:"content_type"`
	Data        string `json:"data"`
	Filename    string `json:"filename"`
	Size        int    `json:"size"`
}

type localMultipartResponse struct {
	Files map[string]localMultipartFile `json:"files"`
	Form  map[string]string             `json:"form"`
}

type localProxyProvider struct {
	mu        sync.Mutex
	urls      []string
	index     int
	destroyed chan struct{}
}

func newLocalProxyProvider(urls ...string) *localProxyProvider {
	return &localProxyProvider{
		urls:      append([]string(nil), urls...),
		destroyed: make(chan struct{}),
	}
}

func (p *localProxyProvider) NextProxy() (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.urls) == 0 {
		return "", false
	}
	url := p.urls[p.index%len(p.urls)]
	p.index++
	return url, true
}

func (p *localProxyProvider) Len() uint {
	p.mu.Lock()
	defer p.mu.Unlock()
	return uint(len(p.urls))
}

func (p *localProxyProvider) IsDynamic() bool {
	return false
}

func (p *localProxyProvider) Destroy() {
	select {
	case <-p.destroyed:
	default:
		close(p.destroyed)
	}
}

func TestInitLog(t *testing.T) {
	path := t.TempDir() + "/lkrequest.log"
	if err := InitLog("info", path); err != nil {
		t.Fatalf("InitLog() error = %v", err)
	}
}

func TestNewSessionWithConfig(t *testing.T) {
	client, err := newBuilderClient()
	if err != nil {
		t.Fatalf("newBuilderClient() error = %v", err)
	}
	t.Cleanup(client.Close)

	session, err := NewSessionWithConfig(client, "http://127.0.0.1:8080", 2)
	if err != nil {
		t.Fatalf("NewSessionWithConfig() error = %v", err)
	}
	t.Cleanup(session.Close)
}

func TestSessionCookieJarLifecycle(t *testing.T) {
	server := newLocalFeatureServer(t)
	client, session := newLocalSession(t)
	t.Cleanup(client.Close)
	t.Cleanup(session.Close)

	baseURL := server.URL + "/echo"
	if err := session.SetCookie(baseURL, "session", "base-value"); err != nil {
		t.Fatalf("SetCookie() error = %v", err)
	}

	got, err := session.GetCookie(baseURL, "session")
	if err != nil {
		t.Fatalf("GetCookie() error = %v", err)
	}
	if got != "base-value" {
		t.Fatalf("GetCookie() = %q, want %q", got, "base-value")
	}

	if err := session.SetCookieWithAttrs(baseURL, "scoped", "cookie", CookieAttrs{
		Path:     "/echo",
		Domain:   "127.0.0.1",
		HTTPOnly: true,
	}); err != nil {
		t.Fatalf("SetCookieWithAttrs() error = %v", err)
	}

	cookiesJSON, err := session.GetCookiesJSON(baseURL)
	if err != nil {
		t.Fatalf("GetCookiesJSON() error = %v", err)
	}
	if !strings.Contains(cookiesJSON, "session") || !strings.Contains(cookiesJSON, "scoped") {
		t.Fatalf("GetCookiesJSON() = %q, want session and scoped cookies", cookiesJSON)
	}

	if err := session.RemoveCookie(baseURL, "session"); err != nil {
		t.Fatalf("RemoveCookie() error = %v", err)
	}
	if _, err := session.GetCookie(baseURL, "session"); err == nil {
		t.Fatal("GetCookie() after RemoveCookie() error = nil, want miss")
	}

	if err := session.ClearCookies(); err != nil {
		t.Fatalf("ClearCookies() error = %v", err)
	}
	if _, err := session.GetCookie(baseURL, "scoped"); err == nil {
		t.Fatal("GetCookie() after ClearCookies() error = nil, want miss")
	}
}

func TestSessionPreconnectAndConnectionPoolStats(t *testing.T) {
	server := newLocalFeatureServer(t)

	client, err := newBuilderClient()
	if err != nil {
		t.Fatalf("newBuilderClient() error = %v", err)
	}
	t.Cleanup(client.Close)

	session, err := NewSessionBuilder(client).
		SetMaxConnections(2).
		SetIdleTimeout(5_000).
		Build()
	if err != nil {
		t.Fatalf("SessionBuilder.Build() error = %v", err)
	}
	t.Cleanup(session.Close)

	if err := session.Preconnect(server.URL + "/echo"); err != nil {
		t.Fatalf("Preconnect() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := session.PreconnectAsync(ctx, server.URL+"/echo"); err != nil {
		t.Fatalf("PreconnectAsync() error = %v", err)
	}

	stats, err := session.ConnectionPoolStats()
	if err != nil {
		t.Fatalf("ConnectionPoolStats() error = %v", err)
	}
	if stats.Max == 0 {
		t.Fatalf("ConnectionPoolStats().Max = %d, want > 0", stats.Max)
	}

	if err := session.ConnectionPoolClear(); err != nil {
		t.Fatalf("ConnectionPoolClear() error = %v", err)
	}

	stats, err = session.ConnectionPoolStats()
	if err != nil {
		t.Fatalf("ConnectionPoolStats() after clear error = %v", err)
	}
	if stats.Total != 0 {
		t.Fatalf("ConnectionPoolStats().Total after clear = %d, want 0", stats.Total)
	}
	if stats.H3 != 0 {
		t.Fatalf("ConnectionPoolStats().H3 after clear = %d, want 0", stats.H3)
	}
}

func TestSessionBuilderDisableRedirects(t *testing.T) {
	server := newLocalFeatureServer(t)
	client, err := newBuilderClient()
	if err != nil {
		t.Fatalf("newBuilderClient() error = %v", err)
	}
	t.Cleanup(client.Close)

	session, err := NewSessionBuilder(client).
		DisableRedirects().
		Build()
	if err != nil {
		t.Fatalf("SessionBuilder.DisableRedirects().Build() error = %v", err)
	}
	t.Cleanup(session.Close)

	resp, err := mustNewRequest(t, session, "GET", server.URL+"/redirect/one").Send()
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	cleanupResponse(t, resp)

	if got := resp.StatusCode(); got != http.StatusFound {
		t.Fatalf("StatusCode() = %d, want %d", got, http.StatusFound)
	}
	if got := resp.RedirectCount(); got != 0 {
		t.Fatalf("RedirectCount() = %d, want 0", got)
	}
	if resp.WasRedirected() {
		t.Fatal("WasRedirected() = true, want false")
	}
	if got := resp.Header("location"); got != "/redirect/two" {
		t.Fatalf("Header(location) = %q, want %q", got, "/redirect/two")
	}
}

func TestH3SpecificConfigurationMethods(t *testing.T) {
	server := newLocalFeatureServer(t)

	client, err := NewClientBuilder().
		AddH3HeaderOrder("priority").
		Build()
	if err != nil {
		t.Fatalf("ClientBuilder.AddH3HeaderOrder().Build() error = %v", err)
	}
	t.Cleanup(client.Close)

	session, err := NewSessionBuilder(client).
		AddH3HeaderOrder("priority").
		Build()
	if err != nil {
		t.Fatalf("SessionBuilder.AddH3HeaderOrder().Build() error = %v", err)
	}
	t.Cleanup(session.Close)

	req := mustNewRequest(t, session, "GET", server.URL+"/echo")
	resp := mustSendRequest(t, req.AddH3HeaderOrder("priority"))
	cleanupResponse(t, resp)
}

func TestSessionBuilderSetHTTP3OnlyBuilds(t *testing.T) {
	client, err := newBuilderClient()
	if err != nil {
		t.Fatalf("newBuilderClient() error = %v", err)
	}
	t.Cleanup(client.Close)

	session, err := NewSessionBuilder(client).
		SetHTTP3Only().
		Build()
	if err != nil {
		t.Fatalf("SessionBuilder.SetHTTP3Only().Build() error = %v", err)
	}
	t.Cleanup(session.Close)
}

func TestRequestSetCookieOverride(t *testing.T) {
	server := newLocalFeatureServer(t)
	client, session := newLocalSession(t)
	t.Cleanup(client.Close)
	t.Cleanup(session.Close)

	targetURL := server.URL + "/echo"
	if err := session.SetCookie(targetURL, "session", "from-session"); err != nil {
		t.Fatalf("SetCookie() error = %v", err)
	}

	req := mustNewRequest(t, session, "GET", targetURL)
	resp := mustSendRequest(t, req.SetCookieOverride("session", "from-request"))
	cleanupResponse(t, resp)

	payload := decodeResponseJSON[localEchoResponse](t, resp)
	if got := payload.Cookies["session"]; got != "from-request" {
		t.Fatalf("payload.Cookies[session] = %q, want %q", got, "from-request")
	}
}

func TestRequestSetMultipart(t *testing.T) {
	server := newLocalFeatureServer(t)
	client, session := newLocalSession(t)
	t.Cleanup(client.Close)
	t.Cleanup(session.Close)

	req := mustNewRequest(t, session, "POST", server.URL+"/multipart")
	mp := NewMultipart().
		AddText("username", "alice").
		AddFile("avatar", "avatar.txt", "text/plain", []byte("avatar-bytes"))

	resp := mustSendRequest(t, req.SetMultipart(mp))
	cleanupResponse(t, resp)

	payload := decodeResponseJSON[localMultipartResponse](t, resp)
	if got := payload.Form["username"]; got != "alice" {
		t.Fatalf("payload.Form[username] = %q, want %q", got, "alice")
	}

	file, ok := payload.Files["avatar"]
	if !ok {
		t.Fatalf("payload.Files = %#v, want avatar entry", payload.Files)
	}
	if file.Filename != "avatar.txt" {
		t.Fatalf("payload.Files[avatar].Filename = %q, want %q", file.Filename, "avatar.txt")
	}
	if file.ContentType != "text/plain" {
		t.Fatalf("payload.Files[avatar].ContentType = %q, want %q", file.ContentType, "text/plain")
	}
	if file.Data != "avatar-bytes" {
		t.Fatalf("payload.Files[avatar].Data = %q, want %q", file.Data, "avatar-bytes")
	}
}

func TestResponseTextAndErrorForStatus(t *testing.T) {
	server := newLocalFeatureServer(t)
	client, session := newLocalSession(t)
	t.Cleanup(client.Close)
	t.Cleanup(session.Close)

	t.Run("Text", func(t *testing.T) {
		resp := mustSendRequest(t, mustNewRequest(t, session, "GET", server.URL+"/text"))
		cleanupResponse(t, resp)

		text, err := resp.Text()
		if err != nil {
			t.Fatalf("Text() error = %v", err)
		}
		if text != "plain response" {
			t.Fatalf("Text() = %q, want %q", text, "plain response")
		}
	})

	t.Run("ErrorForStatus", func(t *testing.T) {
		req := mustNewRequest(t, session, "GET", server.URL+"/status/418")
		resp, err := req.Send()
		if err != nil {
			t.Fatalf("Send() error = %v", err)
		}
		cleanupResponse(t, resp)

		if got := resp.StatusCode(); got != http.StatusTeapot {
			t.Fatalf("StatusCode() = %d, want %d", got, http.StatusTeapot)
		}

		err = resp.ErrorForStatus()
		if err == nil {
			t.Fatal("ErrorForStatus() error = nil, want *LkError")
		}

		var lkErr *LkError
		if !errors.As(err, &lkErr) {
			t.Fatalf("ErrorForStatus() error = %T %v, want *LkError", err, err)
		}
		if got := lkErr.HttpStatus(); got != http.StatusTeapot {
			t.Fatalf("LkError.HttpStatus() = %d, want %d", got, http.StatusTeapot)
		}
		if got := lkErr.Code(); got == 0 {
			t.Fatal("LkError.Code() = 0, want non-zero error code")
		}
		if msg := lkErr.Error(); msg == "" {
			t.Fatal("LkError.Error() returned empty string")
		}
		if diag := lkErr.DiagnosticsJSON(); diag == "" {
			t.Fatal("LkError.DiagnosticsJSON() returned empty string")
		}
	})
}

func TestResponseCookiesAndRedirects(t *testing.T) {
	server := newLocalFeatureServer(t)
	client, session := newLocalSession(t)
	t.Cleanup(client.Close)
	t.Cleanup(session.Close)

	t.Run("cookies", func(t *testing.T) {
		resp := mustSendRequest(t, mustNewRequest(t, session, "GET", server.URL+"/set-cookies"))
		cleanupResponse(t, resp)

		if got := resp.CookieCount(); got != 2 {
			t.Fatalf("CookieCount() = %d, want 2", got)
		}

		cookies := make(map[string]string, resp.CookieCount())
		for i := 0; i < resp.CookieCount(); i++ {
			name, value := resp.CookieAt(i)
			cookies[name] = value
		}
		if cookies["alpha"] != "one" || cookies["beta"] != "two" {
			t.Fatalf("cookies = %#v, want alpha=one beta=two", cookies)
		}
	})

	t.Run("redirects", func(t *testing.T) {
		resp := mustSendRequest(t, mustNewRequest(t, session, "GET", server.URL+"/redirect/one"))
		cleanupResponse(t, resp)

		if !resp.WasRedirected() {
			t.Fatal("WasRedirected() = false, want true")
		}
		if got := resp.RedirectCount(); got == 0 {
			t.Fatal("RedirectCount() = 0, want > 0")
		}

		for i := 0; i < resp.RedirectCount(); i++ {
			url, status := resp.RedirectAt(i)
			if url == "" {
				t.Fatalf("RedirectAt(%d).URL = %q, want non-empty", i, url)
			}
			if status == 0 {
				t.Fatalf("RedirectAt(%d).Status = %d, want non-zero", i, status)
			}
		}
	})
}

func TestStreamingResponseDiagnosticsAndHeader(t *testing.T) {
	server := newLocalFeatureServer(t)
	client, session := newLocalSession(t)
	t.Cleanup(client.Close)
	t.Cleanup(session.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mustNewRequest(t, session, "GET", server.URL+"/stream")
	stream, err := req.SendStreamingWithContext(ctx)
	if err != nil {
		t.Fatalf("SendStreamingWithContext() error = %v", err)
	}
	cleanupStream(t, stream)

	if got := stream.Header("x-stream"); got != "yes" {
		t.Fatalf("Header(x-stream) = %q, want %q", got, "yes")
	}
	if diag := stream.DiagnosticsJSON(); diag == "" {
		t.Fatal("DiagnosticsJSON() returned empty string")
	}

	body, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if got := string(body); got != "chunk-onechunk-two" {
		t.Fatalf("stream body = %q, want %q", got, "chunk-onechunk-two")
	}
}

func TestProxyPoolAcquireFlows(t *testing.T) {
	proxies := []string{
		"http://proxy-one.example:8080",
		"http://proxy-two.example:8080",
	}

	pool, err := NewProxyPoolBuilder().
		AddProxy(proxies[0]).
		AddProxies(proxies[1:]).
		SetRotation(RotationRoundRobin).
		SetProxyBuffer(2).
		SetBadProxyConfig(1, 100, 250, 1).
		SetMaxProxies(2).
		Build()
	if err != nil {
		t.Fatalf("ProxyPoolBuilder.Build() error = %v", err)
	}
	t.Cleanup(pool.Close)

	if got := pool.MaxConcurrent(); got == 0 {
		t.Fatalf("MaxConcurrent() = %d, want > 0", got)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	asyncGuard, err := pool.AcquireAsync(ctx)
	if err != nil {
		t.Fatalf("AcquireAsync() error = %v", err)
	}
	if asyncGuard.URL() == "" {
		asyncGuard.Close()
		t.Fatal("AcquireAsync().URL() returned empty string")
	}
	asyncGuard.Close()

	guard, err := pool.Acquire()
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	firstURL := guard.URL()
	if firstURL == "" {
		guard.Close()
		t.Fatal("Acquire().URL() returned empty string")
	}

	fresh, err := pool.AcquireFresh(guard)
	if err != nil {
		guard.Close()
		t.Fatalf("AcquireFresh() error = %v", err)
	}

	if got := fresh.URL(); got == "" {
		fresh.Close()
		guard.Close()
		t.Fatal("AcquireFresh().URL() returned empty string")
	} else if got == firstURL {
		fresh.Close()
		guard.Close()
		t.Fatalf("AcquireFresh().URL() = %q, want different proxy", got)
	}

	if err := guard.MarkBad(); err != nil {
		fresh.Close()
		guard.Close()
		t.Fatalf("ProxyGuard.MarkBad() error = %v", err)
	}
	fresh.Close()
	guard.Close()

	if err := pool.MarkBad(firstURL); err != nil {
		t.Fatalf("ProxyPool.MarkBad() error = %v", err)
	}
}

func TestProxyPoolBuilderSetProvider(t *testing.T) {
	if os.Getenv("LKREQUEST_TEST_PROXY_PROVIDER_SUBPROCESS") == "" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestProxyPoolBuilderSetProvider$")
		cmd.Env = append(os.Environ(), "LKREQUEST_TEST_PROXY_PROVIDER_SUBPROCESS=1")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("proxy provider subprocess failed: %v\n%s", err, string(output))
		}
		return
	}

	provider := newLocalProxyProvider(
		"http://provider-one.example:8080",
		"http://provider-two.example:8080",
	)

	pool, err := NewProxyPoolBuilder().
		SetProvider(provider).
		SetMaxProxies(2).
		Build()
	if err != nil {
		t.Fatalf("ProxyPoolBuilder.SetProvider().Build() error = %v", err)
	}

	guard, err := pool.Acquire()
	if err != nil {
		pool.Close()
		t.Fatalf("ProxyPool.Acquire() error = %v", err)
	}
	firstURL := guard.URL()
	if firstURL != "http://provider-one.example:8080" {
		guard.Close()
		pool.Close()
		t.Fatalf("Acquire().URL() = %q, want %q", firstURL, "http://provider-one.example:8080")
	}

	fresh, err := pool.AcquireFresh(guard)
	if err != nil {
		guard.Close()
		pool.Close()
		t.Fatalf("ProxyPool.AcquireFresh() error = %v", err)
	}
	if got := fresh.URL(); got != "http://provider-two.example:8080" {
		fresh.Close()
		guard.Close()
		pool.Close()
		t.Fatalf("AcquireFresh().URL() = %q, want %q", got, "http://provider-two.example:8080")
	}

	fresh.Close()
	guard.Close()
	pool.Close()
}

func TestSessionPoolAcquireFlows(t *testing.T) {
	client, err := newBuilderClient()
	if err != nil {
		t.Fatalf("newBuilderClient() error = %v", err)
	}
	t.Cleanup(client.Close)

	proxies := []string{
		"http://proxy-one.example:8080",
		"http://proxy-two.example:8080",
	}

	pool, err := NewSessionPoolBuilder(client).
		AddProxy(proxies[0]).
		AddProxies(proxies[1:]).
		SetRotation(RotationRoundRobin).
		SetProxyBuffer(2).
		SetMaxSessions(2).
		SetIdleTimeout(2_000).
		SetBadProxyConfig(1, 100, 250, 5).
		Build()
	if err != nil {
		t.Fatalf("SessionPoolBuilder.Build() error = %v", err)
	}
	t.Cleanup(pool.Close)

	stats, err := pool.Stats()
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.Max == 0 {
		t.Fatalf("Stats().Max = %d, want > 0", stats.Max)
	}

	guard, err := pool.Acquire()
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer guard.Close()

	req, err := guard.NewRequest("GET", "http://example.com")
	if err != nil {
		t.Fatalf("SessionPoolGuard.NewRequest() error = %v", err)
	}
	req.release()

	fresh, err := pool.AcquireFresh(guard)
	if err != nil {
		t.Fatalf("AcquireFresh() error = %v", err)
	}
	if err := pool.MarkBad(fresh); err != nil {
		fresh.Close()
		t.Fatalf("MarkBad() error = %v", err)
	}
	fresh.Close()

	// Both proxies are cooling down now. The pool waits for one to recover
	// instead of handing out a direct session, so this completes once the
	// 250 ms cooldown ends. A proxy removed for good (maxCooldowns 1) would
	// leave it waiting until ctx expires.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	asyncGuard, err := pool.AcquireAsync(ctx)
	if err != nil {
		t.Fatalf("AcquireAsync() error = %v", err)
	}
	asyncGuard.Close()
}

func TestSessionPoolBuilderSetProvider(t *testing.T) {
	if os.Getenv("LKREQUEST_TEST_SESSION_PROVIDER_SUBPROCESS") == "" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestSessionPoolBuilderSetProvider$")
		cmd.Env = append(os.Environ(), "LKREQUEST_TEST_SESSION_PROVIDER_SUBPROCESS=1")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("session pool provider subprocess failed: %v\n%s", err, string(output))
		}
		return
	}

	client, err := newBuilderClient()
	if err != nil {
		t.Fatalf("newBuilderClient() error = %v", err)
	}
	t.Cleanup(client.Close)

	provider := newLocalProxyProvider("http://provider-session.example:8080")
	pool, err := NewSessionPoolBuilder(client).
		SetProvider(provider).
		SetMaxSessions(1).
		Build()
	if err != nil {
		t.Fatalf("SessionPoolBuilder.SetProvider().Build() error = %v", err)
	}

	guard, err := pool.Acquire()
	if err != nil {
		pool.Close()
		t.Fatalf("SessionPool.Acquire() error = %v", err)
	}

	req, err := guard.NewRequest("GET", "http://example.com/from-provider")
	if err != nil {
		guard.Close()
		pool.Close()
		t.Fatalf("SessionPoolGuard.NewRequest() error = %v", err)
	}
	req.release()

	guard.Close()
	pool.Close()
}

func newLocalSession(t testing.TB) (*Client, *Session) {
	t.Helper()

	client, err := newBuilderClient()
	if err != nil {
		t.Fatalf("newBuilderClient() error = %v", err)
	}

	session, err := NewSession(client)
	if err != nil {
		client.Close()
		t.Fatalf("NewSession() error = %v", err)
	}

	return client, session
}

func newBuilderClient() (*Client, error) {
	return NewClientBuilder().Build()
}

func newLocalFeatureServer(t testing.TB) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		cookies := make(map[string]string, len(r.Cookies()))
		for _, cookie := range r.Cookies() {
			cookies[cookie.Name] = cookie.Value
		}

		writeTestJSON(w, localEchoResponse{
			Method:  r.Method,
			Body:    string(body),
			Cookies: cookies,
			Headers: r.Header,
			Proto:   r.Proto,
			URL:     r.URL.String(),
		})
	})

	mux.HandleFunc("/gzip", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", "text/plain")

		gz := gzip.NewWriter(w)
		_, _ = gz.Write([]byte("compressed hello"))
		_ = gz.Close()
	})

	mux.HandleFunc("/multipart", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		form := make(map[string]string, len(r.MultipartForm.Value))
		for key, values := range r.MultipartForm.Value {
			if len(values) > 0 {
				form[key] = values[0]
			}
		}

		files := make(map[string]localMultipartFile, len(r.MultipartForm.File))
		for key, headers := range r.MultipartForm.File {
			if len(headers) == 0 {
				continue
			}

			file, err := headers[0].Open()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			data, err := io.ReadAll(file)
			_ = file.Close()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			files[key] = localMultipartFile{
				ContentType: headers[0].Header.Get("Content-Type"),
				Data:        string(data),
				Filename:    headers[0].Filename,
				Size:        len(data),
			}
		}

		writeTestJSON(w, localMultipartResponse{
			Files: files,
			Form:  form,
		})
	})

	mux.HandleFunc("/text", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "plain response")
	})

	mux.HandleFunc("/status/418", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusTeapot)
		_, _ = io.WriteString(w, "short and stout")
	})

	mux.HandleFunc("/set-cookies", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "alpha", Value: "one", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "beta", Value: "two", Path: "/"})
		_, _ = io.WriteString(w, "ok")
	})

	mux.HandleFunc("/redirect/one", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/redirect/two", http.StatusFound)
	})
	mux.HandleFunc("/redirect/two", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/redirect/final", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/redirect/final", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "redirect complete")
	})

	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("X-Stream", "yes")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		for _, chunk := range []string{"chunk-one", "chunk-two"} {
			_, _ = io.WriteString(w, chunk)
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)
		}
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func writeTestJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// TestFFIBoolMasksHighBits pins the fix for a real defect: a C bool-returning
// FFI function only sets the low byte of the return register, and an optimized
// library build can leave stale garbage in the high bits. Reading the whole
// uintptr with `!= 0` misreports a `false` as true. ffiBool must mask to the
// low byte. The high-bits value below is the exact garbage observed from
// lk_response_was_redirected when it returned false.
func TestFFIBoolMasksHighBits(t *testing.T) {
	cases := []struct {
		raw  uintptr
		want bool
	}{
		{0x2ae3c441d00, false}, // low byte 0x00 with dirty high bits -> false
		{0x2ae3c441d01, true},  // low byte 0x01 -> true
		{0, false},
		{1, true},
		{0xFF, true},
		{0xFFFFFFFFFFFFFF00, false}, // all high bits set, low byte 0 -> false
	}
	for _, c := range cases {
		if got := ffiBool(c.raw); got != c.want {
			t.Errorf("ffiBool(0x%x) = %v, want %v", c.raw, got, c.want)
		}
	}
}

// TestRandomizeModeAndLayersConstants pins the Go enum/bitmask values to the
// FFI ABI in include/lkrequest.h (lk_randomize_mode_t and the LK_FP_LAYER_*
// macros). Drift here would silently mis-select a randomization tier or layer.
func TestRandomizeModeAndLayersConstants(t *testing.T) {
	modes := []struct {
		mode RandomizeMode
		val  int32
		str  string
	}{
		{RandomizeOff, 0, "RandomizeOff"},
		{RandomizeExtensionOrder, 1, "RandomizeExtensionOrder"},
		{RandomizeRecombine, 2, "RandomizeRecombine"},
		{RandomizeFull, 3, "RandomizeFull"},
	}
	for _, c := range modes {
		if int32(c.mode) != c.val {
			t.Errorf("%s = %d, want %d", c.str, int32(c.mode), c.val)
		}
		if got := c.mode.String(); got != c.str {
			t.Errorf("RandomizeMode(%d).String() = %q, want %q", c.val, got, c.str)
		}
	}
	if got := RandomizeMode(99).String(); got != "RandomizeMode(99)" {
		t.Errorf("unknown RandomizeMode.String() = %q", got)
	}

	if FingerprintLayersAll != 0 {
		t.Errorf("FingerprintLayersAll = %d, want 0", FingerprintLayersAll)
	}
	if LayerTLS != 1 || LayerH2 != 2 || LayerQUIC != 4 {
		t.Errorf("FingerprintLayers ABI drift: tls=%d h2=%d quic=%d, want 1/2/4",
			LayerTLS, LayerH2, LayerQUIC)
	}
}

// TestClientBuilderSetRandomizeBuilds exercises SetRandomize end-to-end. It
// skips when the loaded library predates the lk_client_builder_set_randomize
// export (e.g. an older embedded DLL), and tolerates synthetic modes being
// absent when the library lacks the synthetic-fp build feature.
func TestClientBuilderSetRandomizeBuilds(t *testing.T) {
	if ffi_lk_client_builder_set_randomize == nil {
		t.Skip("lk_client_builder_set_randomize absent from the loaded lkrequest library")
	}

	// Tier 0/1 are always available once the symbol is exported.
	for _, mode := range []RandomizeMode{RandomizeOff, RandomizeExtensionOrder} {
		client, err := NewClientBuilder().
			SetRandomize(mode, FingerprintLayersAll).
			Build()
		if err != nil {
			t.Fatalf("SetRandomize(%s).Build() error = %v", mode, err)
		}
		client.Close()
	}

	// Synthetic modes need a synthetic-fp build; either they build, or the
	// setter returns a clean error (must not panic or nil-deref).
	client, err := NewClientBuilder().
		SetRandomize(RandomizeRecombine, LayerTLS|LayerH2).
		Build()
	if err != nil {
		t.Logf("RandomizeRecombine unavailable (synthetic-fp not compiled in): %v", err)
		return
	}
	client.Close()
}
