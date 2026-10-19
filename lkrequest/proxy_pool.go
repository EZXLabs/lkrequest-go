package lkrequest

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// ProxyGuard represents a lease on a proxy from a ProxyPool.
type ProxyGuard struct {
	ptr  uintptr
	once sync.Once
}

func newProxyGuardFromPtr(ptr uintptr) *ProxyGuard {
	if ptr == 0 {
		return nil
	}

	g := &ProxyGuard{ptr: ptr}
	runtime.SetFinalizer(g, (*ProxyGuard).Close)
	return g
}

// URL returns the URL of the proxy the lease holds, or "" for a lease from a
// pool built without any proxy, which connects directly.
func (g *ProxyGuard) URL() string {
	defer runtime.KeepAlive(g)
	if g == nil || g.ptr == 0 {
		return ""
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	if ffi_lk_proxy_guard_url(g.ptr, uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1]))) != ffiStatusOK {
		return ""
	}
	return goStringN(unsafe.Pointer(cell[0]), cell[1])
}

func (g *ProxyGuard) MarkBad() error {
	if g == nil || g.ptr == 0 {
		return ErrNilHandle
	}
	return statusError(ffi_lk_proxy_guard_mark_bad(g.ptr), "proxy guard mark bad")
}

func (g *ProxyGuard) Close() {
	if g == nil {
		return
	}

	g.once.Do(func() {
		ptr := g.ptr
		g.ptr = 0
		runtime.SetFinalizer(g, nil)
		if ptr != 0 {
			ffi_lk_proxy_guard_free(ptr)
		}
	})
}

// ProxyPool manages a pool of proxies with rotation, health checking, etc.
type ProxyPool struct {
	ptr  uintptr
	once sync.Once
}

func newProxyPoolFromPtr(ptr uintptr) *ProxyPool {
	if ptr == 0 {
		return nil
	}

	p := &ProxyPool{ptr: ptr}
	runtime.SetFinalizer(p, (*ProxyPool).Close)
	return p
}

func (p *ProxyPool) loadPtr() uintptr {
	if p == nil {
		return 0
	}
	return atomic.LoadUintptr(&p.ptr)
}

// Acquire leases a proxy, blocking until the pool can hand one out. A pool
// with proxies never answers with a direct lease: while every proxy is cooling
// down, Acquire waits for one to recover, and once every proxy is removed for
// good it never returns. Use AcquireAsync to bound the wait.
func (p *ProxyPool) Acquire() (*ProxyGuard, error) {
	ptr := p.loadPtr()
	if ptr == 0 {
		return nil, ErrNilHandle
	}

	cell := acquireFFIOutCell()
	status := ffi_lk_proxy_pool_acquire(ptr, uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])))
	outGuard, outErr := cell[0], cell[1]
	ffiOutCellPool.Put(cell)
	runtime.KeepAlive(p)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "proxy pool acquire"); err != nil {
		return nil, err
	}
	if outGuard == 0 {
		return nil, nilHandleError("proxy pool acquire")
	}

	return newProxyGuardFromPtr(outGuard), nil
}

// AcquireAsync is Acquire bounded by ctx. If ctx ends first, it gives up the
// wait, frees the permit the acquisition held, and returns ctx's error.
func (p *ProxyPool) AcquireAsync(ctx context.Context) (*ProxyGuard, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ptr := p.loadPtr()
	if ptr == 0 {
		return nil, ErrNilHandle
	}

	cell := acquireFFIOutCell()
	status := ffi_lk_proxy_pool_acquire_async(ptr, uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])))
	outOp, outErr := cell[0], cell[1]
	ffiOutCellPool.Put(cell)
	runtime.KeepAlive(p)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "proxy pool acquire async"); err != nil {
		return nil, err
	}
	if outOp == 0 {
		return nil, nilHandleError("proxy pool acquire async")
	}
	defer ffi_lk_op_free(outOp)

	state, err := waitForOp(ctx, outOp)
	if err != nil {
		return nil, err
	}

	return takeProxyGuardFromOp(outOp, state)
}

// AcquireFresh marks badGuard's proxy bad and leases a proxy again, waiting as
// Acquire does. badGuard keeps its permit until it is closed.
func (p *ProxyPool) AcquireFresh(badGuard *ProxyGuard) (*ProxyGuard, error) {
	ptr := p.loadPtr()
	if ptr == 0 {
		return nil, ErrNilHandle
	}

	var badPtr uintptr
	if badGuard != nil {
		badPtr = badGuard.ptr
	}

	cell := acquireFFIOutCell()
	status := ffi_lk_proxy_pool_acquire_fresh(ptr, badPtr, uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])))
	outGuard, outErr := cell[0], cell[1]
	ffiOutCellPool.Put(cell)
	runtime.KeepAlive(p)
	runtime.KeepAlive(badGuard)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "proxy pool acquire fresh"); err != nil {
		return nil, err
	}
	if outGuard == 0 {
		return nil, nilHandleError("proxy pool acquire fresh")
	}

	return newProxyGuardFromPtr(outGuard), nil
}

// MarkBad records a failure against the proxy with the given identity, as
// reported by ProxyConfig.Identity — not its URL, which matches nothing. A
// cooldown or removal the failure triggers is in effect when MarkBad returns.
func (p *ProxyPool) MarkBad(identity string) error {
	ptr := p.loadPtr()
	if ptr == 0 {
		return ErrNilHandle
	}

	idPtr, idBuf := stringToCString(identity)
	status := ffi_lk_proxy_pool_mark_bad(ptr, idPtr, uintptr(len(identity)))
	runtime.KeepAlive(idBuf)
	return statusError(status, "proxy pool mark bad")
}

func (p *ProxyPool) MaxConcurrent() uint {
	ptr := p.loadPtr()
	if ptr == 0 {
		return 0
	}
	return uint(ffi_lk_proxy_pool_max_concurrent(ptr))
}

func (p *ProxyPool) Close() {
	if p == nil {
		return
	}

	p.once.Do(func() {
		ptr := atomic.SwapUintptr(&p.ptr, 0)
		runtime.SetFinalizer(p, nil)
		if ptr != 0 {
			ffi_lk_proxy_pool_free(ptr)
		}
	})
}

func takeProxyGuardFromOp(op uintptr, state OpState) (*ProxyGuard, error) {
	switch state {
	case OpCompletedOK:
		cell := acquireFFIOutCell()
		defer ffiOutCellPool.Put(cell)
		status := ffi_lk_op_take_proxy_guard(op, uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])))
		outGuard, outErr := cell[0], cell[1]
		if err := extractError(outErr); err != nil {
			return nil, err
		}
		if err := statusError(status, "op take proxy guard"); err != nil {
			return nil, err
		}
		if outGuard == 0 {
			return nil, nilHandleError("op take proxy guard")
		}
		return newProxyGuardFromPtr(outGuard), nil
	case OpCompletedErr:
		return nil, takeOperationError(op, "proxy pool acquire")
	case OpCancelled:
		return nil, context.Canceled
	case OpConsumed:
		return nil, fmt.Errorf("lk: op already consumed")
	default:
		return nil, fmt.Errorf("lk: unexpected op state %s", state)
	}
}

// ProxyPoolBuilder builds a ProxyPool with configuration.
type ProxyPoolBuilder struct {
	ptr      uintptr
	firstErr error
}

func NewProxyPoolBuilder() *ProxyPoolBuilder {
	b := &ProxyPoolBuilder{ptr: ffi_lk_proxy_pool_builder_new()}
	if b.ptr == 0 {
		b.firstErr = nilHandleError("proxy pool builder new")
	}

	runtime.SetFinalizer(b, (*ProxyPoolBuilder).finalize)
	return b
}

func (b *ProxyPoolBuilder) finalize() {
	b.release()
}

func (b *ProxyPoolBuilder) release() {
	if b == nil {
		return
	}

	ptr := b.ptr
	b.ptr = 0
	runtime.SetFinalizer(b, nil)
	if ptr != 0 {
		ffi_lk_proxy_pool_builder_free(ptr)
	}
}

func (b *ProxyPoolBuilder) check() bool {
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

func (b *ProxyPoolBuilder) setStatus(status int32, action string) *ProxyPoolBuilder {
	if err := statusError(status, action); err != nil && b.firstErr == nil {
		b.firstErr = err
	}
	return b
}

func (b *ProxyPoolBuilder) AddProxy(url string) *ProxyPoolBuilder {
	if !b.check() {
		return b
	}

	urlPtr, urlBuf := stringToCString(url)
	status := ffi_lk_proxy_pool_builder_add_proxy(b.ptr, urlPtr, uintptr(len(url)))
	runtime.KeepAlive(urlBuf)
	return b.setStatus(status, "proxy pool builder add proxy")
}

func (b *ProxyPoolBuilder) AddProxies(urls []string) *ProxyPoolBuilder {
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

	ptrsPtr, lensPtr := unsafe.Pointer(&ptrs[0]), unsafe.Pointer(&lens[0])
	escapeToHeap(ptrsPtr)
	escapeToHeap(lensPtr)
	status := ffi_lk_proxy_pool_builder_add_proxies(b.ptr, uintptr(ptrsPtr), uintptr(lensPtr), uintptr(len(urls)))
	runtime.KeepAlive(ptrs)
	runtime.KeepAlive(lens)
	runtime.KeepAlive(bufs)
	return b.setStatus(status, "proxy pool builder add proxies")
}

func (b *ProxyPoolBuilder) SetRotation(strategy RotationStrategy) *ProxyPoolBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(ffi_lk_proxy_pool_builder_set_rotation(b.ptr, int32(strategy)), "proxy pool builder set rotation")
}

func (b *ProxyPoolBuilder) SetProxyBuffer(capacity uint) *ProxyPoolBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(ffi_lk_proxy_pool_builder_set_proxy_buffer(b.ptr, uintptr(capacity)), "proxy pool builder set proxy buffer")
}

func (b *ProxyPoolBuilder) SetHealthCheck(host string, port uint16, intervalMs, timeoutMs uint64) *ProxyPoolBuilder {
	if !b.check() {
		return b
	}

	hostPtr, hostBuf := stringToCString(host)
	status := ffi_lk_proxy_pool_builder_set_health_check(b.ptr, hostPtr, uintptr(len(host)), port, intervalMs, timeoutMs)
	runtime.KeepAlive(hostBuf)
	return b.setStatus(status, "proxy pool builder set health check")
}

func (b *ProxyPoolBuilder) SetProvider(provider ProxyProvider) *ProxyPoolBuilder {
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

	status := setProxyPoolProvider(b.ptr, ffiProvider)
	runtime.KeepAlive(ffiProvider)
	if status != ffiStatusOK {
		release()
	}
	return b.setStatus(status, "proxy pool builder set provider")
}

func (b *ProxyPoolBuilder) SetBadProxyConfig(failureThreshold uint32, windowMs, cooldownMs uint64, maxCooldowns uint32) *ProxyPoolBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(ffi_lk_proxy_pool_builder_set_bad_proxy_config(b.ptr, failureThreshold, windowMs, cooldownMs, maxCooldowns), "proxy pool builder set bad proxy config")
}

func (b *ProxyPoolBuilder) SetMaxProxies(n uint) *ProxyPoolBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(ffi_lk_proxy_pool_builder_set_max_proxies(b.ptr, uintptr(n)), "proxy pool builder set max proxies")
}

func (b *ProxyPoolBuilder) Build() (*ProxyPool, error) {
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
	status := ffi_lk_proxy_pool_builder_build(b.ptr, uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])))
	outPool, outErr := cell[0], cell[1]

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "proxy pool builder build"); err != nil {
		return nil, err
	}
	if outPool == 0 {
		return nil, nilHandleError("proxy pool builder build")
	}

	return newProxyPoolFromPtr(outPool), nil
}
