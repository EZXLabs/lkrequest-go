package lkrequest

import (
	"encoding/json"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// DNSResolver is a custom name resolver installed on a client via
// ClientBuilder.SetDNSResolver.
//
// Resolve is invoked on the library runtime thread and must return
// synchronously without blocking on network I/O or other FFI calls. It returns
// socket-address strings such as "127.0.0.1:443" or "[2606:4700:4700::1111]:443".
// Returning an error (or no addresses) is treated as a resolution failure.
type DNSResolver interface {
	Resolve(host string, port uint16) ([]string, error)
}

// HTTPSLookuper is an optional capability a DNSResolver may implement to answer
// HTTPS/SVCB record lookups used for HTTP/3 discovery. LookupHTTPS returns a
// JSON value matching lkrequest's HttpsRecord, or "" for "no HTTPS record".
type HTTPSLookuper interface {
	LookupHTTPS(host string) (record string, err error)
}

// dnsResolverDestroyer is implemented by resolvers that need cleanup when the
// owning client/builder drops them.
type dnsResolverDestroyer interface {
	Destroy()
}

// ffiDNSResolver mirrors the C lk_dns_resolver_t layout.
type ffiDNSResolver struct {
	context     uintptr
	resolve     uintptr
	lookupHTTPS uintptr
	destroy     uintptr
}

type dnsResolverState struct {
	resolver DNSResolver
	mu       sync.Mutex
	scratch  scratchRing
}

func (s *dnsResolverState) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.scratch.release()
	if destroyer, ok := s.resolver.(dnsResolverDestroyer); ok {
		destroyer.Destroy()
	}
}

const dnsResolverErr = ^uintptr(0) // LK_ERR

var (
	dnsResolverRegistry sync.Map

	dnsResolveBridgePtr     = purego.NewCallback(dnsResolveBridge)
	dnsLookupHTTPSBridgePtr = purego.NewCallback(dnsLookupHTTPSBridge)
	dnsDestroyBridgePtr     = purego.NewCallback(dnsDestroyBridge)
)

func newFFIDNSResolver(resolver DNSResolver) (ffiDNSResolver, func(), error) {
	if resolver == nil {
		return ffiDNSResolver{}, nil, fmt.Errorf("lk: dns resolver is nil")
	}

	contextID := nextCallbackContext()
	state := &dnsResolverState{resolver: resolver}
	dnsResolverRegistry.Store(contextID, state)

	release := func() {
		if value, ok := dnsResolverRegistry.LoadAndDelete(contextID); ok {
			value.(*dnsResolverState).close()
		}
	}

	return ffiDNSResolver{
		context:     contextID,
		resolve:     dnsResolveBridgePtr,
		lookupHTTPS: dnsLookupHTTPSBridgePtr,
		destroy:     dnsDestroyBridgePtr,
	}, release, nil
}

func dnsResolverStateFor(context uintptr) *dnsResolverState {
	if context == 0 {
		return nil
	}
	value, ok := dnsResolverRegistry.Load(context)
	if !ok {
		return nil
	}
	return value.(*dnsResolverState)
}

func dnsResolveBridge(context, hostPtr, hostLen, port, outJSONPtr, outJSONLen uintptr) (result uintptr) {
	defer recoverCallbackPanic("DNS resolver Resolve", &result, dnsResolverErr)

	if outJSONPtr != 0 {
		*(*uintptr)(unsafe.Pointer(outJSONPtr)) = 0
	}
	if outJSONLen != 0 {
		*(*uintptr)(unsafe.Pointer(outJSONLen)) = 0
	}

	state := dnsResolverStateFor(context)
	if state == nil {
		return dnsResolverErr
	}

	host := goStringN(unsafe.Pointer(hostPtr), hostLen)

	state.mu.Lock()
	defer state.mu.Unlock()

	addrs, err := state.resolver.Resolve(host, uint16(port))
	if err != nil || len(addrs) == 0 {
		return dnsResolverErr
	}

	payload, err := json.Marshal(addrs)
	if err != nil {
		return dnsResolverErr
	}

	ptr, n := state.scratch.put(payload)
	if ptr == 0 {
		return dnsResolverErr
	}

	if outJSONPtr != 0 {
		*(*uintptr)(unsafe.Pointer(outJSONPtr)) = ptr
	}
	if outJSONLen != 0 {
		*(*uintptr)(unsafe.Pointer(outJSONLen)) = n
	}
	return 0
}

func dnsLookupHTTPSBridge(context, hostPtr, hostLen, outJSONPtr, outJSONLen uintptr) (result uintptr) {
	defer recoverCallbackPanic("DNS resolver LookupHTTPS", &result, dnsResolverErr)

	if outJSONPtr != 0 {
		*(*uintptr)(unsafe.Pointer(outJSONPtr)) = 0
	}
	if outJSONLen != 0 {
		*(*uintptr)(unsafe.Pointer(outJSONLen)) = 0
	}

	state := dnsResolverStateFor(context)
	if state == nil {
		return dnsResolverErr
	}

	lookuper, ok := state.resolver.(HTTPSLookuper)
	if !ok {
		return 0 // no HTTPS capability: report "no record" without failing.
	}

	host := goStringN(unsafe.Pointer(hostPtr), hostLen)

	state.mu.Lock()
	defer state.mu.Unlock()

	record, err := lookuper.LookupHTTPS(host)
	if err != nil {
		return dnsResolverErr
	}
	if record == "" {
		return 0
	}

	ptr, n := state.scratch.put([]byte(record))
	if ptr == 0 {
		return dnsResolverErr
	}

	if outJSONPtr != 0 {
		*(*uintptr)(unsafe.Pointer(outJSONPtr)) = ptr
	}
	if outJSONLen != 0 {
		*(*uintptr)(unsafe.Pointer(outJSONLen)) = n
	}
	return 0
}

func dnsDestroyBridge(context uintptr) (result uintptr) {
	defer recoverCallbackPanic("DNS resolver Destroy", &result, 0)

	if value, ok := dnsResolverRegistry.LoadAndDelete(context); ok {
		value.(*dnsResolverState).close()
	}
	return 0
}

// SetDNSResolver installs a custom name resolver on the client. Requires a
// library that exports lk_client_builder_set_dns_resolver; otherwise the
// builder records an "unsupported" error surfaced by Build.
func (b *ClientBuilder) SetDNSResolver(resolver DNSResolver) *ClientBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_client_builder_set_dns_resolver == nil {
		return b.setUnsupported("client builder set dns resolver")
	}

	ffiResolver, release, err := newFFIDNSResolver(resolver)
	if err != nil {
		if b.firstErr == nil {
			b.firstErr = err
		}
		return b
	}

	status := setClientBuilderDNSResolver(b.ptr, ffiResolver)
	if status != ffiStatusOK {
		release()
	}
	return b.setStatus(status, "client builder set dns resolver")
}
