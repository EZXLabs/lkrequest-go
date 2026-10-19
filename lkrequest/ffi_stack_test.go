package lkrequest

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestFFIArgumentsSurviveStackMoves passes short strings — small enough that
// escape analysis would keep their C copies on the goroutine stack — from
// fresh goroutines, which start on small stacks that grow inside the FFI call
// path, while a GC loop shrinks stacks. Argument memory left on a stack would
// be read after the stack moved: purego calls then fail or see the wrong
// value, and cgo aborts with "found bad pointer in Go heap". Each worker owns
// its session, so no cookie-buffer sharing is involved.
func TestFFIArgumentsSurviveStackMoves(t *testing.T) {
	client, err := NewClientBuilder().Build()
	if err != nil {
		t.Fatalf("NewClientBuilder().Build() error = %v", err)
	}
	t.Cleanup(client.Close)

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				runtime.GC()
			}
		}
	}()

	var calls, failures atomic.Int64
	var firstFailure atomic.Value
	deadline := time.Now().Add(1500 * time.Millisecond)
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		session, err := NewSession(client)
		if err != nil {
			t.Fatalf("NewSession() error = %v", err)
		}
		t.Cleanup(session.Close)
		url := fmt.Sprintf("https://h%d.test/", w)

		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				done := make(chan struct{})
				go func() {
					defer close(done)
					calls.Add(1)
					if err := setAndReadCookie(session, url); err != nil {
						failures.Add(1)
						firstFailure.CompareAndSwap(nil, err.Error())
					}
				}()
				<-done
			}
		}()
	}
	wg.Wait()

	if n := failures.Load(); n > 0 {
		t.Fatalf("%d of %d FFI round trips saw corrupted arguments; first: %v", n, calls.Load(), firstFailure.Load())
	}
}

// setAndReadCookie writes a cookie and reads it back.
func setAndReadCookie(session *Session, url string) error {
	if err := session.SetCookie(url, "k", "v"); err != nil {
		return err
	}
	if v, err := session.GetCookie(url, "k"); err != nil || v != "v" {
		return fmt.Errorf("GetCookie() = %q, %v; want \"v\"", v, err)
	}
	return nil
}
