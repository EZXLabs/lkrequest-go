package lkrequest

// SetTLSSessionCacheEnabled turns the client's TLS and QUIC session cache on
// or off; it is on by default. With the cache off every new connection makes a
// full handshake, while pooled connections and the extensions the profile
// advertises stay as they are. Requires a library exporting
// lk_client_builder_set_tls_session_cache_enabled.
func (b *ClientBuilder) SetTLSSessionCacheEnabled(enabled bool) *ClientBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_client_builder_set_tls_session_cache_enabled == nil {
		return b.setUnsupported("client builder set tls session cache enabled")
	}
	return b.setStatus(
		ffi_lk_client_builder_set_tls_session_cache_enabled(b.ptr, boolToUintptr(enabled)),
		"client builder set tls session cache enabled",
	)
}

// SetTLSSessionCacheTTL limits, in milliseconds, how long a cached TLS or QUIC
// session may be resumed; 0 keeps sessions out of the cache. A shorter lifetime
// the server sets and TLS 1.3's seven-day limit still apply, and a TLS 1.2
// session keeps the age of its original secret. It does not turn a disabled
// cache back on. Requires a library exporting
// lk_client_builder_set_tls_session_cache_ttl.
func (b *ClientBuilder) SetTLSSessionCacheTTL(ms uint64) *ClientBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_client_builder_set_tls_session_cache_ttl == nil {
		return b.setUnsupported("client builder set tls session cache ttl")
	}
	return b.setStatus(
		ffi_lk_client_builder_set_tls_session_cache_ttl(b.ptr, ms),
		"client builder set tls session cache ttl",
	)
}

// ResetTLSSessionCacheTTL restores the default session lifetimes, one hour for
// TLS 1.2 and up to seven days for TLS 1.3 and QUIC, leaving the cache on or
// off as it was. Requires a library exporting
// lk_client_builder_reset_tls_session_cache_ttl.
func (b *ClientBuilder) ResetTLSSessionCacheTTL() *ClientBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_client_builder_reset_tls_session_cache_ttl == nil {
		return b.setUnsupported("client builder reset tls session cache ttl")
	}
	return b.setStatus(
		ffi_lk_client_builder_reset_tls_session_cache_ttl(b.ptr),
		"client builder reset tls session cache ttl",
	)
}
