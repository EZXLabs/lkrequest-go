package lkrequest

import (
	"os"
	"os/exec"
	"sync/atomic"
	"testing"
)

// TestAcquireWithLogCallbackSubprocess covers synchronous Go re-entry when a
// log callback and provider are enabled together. During an FFI call, Rust can
// invoke Go on the calling thread for tracing events or provider queries. The
// callback may grow and move the goroutine stack (copystack). If an out
// parameter points into that stack, Rust writes to the stale pre-move address,
// which appears in Go as a clean nil handle. ffiOutCellPool fixes this by
// allocating all out parameters on the heap.
func TestAcquireWithLogCallbackSubprocess(t *testing.T) {
	if os.Getenv("LKREQUEST_TEST_LOG_CB_ACQUIRE_SUBPROCESS") == "" {
		// Re-exec the test in subprocess mode because the global log subscriber
		// can only be installed once per process.
		cmd := exec.Command(os.Args[0], "-test.run=^TestAcquireWithLogCallbackSubprocess$")
		cmd.Env = append(os.Environ(), "LKREQUEST_TEST_LOG_CB_ACQUIRE_SUBPROCESS=1")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("log-callback acquire subprocess failed: %v\n%s", err, output)
		}
		return
	}

	// Trace-level logging ensures synchronous re-entry during build and acquire.
	var logFired atomic.Int64
	err := InitLogCallback(func(LogLevel, string, string) {
		logFired.Add(1)
	}, LogLevelTrace)
	if err != nil {
		t.Fatalf("InitLogCallback() error = %v", err)
	}

	// Build the client under callback re-entry, matching the Windows failure
	// path covered by TestInitLogCallbackSubprocess.
	client, err := NewClientBuilder().Build()
	if err != nil {
		t.Fatalf("ClientBuilder.Build() error = %v (logFired=%d)", err, logFired.Load())
	}
	defer client.Close()

	const wantURL = "http://reentrancy-probe.example:8080"
	provider := newLocalProxyProvider(wantURL)

	pool, err := NewProxyPoolBuilder().SetProvider(provider).SetMaxProxies(1).Build()
	if err != nil {
		t.Fatalf("ProxyPoolBuilder.Build() error = %v", err)
	}
	defer pool.Close()

	guard, err := pool.Acquire()
	if err != nil {
		t.Fatalf("ProxyPool.Acquire() error = %v (logFired=%d)", err, logFired.Load())
	}
	defer guard.Close()

	if got := guard.URL(); got != wantURL {
		t.Fatalf("Acquire().URL() = %q, want %q", got, wantURL)
	}

	// Fail explicitly if upstream stops emitting tracing events during acquire,
	// because the test would no longer exercise callback re-entry.
	if logFired.Load() == 0 {
		t.Fatal("log callback never fired; test no longer exercises callback reentrancy")
	}
}
