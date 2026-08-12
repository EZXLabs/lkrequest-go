//go:build !race

package lkrequest

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"
)

// Excluded under -race: this test calls the bridge directly with Go-allocated
// out cells, which trips checkptr. On the real path the out pointers come from
// the Rust stack.

type varyingResolver struct{}

// Resolve returns a payload whose size depends on the host, so a buffer shared
// between concurrent invocations produces visibly mismatched results.
func (varyingResolver) Resolve(host string, port uint16) ([]string, error) {
	n := 1
	if len(host) > 6 {
		n = 8
	}
	addrs := make([]string, 0, n)
	for i := 0; i < n; i++ {
		addrs = append(addrs, fmt.Sprintf("%s:%d", host, port))
	}
	return addrs, nil
}

// TestCallbackScratchNoCrossTalk guards against callbacks sharing one output
// buffer. It replicates the native call sequence: invoke the callback, let it
// return (releasing the bridge's lock), and only then dereference the out
// pointer — which is exactly when a concurrent invocation used to overwrite
// the bytes.
func TestCallbackScratchNoCrossTalk(t *testing.T) {
	state := &dnsResolverState{resolver: varyingResolver{}}
	const ctxID = uintptr(0xBEEF)
	dnsResolverRegistry.Store(ctxID, state)
	defer func() {
		dnsResolverRegistry.Delete(ctxID)
		state.close()
	}()

	const (
		goroutines = 8
		iterations = 3000
	)

	var crossReads, total int64
	var wg sync.WaitGroup

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()

			host := "aa"
			if g%2 == 0 {
				host = "longhostname"
			}
			hostPtr, hostBuf := stringToCString(host)
			// Pin: otherwise escape analysis keeps the buffer on the goroutine
			// stack and stack growth would invalidate hostPtr.
			var pinner runtime.Pinner
			pinner.Pin(&hostBuf[0])
			defer pinner.Unpin()

			for i := 0; i < iterations; i++ {
				cell := acquireFFIOutCell()
				rc := dnsResolveBridge(
					ctxID,
					hostPtr, uintptr(len(host)), 80,
					uintptr(unsafe.Pointer(&cell[0])),
					uintptr(unsafe.Pointer(&cell[1])),
				)
				outPtr, outLen := cell[0], cell[1]
				ffiOutCellPool.Put(cell)
				if rc != 0 || outPtr == 0 {
					continue
				}

				got := string(unsafe.Slice((*byte)(unsafe.Pointer(outPtr)), outLen))
				atomic.AddInt64(&total, 1)
				if !strings.Contains(got, host) {
					atomic.AddInt64(&crossReads, 1)
				}
			}
		}(g)
	}
	wg.Wait()

	if total == 0 {
		t.Fatal("no successful reads: the test did not exercise the bridge")
	}
	if crossReads != 0 {
		t.Fatalf("callback output was overwritten by a concurrent invocation: %d of %d reads", crossReads, total)
	}
}

type panicingResolver struct{}

func (panicingResolver) Resolve(host string, port uint16) ([]string, error) {
	var s []int
	_ = s[42] // out of range: an ordinary bug in caller code
	return nil, nil
}

type panicingProvider struct{}

func (panicingProvider) NextProxy() (string, bool) {
	var m map[string]string
	m["boom"] = "x" // assignment to nil map
	return "", false
}
func (panicingProvider) Len() uint       { panic("Len") }
func (panicingProvider) IsDynamic() bool { panic("IsDynamic") }

// TestCallbackPanicIsContained guards the recover in the callback bridges. The
// bridges run with Go frames on top of Rust frames, so a panic escaping a
// user-supplied callback would unwind across that boundary and abort the whole
// process instead of failing the single request.
func TestCallbackPanicIsContained(t *testing.T) {
	resolverState := &dnsResolverState{resolver: panicingResolver{}}
	const dnsCtx = uintptr(0xD1)
	dnsResolverRegistry.Store(dnsCtx, resolverState)
	defer func() {
		dnsResolverRegistry.Delete(dnsCtx)
		resolverState.close()
	}()

	host := "example.com"
	hostPtr, hostBuf := stringToCString(host)
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	rc := dnsResolveBridge(dnsCtx, hostPtr, uintptr(len(host)), 443,
		uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])))
	runtime.KeepAlive(hostBuf)
	if rc != dnsResolverErr {
		t.Fatalf("dnsResolveBridge = %#x, want dnsResolverErr", rc)
	}

	providerState := &proxyProviderState{provider: panicingProvider{}}
	const proxyCtx = uintptr(0xD2)
	providerRegistry.Store(proxyCtx, providerState)
	defer func() {
		providerRegistry.Delete(proxyCtx)
		providerState.close()
	}()

	pcell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(pcell)
	if got := providerNextProxyBridge(proxyCtx,
		uintptr(unsafe.Pointer(&pcell[0])), uintptr(unsafe.Pointer(&pcell[1]))); got != ^uintptr(0) {
		t.Fatalf("providerNextProxyBridge = %#x, want LK_ERR", got)
	}
	if got := providerLenBridge(proxyCtx); got != 0 {
		t.Fatalf("providerLenBridge = %d, want 0", got)
	}
	if got := providerIsDynamicBridge(proxyCtx); got != 0 {
		t.Fatalf("providerIsDynamicBridge = %d, want 0", got)
	}
}
