package lkrequest

import (
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestRequestHandleFreedAfterSend guards against re-introducing a leak of the
// request handle. The FFI send functions only borrow the handle, so a detached
// handle that is never freed keeps the RequestHandle alive along with the
// client and session Arcs it holds, which prevents the native session from
// being dropped even after Session.Close and Client.Close.
//
// The leak is observed indirectly: a leaked session keeps its pooled
// keep-alive connection open, so the test server still sees it after every Go
// wrapper has been closed and the collector has run.
func TestRequestHandleFreedAfterSend(t *testing.T) {
	var mu sync.Mutex
	openConns := make(map[net.Conn]struct{})

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	srv.Config.ConnState = func(c net.Conn, s http.ConnState) {
		mu.Lock()
		defer mu.Unlock()
		switch s {
		case http.StateNew:
			openConns[c] = struct{}{}
		case http.StateClosed, http.StateHijacked:
			delete(openConns, c)
		}
	}
	srv.Start()
	defer srv.Close()

	func() {
		client, err := NewDefaultClient()
		if err != nil {
			t.Fatalf("NewDefaultClient: %v", err)
		}
		session, err := NewSession(client)
		if err != nil {
			t.Fatalf("NewSession: %v", err)
		}
		req, err := NewRequest(session, "GET", srv.URL)
		if err != nil {
			t.Fatalf("NewRequest: %v", err)
		}
		resp, err := req.Send()
		if err != nil {
			t.Fatalf("Send: %v", err)
		}
		if got := resp.StatusCode(); got != http.StatusOK {
			t.Fatalf("StatusCode = %d, want %d", got, http.StatusOK)
		}
		resp.Close()
		session.Close()
		client.Close()
	}()

	for i := 0; i < 5; i++ {
		runtime.GC()
		time.Sleep(150 * time.Millisecond)
	}

	mu.Lock()
	n := len(openConns)
	mu.Unlock()
	if n != 0 {
		t.Fatalf("native session was not dropped: %d connection(s) still open", n)
	}
}

// TestResponseFinalizerRuns guards the SetFinalizer safety net on Response.
// Storing a bound method value on the response itself would make the object
// part of a reference cycle, and the runtime never runs finalizers for those,
// so an unclosed Response would leak its whole buffered body.
func TestResponseFinalizerRuns(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"n":7}`))
	}))
	defer srv.Close()

	client, err := NewDefaultClient()
	if err != nil {
		t.Fatalf("NewDefaultClient: %v", err)
	}
	defer client.Close()
	session, err := NewSession(client)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer session.Close()

	// UnmarshalJSON stays a field, so this call shape must keep working.
	req, err := NewRequest(session, "GET", srv.URL)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := req.Send()
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	var payload struct {
		OK bool `json:"ok"`
		N  int  `json:"n"`
	}
	if err := resp.UnmarshalJSON(&payload); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if !payload.OK || payload.N != 7 {
		t.Fatalf("payload = %+v, want {true 7}", payload)
	}
	resp.Close()

	fired := make(chan struct{})
	func() {
		req2, err := NewRequest(session, "GET", srv.URL)
		if err != nil {
			t.Fatalf("NewRequest: %v", err)
		}
		unclosed, err := req2.Send()
		if err != nil {
			t.Fatalf("Send: %v", err)
		}
		_ = unclosed.StatusCode()
		// Swap in an observable finalizer; it still closes the response so the
		// handle is released exactly once.
		runtime.SetFinalizer(unclosed, nil)
		runtime.SetFinalizer(unclosed, func(r *Response) {
			r.Close()
			close(fired)
		})
	}()

	for i := 0; i < 10; i++ {
		runtime.GC()
		select {
		case <-fired:
			return
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}
	t.Fatal("Response finalizer never ran: the object is part of a reference cycle")
}
