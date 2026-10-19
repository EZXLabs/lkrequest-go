package lkrequest

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestProxyPoolNeverFallsBackToDirect pins what a pool that has proxies does
// when it cannot hand one out: it waits, and never answers with a direct lease,
// which a caller would take for a direct connection and so expose its own
// address. Only a pool built without any proxy is direct.
func TestProxyPoolNeverFallsBackToDirect(t *testing.T) {
	const proxyURL = "http://only.example:8080"

	// markedPool returns a one-permit pool whose only proxy has just been
	// marked bad, which with a failure threshold of 1 cools it down.
	markedPool := func(t *testing.T, cooldownMs uint64, maxCooldowns uint32) *ProxyPool {
		t.Helper()
		identity, err := newProxyConfig(t, proxyURL).Identity()
		if err != nil {
			t.Fatalf("Identity() error = %v", err)
		}
		pool, err := NewProxyPoolBuilder().
			AddProxy(proxyURL).
			SetMaxProxies(1).
			SetBadProxyConfig(1, 60_000, cooldownMs, maxCooldowns).
			Build()
		if err != nil {
			t.Fatalf("ProxyPoolBuilder.Build() error = %v", err)
		}
		t.Cleanup(pool.Close)
		if err := pool.MarkBad(identity); err != nil {
			t.Fatalf("MarkBad() error = %v", err)
		}
		return pool
	}

	t.Run("cooling down", func(t *testing.T) {
		pool := markedPool(t, 500, 5)
		expectAcquirePending(t, pool)

		// The abandoned acquisition gave back the pool's only permit, and the
		// proxy is handed out again once its cooldown ends.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		guard, err := pool.AcquireAsync(ctx)
		if err != nil {
			t.Fatalf("AcquireAsync() after the cooldown error = %v", err)
		}
		defer guard.Close()
		if got := guard.URL(); got != proxyURL {
			t.Fatalf("URL() after the cooldown = %q, want %q", got, proxyURL)
		}
	})

	t.Run("removed for good", func(t *testing.T) {
		expectAcquirePending(t, markedPool(t, 60_000, 1))
	})

	t.Run("no proxies", func(t *testing.T) {
		pool, err := NewProxyPoolBuilder().Build()
		if err != nil {
			t.Fatalf("ProxyPoolBuilder.Build() error = %v", err)
		}
		t.Cleanup(pool.Close)
		guard, err := pool.Acquire()
		if err != nil {
			t.Fatalf("Acquire() error = %v", err)
		}
		defer guard.Close()
		if got := guard.URL(); got != "" {
			t.Fatalf("URL() from a pool without proxies = %q, want a direct lease", got)
		}
	})
}

// expectAcquirePending checks that an acquisition from pool is still waiting
// when its context expires, rather than returning a lease.
func expectAcquirePending(t *testing.T, pool *ProxyPool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	guard, err := pool.AcquireAsync(ctx)
	if err == nil {
		url := guard.URL()
		guard.Close()
		t.Fatalf("AcquireAsync() returned a lease for %q, want it to wait until the context expires", url)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("AcquireAsync() error = %v, want context.DeadlineExceeded", err)
	}
}

// TestSessionPoolDiscardsMarkedSessions pins that a session marked bad is
// dropped when its guard closes instead of going back to the idle pool, even
// while its proxy stays below the failure threshold, and that a healthy
// session does go back for reuse.
func TestSessionPoolDiscardsMarkedSessions(t *testing.T) {
	client, err := newBuilderClient()
	if err != nil {
		t.Fatalf("newBuilderClient() error = %v", err)
	}
	t.Cleanup(client.Close)

	pool, err := NewSessionPoolBuilder(client).
		AddProxy("http://proxy-one.example:8080").
		AddProxy("http://proxy-two.example:8080").
		SetMaxSessions(2).
		SetIdleTimeout(60_000).
		Build()
	if err != nil {
		t.Fatalf("SessionPoolBuilder.Build() error = %v", err)
	}
	t.Cleanup(pool.Close)

	idle := func() uint {
		t.Helper()
		stats, err := pool.Stats()
		if err != nil {
			t.Fatalf("Stats() error = %v", err)
		}
		return stats.Idle
	}

	marked, err := pool.Acquire()
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if err := pool.MarkBad(marked); err != nil {
		marked.Close()
		t.Fatalf("MarkBad() error = %v", err)
	}
	marked.Close()
	// Closing returns a session there and then, or not at all. Older libraries
	// returned it on a background task; give such a return time to land, so
	// the check below does not pass merely because it has not landed yet.
	time.Sleep(50 * time.Millisecond)
	if got := idle(); got != 0 {
		t.Fatalf("idle sessions after closing a marked guard = %d, want 0", got)
	}

	healthy, err := pool.Acquire()
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	healthy.Close()
	if got := idle(); got != 1 {
		t.Fatalf("idle sessions after closing a healthy guard = %d, want 1", got)
	}
}
