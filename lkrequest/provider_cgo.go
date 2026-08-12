//go:build lkcgo

package lkrequest

import "unsafe"

// These are the provider-related FFI bindings for the cgo engine.
//
// The variables are declared here and assigned by init in binding_cgo.go.
// Their C wrappers let the C compiler apply each platform's ABI correctly to
// by-value structs.
var (
	ffi_lk_proxy_pool_builder_set_provider   func(builder uintptr, provider ffiProxyProvider) int32
	ffi_lk_session_pool_builder_set_provider func(builder uintptr, provider ffiProxyProvider) int32
	ffi_lk_client_builder_set_dns_resolver   func(builder uintptr, resolver uintptr) int32
)

// setProxyPoolProvider passes a provider to the proxy pool builder.
func setProxyPoolProvider(builder uintptr, provider ffiProxyProvider) int32 {
	return ffi_lk_proxy_pool_builder_set_provider(builder, provider)
}

// setSessionPoolProvider passes a provider to the session pool builder.
func setSessionPoolProvider(builder uintptr, provider ffiProxyProvider) int32 {
	return ffi_lk_session_pool_builder_set_provider(builder, provider)
}

// setClientBuilderDNSResolver passes a resolver to the client builder.
// The cgo wrapper in binding_cgo.go immediately dereferences the pointer and
// passes the value to C, leaving no callback re-entry window while the local
// address is exposed.
func setClientBuilderDNSResolver(builder uintptr, resolver ffiDNSResolver) int32 {
	return ffi_lk_client_builder_set_dns_resolver(builder, uintptr(unsafe.Pointer(&resolver)))
}
