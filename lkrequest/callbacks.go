package lkrequest

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ebitengine/purego"
)

// recoverCallbackPanic turns a panic raised inside a user-supplied callback
// into an FFI status code.
//
// These bridges are invoked by the native library, so the Go frames sit on top
// of C/Rust frames. A panic unwinding across that boundary skips every Rust
// destructor in between and aborts the whole process, which would turn an
// ordinary bug in a caller's resolver or proxy provider into a hard crash of
// the host program. Reporting a failure status instead lets the native side
// handle it as a normal callback error.
//
// The panic is written to stderr because a swallowed panic would otherwise be
// indistinguishable from a callback that simply returned an error.
func recoverCallbackPanic(name string, result *uintptr, status uintptr) {
	r := recover()
	if r == nil {
		return
	}

	*result = status

	buf := make([]byte, 8192)
	buf = buf[:runtime.Stack(buf, false)]
	fmt.Fprintf(os.Stderr, "lk: recovered panic in %s callback: %v\n%s\n", name, r, buf)
}

// LogLevel controls the minimum severity captured by InitLogCallback.
type LogLevel int32

const (
	LogLevelTrace LogLevel = 0
	LogLevelDebug LogLevel = 1
	LogLevelInfo  LogLevel = 2
	LogLevelWarn  LogLevel = 3
	LogLevelError LogLevel = 4
)

// ProxyProvider supplies proxy URLs on demand for ProxyPoolBuilder / SessionPoolBuilder.
// Implementations must return quickly and avoid blocking I/O.
type ProxyProvider interface {
	NextProxy() (url string, ok bool)
	Len() uint
	IsDynamic() bool
}

// ProxyProviderFuncs is a function-based ProxyProvider helper.
type ProxyProviderFuncs struct {
	NextProxyFunc func() (string, bool)
	LenFunc       func() uint
	IsDynamicFunc func() bool
	DestroyFunc   func()
}

func (p ProxyProviderFuncs) NextProxy() (string, bool) {
	if p.NextProxyFunc == nil {
		return "", false
	}
	return p.NextProxyFunc()
}

func (p ProxyProviderFuncs) Len() uint {
	if p.LenFunc == nil {
		return 0
	}
	return p.LenFunc()
}

func (p ProxyProviderFuncs) IsDynamic() bool {
	if p.IsDynamicFunc == nil {
		return false
	}
	return p.IsDynamicFunc()
}

func (p ProxyProviderFuncs) Destroy() {
	if p.DestroyFunc != nil {
		p.DestroyFunc()
	}
}

type proxyProviderDestroyer interface {
	Destroy()
}

type ffiProxyProvider struct {
	context   uintptr
	nextProxy uintptr
	len       uintptr
	isDynamic uintptr
	destroy   uintptr
}

// callbackScratchSlots is how many recent callback buffers a scratchRing keeps
// reachable before letting the garbage collector reclaim them.
const callbackScratchSlots = 32

// scratchRing owns the buffers handed to the native side by callback bridges.
//
// The native side reads the pointer a bridge writes out *after* the bridge has
// returned: it first checks the status, then turns the pointer into a slice
// and copies the data. A single buffer per resolver/provider is therefore
// unsafe even under a mutex, because the lock is already released by the time
// the data is read, so a concurrent invocation would overwrite the bytes the
// native side is about to copy. That produced torn reads which, when they
// happened to stay well-formed, silently resolved to another caller's address
// or routed through another caller's proxy.
//
// Every invocation therefore gets a freshly allocated buffer that is never
// written again; reuse cannot corrupt an in-flight read because there is no
// reuse. The ring only keeps the most recent buffers reachable so they are not
// collected while the native side may still hold a pointer, and pinning is
// rotated between two pinners so a buffer stays pinned for at least a full lap
// of the ring before its pin is dropped.
//
// This mirrors what the Java binding does with Arena.ofAuto(), and like it the
// safety ultimately rests on the native side copying the data out promptly
// after the callback returns, which is inherent to this FFI contract.
type scratchRing struct {
	mu    sync.Mutex
	slots [callbackScratchSlots][]byte
	next  int
	cur   *runtime.Pinner
	prev  *runtime.Pinner
}

// put copies payload into a fresh buffer and returns a pointer/length pair that
// stays valid for the native side to read after the bridge has returned.
func (r *scratchRing) put(payload []byte) (uintptr, uintptr) {
	if len(payload) == 0 {
		return 0, 0
	}

	buf := make([]byte, len(payload))
	copy(buf, payload)

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cur == nil {
		r.cur = new(runtime.Pinner)
	}
	if r.next == 0 {
		// A full lap: everything pinned by prev is at least callbackScratchSlots
		// calls old, so its pins can be dropped.
		if r.prev != nil {
			r.prev.Unpin()
		}
		r.prev, r.cur = r.cur, new(runtime.Pinner)
	}
	r.cur.Pin(&buf[0])

	r.slots[r.next] = buf
	r.next = (r.next + 1) % callbackScratchSlots

	return bytesPtr(buf), uintptr(len(buf))
}

// release unpins every buffer. It must run once the native side can no longer
// invoke the callback, i.e. from the owning state's close.
func (r *scratchRing) release() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.prev != nil {
		r.prev.Unpin()
		r.prev = nil
	}
	if r.cur != nil {
		r.cur.Unpin()
		r.cur = nil
	}
	for i := range r.slots {
		r.slots[i] = nil
	}
}

type proxyProviderState struct {
	provider ProxyProvider
	mu       sync.Mutex
	scratch  scratchRing
}

func (s *proxyProviderState) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.scratch.release()
	if destroyer, ok := s.provider.(proxyProviderDestroyer); ok {
		destroyer.Destroy()
	}
}

var (
	nextCallbackContextID atomic.Uintptr
	logCallbackRegistry   sync.Map
	providerRegistry      sync.Map

	logCallbackBridgePtr       = purego.NewCallback(logCallbackBridge)
	providerNextProxyBridgePtr = purego.NewCallback(providerNextProxyBridge)
	providerLenBridgePtr       = purego.NewCallback(providerLenBridge)
	providerIsDynamicBridgePtr = purego.NewCallback(providerIsDynamicBridge)
	providerDestroyBridgePtr   = purego.NewCallback(providerDestroyBridge)
)

func nextCallbackContext() uintptr {
	return nextCallbackContextID.Add(1)
}

func InitLogCallback(callback func(level LogLevel, target, message string), minLevel LogLevel) error {
	if callback == nil {
		return fmt.Errorf("lk: log callback is nil")
	}

	// Shared with InitLog: the native subscriber is installed once per process.
	// Without this guard a second call would report success while still routing
	// to the first callback, and would leak its closure into the registry.
	if !logInitialized.CompareAndSwap(false, true) {
		return ErrLogAlreadyInitialized
	}

	contextID := nextCallbackContext()
	logCallbackRegistry.Store(contextID, callback)

	status := ffi_lk_log_init_callback(logCallbackBridgePtr, contextID, int32(minLevel))
	if status != ffiStatusOK {
		logCallbackRegistry.Delete(contextID)
		logInitialized.Store(false) // let the caller retry
	}
	return statusError(status, "log init callback")
}

func logCallbackBridge(context uintptr, level int32, target uintptr, message uintptr) (result uintptr) {
	defer recoverCallbackPanic("log", &result, 0)

	value, ok := logCallbackRegistry.Load(context)
	if !ok {
		return 0
	}
	callback, ok := value.(func(level LogLevel, target, message string))
	if !ok || callback == nil {
		return 0
	}
	callback(
		LogLevel(level),
		goCString(unsafe.Pointer(target)),
		goCString(unsafe.Pointer(message)),
	)
	return 0
}

func newFFIProxyProvider(provider ProxyProvider) (ffiProxyProvider, func(), error) {
	if provider == nil {
		return ffiProxyProvider{}, nil, fmt.Errorf("lk: proxy provider is nil")
	}

	contextID := nextCallbackContext()
	state := &proxyProviderState{provider: provider}
	providerRegistry.Store(contextID, state)

	release := func() {
		if value, ok := providerRegistry.LoadAndDelete(contextID); ok {
			value.(*proxyProviderState).close()
		}
	}

	return ffiProxyProvider{
		context:   contextID,
		nextProxy: providerNextProxyBridgePtr,
		len:       providerLenBridgePtr,
		isDynamic: providerIsDynamicBridgePtr,
		destroy:   providerDestroyBridgePtr,
	}, release, nil
}

func providerStateFor(context uintptr) *proxyProviderState {
	if context == 0 {
		return nil
	}
	value, ok := providerRegistry.Load(context)
	if !ok {
		return nil
	}
	return value.(*proxyProviderState)
}

func providerNextProxyBridge(context uintptr, outURLPtr uintptr, outURLLen uintptr) (result uintptr) {
	defer recoverCallbackPanic("proxy provider NextProxy", &result, ^uintptr(0))

	if outURLPtr != 0 {
		*(*uintptr)(unsafe.Pointer(outURLPtr)) = 0
	}
	if outURLLen != 0 {
		*(*uintptr)(unsafe.Pointer(outURLLen)) = 0
	}

	state := providerStateFor(context)
	if state == nil {
		return ^uintptr(0)
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	url, ok := state.provider.NextProxy()
	if !ok {
		return ^uintptr(0)
	}
	if len(url) == 0 {
		return 0
	}

	ptr, n := state.scratch.put([]byte(url))
	if ptr == 0 {
		return ^uintptr(0)
	}

	if outURLPtr != 0 {
		*(*uintptr)(unsafe.Pointer(outURLPtr)) = ptr
	}
	if outURLLen != 0 {
		*(*uintptr)(unsafe.Pointer(outURLLen)) = n
	}
	return 0
}

func providerLenBridge(context uintptr) (result uintptr) {
	defer recoverCallbackPanic("proxy provider Len", &result, 0)

	state := providerStateFor(context)
	if state == nil {
		return 0
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	return uintptr(state.provider.Len())
}

func providerIsDynamicBridge(context uintptr) (result uintptr) {
	defer recoverCallbackPanic("proxy provider IsDynamic", &result, 0)

	state := providerStateFor(context)
	if state == nil {
		return 0
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	if state.provider.IsDynamic() {
		return 1
	}
	return 0
}

func providerDestroyBridge(context uintptr) (result uintptr) {
	defer recoverCallbackPanic("proxy provider Destroy", &result, 0)

	if value, ok := providerRegistry.LoadAndDelete(context); ok {
		value.(*proxyProviderState).close()
	}
	return 0
}
