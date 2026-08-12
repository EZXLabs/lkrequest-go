//go:build !lkcgo && !windows

package lkrequest

import "github.com/ebitengine/purego"

// These are the provider-related FFI bindings for non-Windows purego builds.
//
// The C signatures of lk_*_builder_set_provider take lk_proxy_provider_t,
// represented by ffiProxyProvider, by value. purego supports by-value struct
// arguments on linux/amd64 (SysV) and arm64, so these declarations mirror the
// C signatures and let purego apply the platform calling convention. Windows
// requires a separate representation; see provider_purego_windows.go.
var (
	ffi_lk_proxy_pool_builder_set_provider   func(builder uintptr, provider ffiProxyProvider) int32
	ffi_lk_session_pool_builder_set_provider func(builder uintptr, provider ffiProxyProvider) int32
	// lk_client_builder_set_dns_resolver also takes a by-value struct:
	// lk_dns_resolver_t, represented by ffiDNSResolver. It was previously
	// declared as a pointer, which mismatched the linux/amd64 SysV ABI because
	// structs larger than 16 bytes are passed on the stack. This symbol is
	// optional for older libraries, so a missing export leaves it nil and
	// SetDNSResolver reports the feature as unsupported.
	ffi_lk_client_builder_set_dns_resolver func(builder uintptr, resolver ffiDNSResolver) int32
)

// registerProviderFuncs registers the platform-specific by-value struct
// symbols for the two provider setters and the DNS resolver.
func registerProviderFuncs(handle uintptr) {
	purego.RegisterLibFunc(&ffi_lk_proxy_pool_builder_set_provider, handle, "lk_proxy_pool_builder_set_provider")
	purego.RegisterLibFunc(&ffi_lk_session_pool_builder_set_provider, handle, "lk_session_pool_builder_set_provider")
	registerOptional(&ffi_lk_client_builder_set_dns_resolver, handle, "lk_client_builder_set_dns_resolver")
}

// setProxyPoolProvider passes a provider to the proxy pool builder.
func setProxyPoolProvider(builder uintptr, provider ffiProxyProvider) int32 {
	return ffi_lk_proxy_pool_builder_set_provider(builder, provider)
}

// setSessionPoolProvider passes a provider to the session pool builder.
func setSessionPoolProvider(builder uintptr, provider ffiProxyProvider) int32 {
	return ffi_lk_session_pool_builder_set_provider(builder, provider)
}

// setClientBuilderDNSResolver passes a resolver to the client builder by value.
func setClientBuilderDNSResolver(builder uintptr, resolver ffiDNSResolver) int32 {
	return ffi_lk_client_builder_set_dns_resolver(builder, resolver)
}
