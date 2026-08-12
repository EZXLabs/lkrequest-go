//go:build !lkcgo && windows

package lkrequest

import (
	"unsafe"

	"github.com/ebitengine/purego"
)

// These are the provider-related FFI bindings for Windows purego builds.
//
// purego does not support by-value struct arguments on Windows and panics with
// "struct arguments are only supported on darwin and linux". Under the Windows
// x64 calling convention, however, by-value structs larger than 8 bytes are
// passed indirectly through a caller-allocated copy. Declaring the argument as
// uintptr and passing &provider therefore matches the MS x64 ABI and remains
// binary-compatible with Rust's extern "C" provider parameter.
//
// Every ffiProxyProvider field is an integer or C function address: one context
// ID and four purego callback addresses. It contains no Go heap pointers, so
// passing its address to C is valid.
var (
	ffi_lk_proxy_pool_builder_set_provider   func(builder uintptr, provider uintptr) int32
	ffi_lk_session_pool_builder_set_provider func(builder uintptr, provider uintptr) int32
	// The DNS resolver also has a by-value struct signature and is passed
	// indirectly on Windows. It is optional for older libraries, so a missing
	// export leaves the function pointer nil.
	ffi_lk_client_builder_set_dns_resolver func(builder uintptr, resolver uintptr) int32
)

// registerProviderFuncs registers the platform-specific by-value struct
// symbols for the two provider setters and the DNS resolver.
func registerProviderFuncs(handle uintptr) {
	purego.RegisterLibFunc(&ffi_lk_proxy_pool_builder_set_provider, handle, "lk_proxy_pool_builder_set_provider")
	purego.RegisterLibFunc(&ffi_lk_session_pool_builder_set_provider, handle, "lk_session_pool_builder_set_provider")
	registerOptional(&ffi_lk_client_builder_set_dns_resolver, handle, "lk_client_builder_set_dns_resolver")
}

// setProxyPoolProvider passes the provider indirectly as required by the MS
// x64 ABI. The heap copy is pinned so stack movement cannot invalidate the
// address held by C during the FFI call.
func setProxyPoolProvider(builder uintptr, provider ffiProxyProvider) int32 {
	heap, unpin := ffiPinnedCopy(provider)
	status := ffi_lk_proxy_pool_builder_set_provider(builder, uintptr(unsafe.Pointer(heap)))
	unpin()
	return status
}

// setSessionPoolProvider passes the provider indirectly as required by the MS
// x64 ABI.
func setSessionPoolProvider(builder uintptr, provider ffiProxyProvider) int32 {
	heap, unpin := ffiPinnedCopy(provider)
	status := ffi_lk_session_pool_builder_set_provider(builder, uintptr(unsafe.Pointer(heap)))
	unpin()
	return status
}

// setClientBuilderDNSResolver passes a pointer to a pinned heap copy of the
// resolver to the client builder.
func setClientBuilderDNSResolver(builder uintptr, resolver ffiDNSResolver) int32 {
	heap, unpin := ffiPinnedCopy(resolver)
	status := ffi_lk_client_builder_set_dns_resolver(builder, uintptr(unsafe.Pointer(heap)))
	unpin()
	return status
}
