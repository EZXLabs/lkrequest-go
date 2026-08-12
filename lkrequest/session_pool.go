package lkrequest

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// SessionPoolGuard represents a lease on a session from a SessionPool.
type SessionPoolGuard struct {
	ptr  uintptr
	once sync.Once
}

func newSessionPoolGuardFromPtr(ptr uintptr) *SessionPoolGuard {
	if ptr == 0 {
		return nil
	}

	g := &SessionPoolGuard{ptr: ptr}
	runtime.SetFinalizer(g, (*SessionPoolGuard).Close)
	return g
}

// NewRequest creates a new Request from this guard's session.
// The request holds an internal reference to the session; the guard may be freed independently.
func (g *SessionPoolGuard) NewRequest(method, url string) (*Request, error) {
	if g == nil || g.ptr == 0 {
		return nil, ErrNilHandle
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	methodPtr, methodBuf := stringToCString(method)
	urlPtr, urlBuf := stringToCString(url)
	status := ffi_lk_session_pool_guard_request_new(g.ptr, methodPtr, uintptr(len(method)), urlPtr, uintptr(len(url)), uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])))
	outRequest, outErr := cell[0], cell[1]
	runtime.KeepAlive(methodBuf)
	runtime.KeepAlive(urlBuf)
	runtime.KeepAlive(g)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "session pool guard request new"); err != nil {
		return nil, err
	}
	if outRequest == 0 {
		return nil, nilHandleError("session pool guard request new")
	}

	return newRequestFromPtr(outRequest, nil, method, url), nil
}

func (g *SessionPoolGuard) Close() {
	if g == nil {
		return
	}

	g.once.Do(func() {
		ptr := g.ptr
		g.ptr = 0
		runtime.SetFinalizer(g, nil)
		if ptr != 0 {
			ffi_lk_session_pool_guard_free(ptr)
		}
	})
}

// SessionPool manages a pool of sessions, each backed by a different proxy.
type SessionPool struct {
	ptr  uintptr
	once sync.Once
}

func newSessionPoolFromPtr(ptr uintptr) *SessionPool {
	if ptr == 0 {
		return nil
	}

	p := &SessionPool{ptr: ptr}
	runtime.SetFinalizer(p, (*SessionPool).Close)
	return p
}

func (p *SessionPool) loadPtr() uintptr {
	if p == nil {
		return 0
	}
	return atomic.LoadUintptr(&p.ptr)
}

func (p *SessionPool) Acquire() (*SessionPoolGuard, error) {
	ptr := p.loadPtr()
	if ptr == 0 {
		return nil, ErrNilHandle
	}

	cell := acquireFFIOutCell()
	status := ffi_lk_session_pool_acquire(ptr, uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])))
	outGuard, outErr := cell[0], cell[1]
	ffiOutCellPool.Put(cell)
	runtime.KeepAlive(p)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "session pool acquire"); err != nil {
		return nil, err
	}
	if outGuard == 0 {
		return nil, nilHandleError("session pool acquire")
	}

	return newSessionPoolGuardFromPtr(outGuard), nil
}

func (p *SessionPool) AcquireAsync(ctx context.Context) (*SessionPoolGuard, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ptr := p.loadPtr()
	if ptr == 0 {
		return nil, ErrNilHandle
	}

	cell := acquireFFIOutCell()
	status := ffi_lk_session_pool_acquire_async(ptr, uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])))
	outOp, outErr := cell[0], cell[1]
	ffiOutCellPool.Put(cell)
	runtime.KeepAlive(p)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "session pool acquire async"); err != nil {
		return nil, err
	}
	if outOp == 0 {
		return nil, nilHandleError("session pool acquire async")
	}
	defer ffi_lk_op_free(outOp)

	state, err := waitForOp(ctx, outOp)
	if err != nil {
		return nil, err
	}

	return takeSessionPoolGuardFromOp(outOp, state)
}

func (p *SessionPool) AcquireFresh(badGuard *SessionPoolGuard) (*SessionPoolGuard, error) {
	ptr := p.loadPtr()
	if ptr == 0 {
		return nil, ErrNilHandle
	}

	var badPtr uintptr
	if badGuard != nil {
		badPtr = badGuard.ptr
	}

	cell := acquireFFIOutCell()
	status := ffi_lk_session_pool_acquire_fresh(ptr, badPtr, uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])))
	outGuard, outErr := cell[0], cell[1]
	ffiOutCellPool.Put(cell)
	runtime.KeepAlive(p)
	runtime.KeepAlive(badGuard)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "session pool acquire fresh"); err != nil {
		return nil, err
	}
	if outGuard == 0 {
		return nil, nilHandleError("session pool acquire fresh")
	}

	return newSessionPoolGuardFromPtr(outGuard), nil
}

func (p *SessionPool) MarkBad(guard *SessionPoolGuard) error {
	ptr := p.loadPtr()
	if ptr == 0 {
		return ErrNilHandle
	}

	var guardPtr uintptr
	if guard != nil {
		guardPtr = guard.ptr
	}

	status := ffi_lk_session_pool_mark_bad(ptr, guardPtr)
	runtime.KeepAlive(p)
	runtime.KeepAlive(guard)
	return statusError(status, "session pool mark bad")
}

// SessionPoolStats holds the pool's idle and max session counts.
type SessionPoolStats struct {
	Idle uint
	Max  uint
}

func (p *SessionPool) Stats() (SessionPoolStats, error) {
	var stats SessionPoolStats
	ptr := p.loadPtr()
	if ptr == 0 {
		return stats, ErrNilHandle
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_session_pool_stats(ptr, uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])))
	idle, max := cell[0], cell[1]
	runtime.KeepAlive(p)

	if err := statusError(status, "session pool stats"); err != nil {
		return stats, err
	}
	stats.Idle = uint(idle)
	stats.Max = uint(max)
	return stats, nil
}

func (p *SessionPool) Close() {
	if p == nil {
		return
	}

	p.once.Do(func() {
		ptr := atomic.SwapUintptr(&p.ptr, 0)
		runtime.SetFinalizer(p, nil)
		if ptr != 0 {
			ffi_lk_session_pool_free(ptr)
		}
	})
}

func takeSessionPoolGuardFromOp(op uintptr, state OpState) (*SessionPoolGuard, error) {
	switch state {
	case OpCompletedOK:
		cell := acquireFFIOutCell()
		defer ffiOutCellPool.Put(cell)
		status := ffi_lk_op_take_session_pool_guard(op, uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])))
		outGuard, outErr := cell[0], cell[1]
		if err := extractError(outErr); err != nil {
			return nil, err
		}
		if err := statusError(status, "op take session pool guard"); err != nil {
			return nil, err
		}
		if outGuard == 0 {
			return nil, nilHandleError("op take session pool guard")
		}
		return newSessionPoolGuardFromPtr(outGuard), nil
	case OpCompletedErr:
		return nil, takeOperationError(op, "session pool acquire")
	case OpCancelled:
		return nil, context.Canceled
	case OpConsumed:
		return nil, fmt.Errorf("lk: op already consumed")
	default:
		return nil, fmt.Errorf("lk: unexpected op state %s", state)
	}
}

// SessionPoolBuilder builds a SessionPool with configuration.
type SessionPoolBuilder struct {
	ptr      uintptr
	firstErr error
	client   *Client
}

func NewSessionPoolBuilder(client *Client) *SessionPoolBuilder {
	b := &SessionPoolBuilder{client: client}
	cp := client.loadPtr()
	if cp == 0 {
		b.firstErr = ErrNilHandle
		return b
	}

	b.ptr = ffi_lk_session_pool_builder_new(cp)
	if b.ptr == 0 {
		b.firstErr = nilHandleError("session pool builder new")
	}

	runtime.SetFinalizer(b, (*SessionPoolBuilder).finalize)
	runtime.KeepAlive(client)
	return b
}

func (b *SessionPoolBuilder) finalize() {
	b.release()
}

func (b *SessionPoolBuilder) release() {
	if b == nil {
		return
	}

	ptr := b.ptr
	b.ptr = 0
	runtime.SetFinalizer(b, nil)
	if ptr != 0 {
		ffi_lk_session_pool_builder_free(ptr)
	}
}

func (b *SessionPoolBuilder) check() bool {
	if b == nil {
		return false
	}
	if b.firstErr != nil {
		return false
	}
	if b.ptr == 0 {
		b.firstErr = ErrNilHandle
		return false
	}
	return true
}

func (b *SessionPoolBuilder) setStatus(status int32, action string) *SessionPoolBuilder {
	if err := statusError(status, action); err != nil && b.firstErr == nil {
		b.firstErr = err
	}
	return b
}

func (b *SessionPoolBuilder) AddProxy(url string) *SessionPoolBuilder {
	if !b.check() {
		return b
	}

	urlPtr, urlBuf := stringToCString(url)
	status := ffi_lk_session_pool_builder_add_proxy(b.ptr, urlPtr, uintptr(len(url)))
	runtime.KeepAlive(urlBuf)
	return b.setStatus(status, "session pool builder add proxy")
}

func (b *SessionPoolBuilder) AddProxies(urls []string) *SessionPoolBuilder {
	if !b.check() {
		return b
	}
	if len(urls) == 0 {
		return b
	}

	ptrs := make([]uintptr, len(urls))
	lens := make([]uintptr, len(urls))
	bufs := make([][]byte, len(urls))
	for i, u := range urls {
		p, buf := stringToCString(u)
		ptrs[i] = p
		lens[i] = uintptr(len(u))
		bufs[i] = buf
	}

	status := ffi_lk_session_pool_builder_add_proxies(b.ptr, uintptr(unsafe.Pointer(&ptrs[0])), uintptr(unsafe.Pointer(&lens[0])), uintptr(len(urls)))
	runtime.KeepAlive(ptrs)
	runtime.KeepAlive(lens)
	runtime.KeepAlive(bufs)
	return b.setStatus(status, "session pool builder add proxies")
}

func (b *SessionPoolBuilder) SetRotation(strategy RotationStrategy) *SessionPoolBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(ffi_lk_session_pool_builder_set_rotation(b.ptr, int32(strategy)), "session pool builder set rotation")
}

func (b *SessionPoolBuilder) SetProxyBuffer(capacity uint) *SessionPoolBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(ffi_lk_session_pool_builder_set_proxy_buffer(b.ptr, uintptr(capacity)), "session pool builder set proxy buffer")
}

func (b *SessionPoolBuilder) SetMaxSessions(n uint) *SessionPoolBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(ffi_lk_session_pool_builder_set_max_sessions(b.ptr, uintptr(n)), "session pool builder set max sessions")
}

func (b *SessionPoolBuilder) SetIdleTimeout(ms uint64) *SessionPoolBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(ffi_lk_session_pool_builder_set_idle_timeout(b.ptr, ms), "session pool builder set idle timeout")
}

func (b *SessionPoolBuilder) SetHealthCheck(host string, port uint16, intervalMs, timeoutMs uint64) *SessionPoolBuilder {
	if !b.check() {
		return b
	}

	hostPtr, hostBuf := stringToCString(host)
	status := ffi_lk_session_pool_builder_set_health_check(b.ptr, hostPtr, uintptr(len(host)), port, intervalMs, timeoutMs)
	runtime.KeepAlive(hostBuf)
	return b.setStatus(status, "session pool builder set health check")
}

func (b *SessionPoolBuilder) SetProvider(provider ProxyProvider) *SessionPoolBuilder {
	if !b.check() {
		return b
	}

	ffiProvider, release, err := newFFIProxyProvider(provider)
	if err != nil {
		if b.firstErr == nil {
			b.firstErr = err
		}
		return b
	}

	status := setSessionPoolProvider(b.ptr, ffiProvider)
	runtime.KeepAlive(ffiProvider)
	if status != ffiStatusOK {
		release()
	}
	return b.setStatus(status, "session pool builder set provider")
}

func (b *SessionPoolBuilder) SetBadProxyConfig(failureThreshold uint32, windowMs, cooldownMs uint64, maxCooldowns uint32) *SessionPoolBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(ffi_lk_session_pool_builder_set_bad_proxy_config(b.ptr, failureThreshold, windowMs, cooldownMs, maxCooldowns), "session pool builder set bad proxy config")
}

func (b *SessionPoolBuilder) Build() (*SessionPool, error) {
	if b == nil {
		return nil, ErrNilHandle
	}
	defer b.release()

	if b.firstErr != nil {
		return nil, b.firstErr
	}
	if b.ptr == 0 {
		return nil, ErrNilHandle
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_session_pool_builder_build(b.ptr, uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])))
	outPool, outErr := cell[0], cell[1]
	runtime.KeepAlive(b.client)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "session pool builder build"); err != nil {
		return nil, err
	}
	if outPool == 0 {
		return nil, nilHandleError("session pool builder build")
	}

	return newSessionPoolFromPtr(outPool), nil
}
