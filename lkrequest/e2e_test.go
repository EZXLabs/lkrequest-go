package lkrequest

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestE2EFullLifecycle(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req := mustNewRequest(t, session, "GET", "https://httpbin.org/get")

	resp := mustSendRequest(t, req)
	cleanupResponse(t, resp)

	if got := resp.StatusCode(); got != 200 {
		t.Fatalf("StatusCode() = %d, want 200", got)
	}
	if got := resp.URL(); got != "https://httpbin.org/get" {
		t.Fatalf("URL() = %q, want %q", got, "https://httpbin.org/get")
	}
	if body := resp.Bytes(); len(body) == 0 {
		t.Fatal("Bytes() returned empty body")
	}
	if headers := resp.Headers(); headers.Get("Content-Type") == "" {
		t.Fatal("Headers().Get(Content-Type) returned empty string")
	}
}

func TestE2EMultipleMethods(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)

	cases := []struct {
		name   string
		method string
		body   string
	}{
		{name: "get", method: "GET"},
		{name: "post", method: "POST", body: "post-body"},
		{name: "put", method: "PUT", body: "put-body"},
		{name: "delete", method: "DELETE", body: "delete-body"},
		{name: "patch", method: "PATCH", body: "patch-body"},
		{name: "head", method: "HEAD"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := mustNewRequest(t, session, tc.method, "https://httpbin.org/anything")
			if tc.body != "" {
				req = req.SetTextBody(tc.body)
			}

			resp := mustSendRequest(t, req)
			cleanupResponse(t, resp)

			if got := resp.StatusCode(); got != 200 {
				t.Fatalf("StatusCode() = %d, want 200", got)
			}
			if tc.method == "HEAD" {
				return
			}

			payload := decodeResponseJSON[httpbinAnythingResponse](t, resp)
			if got := payload.Method; got != tc.method {
				t.Fatalf("payload.Method = %q, want %q", got, tc.method)
			}
			if tc.body != "" && payload.Data != tc.body {
				t.Fatalf("payload.Data = %q, want %q", payload.Data, tc.body)
			}
		})
	}
}

func TestE2EConcurrentRequests(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)

	requests := make([]*Request, 0, 10)
	for i := 0; i < 10; i++ {
		requests = append(requests, mustNewRequest(t, session, "GET", "https://httpbin.org/get"))
	}

	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for _, req := range requests {
		wg.Add(1)
		go func(req *Request) {
			defer wg.Done()

			current := req
			for attempt := 1; attempt <= testNetworkAttempts; attempt++ {
				if attempt > 1 {
					var err error
					current, err = req.Clone()
					if err != nil {
						errs <- err
						return
					}
				}

				resp, err := current.Send()
				if err != nil {
					if attempt == testNetworkAttempts {
						errs <- err
					}
					continue
				}

				if resp.StatusCode() != 200 {
					resp.Close()
					if attempt == testNetworkAttempts {
						errs <- fmt.Errorf("StatusCode() = %d, want 200", resp.StatusCode())
					}
					continue
				}

				resp.Close()
				return
			}
		}(req)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatal(err)
	}
}

func TestE2ERedirectFollow(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	resp := mustSendRequest(t, mustNewRequest(t, session, "GET", "https://httpbin.org/redirect/3"))
	cleanupResponse(t, resp)

	if got := resp.StatusCode(); got != 200 {
		t.Fatalf("StatusCode() = %d, want 200", got)
	}
	if got := resp.URL(); got != "https://httpbin.org/get" {
		t.Fatalf("URL() = %q, want %q", got, "https://httpbin.org/get")
	}
}

func TestE2ELargeBody(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	resp := mustSendRequest(t, mustNewRequest(t, session, "GET", "https://httpbin.org/bytes/65536"))
	cleanupResponse(t, resp)

	if got := len(resp.Bytes()); got != 65536 {
		t.Fatalf("len(Bytes()) = %d, want 65536", got)
	}
}

func TestE2EResponseDiagnostics(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	resp := mustSendRequest(t, mustNewRequest(t, session, "GET", "https://httpbin.org/get"))
	cleanupResponse(t, resp)

	diag := resp.DiagnosticsJSON()
	if diag == "" {
		t.Fatal("DiagnosticsJSON() returned empty string")
	}

	payload := decodeJSONText[map[string]any](t, diag)
	if _, ok := payload["schema_version"]; !ok {
		t.Fatalf("DiagnosticsJSON() = %q, want schema_version field", diag)
	}
}

func TestE2EStreamToFile(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	stream, err := mustNewRequest(
		t,
		session,
		"GET",
		"https://httpbin.org/stream-bytes/16384?chunk_size=1024",
	).SendStreaming()
	if err != nil {
		t.Fatalf("SendStreaming() error = %v", err)
	}
	cleanupStream(t, stream)

	path := filepath.Join(t.TempDir(), "stream.bin")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("os.Create() error = %v", err)
	}

	copied, err := io.Copy(file, stream)
	if err != nil {
		file.Close()
		t.Fatalf("io.Copy() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("file.Close() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("streamed file is empty")
	}
	if info.Size() != copied {
		t.Fatalf("file size = %d, want %d", info.Size(), copied)
	}
}

func TestE2ECustomTimeout(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req := mustNewRequest(t, session, "GET", "https://httpbin.org/delay/3")

	resp, err := req.SetTimeout(50).Send()
	if resp != nil {
		resp.Close()
	}
	if err == nil {
		t.Fatal("Send() error = nil, want timeout error")
	}

	var lkErr *LkError
	if !errors.As(err, &lkErr) {
		t.Fatalf("Send() error = %T %v, want *LkError", err, err)
	}
	if got := lkErr.Code(); got != ErrTimeout {
		t.Fatalf("LkError.Code() = %s, want %s", got, ErrTimeout)
	}
}

func TestE2ESessionWithProxy(t *testing.T) {
	requireNetworkTests(t)

	proxyURL := os.Getenv("LKREQUEST_TEST_PROXY_URL")
	if proxyURL == "" {
		t.Skip("set LKREQUEST_TEST_PROXY_URL to run proxy integration test")
	}

	client, err := NewDefaultClient()
	if err != nil {
		t.Fatalf("NewDefaultClient() error = %v", err)
	}
	t.Cleanup(client.Close)

	session, err := NewSessionBuilder(client).
		SetProxy(proxyURL).
		SetMaxRedirects(3).
		Build()
	if err != nil {
		t.Fatalf("SessionBuilder.Build() error = %v", err)
	}
	t.Cleanup(session.Close)

	resp := mustSendRequest(t, mustNewRequest(t, session, "GET", "https://httpbin.org/get"))
	cleanupResponse(t, resp)

	if got := resp.StatusCode(); got != 200 {
		t.Fatalf("StatusCode() = %d, want 200", got)
	}
}

func TestE2ERequestReuse(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)
	req := mustNewRequest(t, session, "POST", "https://httpbin.org/anything/reuse")
	req = req.
		AddHeader("X-Reuse", "true").
		AddQuery("case", "reuse").
		SetBodyBytes([]byte("reuse-body"))

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
		if payload.Data != "reuse-body" {
			t.Fatalf("payload.Data = %q, want %q", payload.Data, "reuse-body")
		}
		if payload.Headers["X-Reuse"] != "true" {
			t.Fatalf("payload.Headers[X-Reuse] = %q, want %q", payload.Headers["X-Reuse"], "true")
		}
		if !strings.Contains(payload.URL, "case=reuse") {
			t.Fatalf("payload.URL = %q, want case=reuse", payload.URL)
		}
	}
}

func TestE2EPostLargeBody(t *testing.T) {
	requireNetworkTests(t)

	_, session := newTestSession(t)

	sizes := []struct {
		name string
		size int
	}{
		{"256KB", 256 * 1024},
		{"1MB", 1024 * 1024},
		{"4MB", 4 * 1024 * 1024},
	}

	for _, tc := range sizes {
		t.Run(tc.name+"_bytes", func(t *testing.T) {
			body := make([]byte, tc.size)
			for i := range body {
				body[i] = byte('A' + i%26)
			}

			req := mustNewRequest(t, session, "POST", "https://httpbin.org/post")
			req.AddHeader("content-type", "application/octet-stream")
			resp := mustSendRequest(t, req.SetBodyBytes(body))
			cleanupResponse(t, resp)

			payload := decodeResponseJSON[httpbinAnythingResponse](t, resp)
			if len(payload.Data) != tc.size {
				t.Fatalf("echoed body length = %d, want %d", len(payload.Data), tc.size)
			}
		})

		t.Run(tc.name+"_text", func(t *testing.T) {
			body := strings.Repeat("abcdefghij", tc.size/10)

			req := mustNewRequest(t, session, "POST", "https://httpbin.org/post")
			resp := mustSendRequest(t, req.SetTextBody(body))
			cleanupResponse(t, resp)

			payload := decodeResponseJSON[httpbinAnythingResponse](t, resp)
			if len(payload.Data) != len(body) {
				t.Fatalf("echoed body length = %d, want %d", len(payload.Data), len(body))
			}
		})
	}

	t.Run("1MB_json", func(t *testing.T) {
		filler := strings.Repeat("x", 1024*1024)
		jsonBody := fmt.Sprintf(`{"payload":"%s"}`, filler)

		req := mustNewRequest(t, session, "POST", "https://httpbin.org/post")
		resp := mustSendRequest(t, req.SetJSONBody(jsonBody))
		cleanupResponse(t, resp)

		var raw struct {
			JSON map[string]any `json:"json"`
		}
		if err := resp.UnmarshalJSON(&raw); err != nil {
			t.Fatalf("UnmarshalJSON() error = %v", err)
		}
		got, ok := raw.JSON["payload"].(string)
		if !ok {
			t.Fatal("json.payload is not a string")
		}
		if len(got) != len(filler) {
			t.Fatalf("json.payload length = %d, want %d", len(got), len(filler))
		}
	})
}

func TestE2EBuilderErrors(t *testing.T) {
	requireNetworkTests(t)

	t.Run("client builder invalid preset", func(t *testing.T) {
		client, err := NewClientBuilder().SetPreset("__does_not_exist__").Build()
		if client != nil {
			t.Cleanup(client.Close)
		}
		if err == nil {
			t.Fatal("Build() error = nil, want invalid preset error")
		}
		if !strings.Contains(err.Error(), "client builder set preset") {
			t.Fatalf("Build() error = %q, want client builder set preset", err.Error())
		}
	})

	t.Run("session builder invalid proxy", func(t *testing.T) {
		client, err := NewDefaultClient()
		if err != nil {
			t.Fatalf("NewDefaultClient() error = %v", err)
		}
		t.Cleanup(client.Close)

		session, err := NewSessionBuilder(client).SetProxy("://bad").Build()
		if session != nil {
			t.Cleanup(session.Close)
		}
		if err == nil {
			t.Fatal("Build() error = nil, want invalid proxy error")
		}
		if !strings.Contains(err.Error(), "session builder set proxy") {
			t.Fatalf("Build() error = %q, want session builder set proxy", err.Error())
		}
	})
}
