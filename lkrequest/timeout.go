package lkrequest

// SetTimeoutReadIdle limits, in milliseconds, how long each read of a
// response body may wait for data, buffered and streaming alike. It is off by
// default and leaves the total deadline as it is; 0 times out at once, and
// DisableTimeoutReadIdle turns it off again. Requires a library exporting
// lk_client_builder_set_timeout_read_idle.
func (b *ClientBuilder) SetTimeoutReadIdle(ms uint64) *ClientBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_client_builder_set_timeout_read_idle == nil {
		return b.setUnsupported("client builder set timeout read idle")
	}
	return b.setStatus(
		ffi_lk_client_builder_set_timeout_read_idle(b.ptr, ms),
		"client builder set timeout read idle",
	)
}

// DisableTimeoutDNS removes the client's DNS deadline; SetTimeoutDNS sets it
// again. A custom resolver's own limits still apply. Requires a library
// exporting lk_client_builder_disable_timeout_dns.
func (b *ClientBuilder) DisableTimeoutDNS() *ClientBuilder {
	return b.disableTimeout(ffi_lk_client_builder_disable_timeout_dns, "client builder disable timeout dns")
}

// DisableTimeoutTCPConnect removes the deadline for connecting to the server
// or the proxy; SetTimeoutTCPConnect sets it again. Requires a library
// exporting lk_client_builder_disable_timeout_tcp_connect.
func (b *ClientBuilder) DisableTimeoutTCPConnect() *ClientBuilder {
	return b.disableTimeout(ffi_lk_client_builder_disable_timeout_tcp_connect, "client builder disable timeout tcp connect")
}

// DisableTimeoutTLSHandshake removes the TLS handshake deadline;
// SetTimeoutTLSHandshake sets it again. Requires a library exporting
// lk_client_builder_disable_timeout_tls_handshake.
func (b *ClientBuilder) DisableTimeoutTLSHandshake() *ClientBuilder {
	return b.disableTimeout(ffi_lk_client_builder_disable_timeout_tls_handshake, "client builder disable timeout tls handshake")
}

// DisableTimeoutTTFB removes the deadline for the response headers, the time
// spent uploading the request body included; SetTimeoutTTFB sets it again.
// Requires a library exporting lk_client_builder_disable_timeout_ttfb.
func (b *ClientBuilder) DisableTimeoutTTFB() *ClientBuilder {
	return b.disableTimeout(ffi_lk_client_builder_disable_timeout_ttfb, "client builder disable timeout ttfb")
}

// DisableTimeoutTotal removes the overall deadline of a request, long-lived
// streaming responses included, while the phase timeouts stay in force.
// SetTimeoutTotal sets it again. Requires a library exporting
// lk_client_builder_disable_timeout_total.
func (b *ClientBuilder) DisableTimeoutTotal() *ClientBuilder {
	return b.disableTimeout(ffi_lk_client_builder_disable_timeout_total, "client builder disable timeout total")
}

// DisableTimeoutQUICConnect removes the deadlines of the QUIC handshake and of
// setting up HTTP/3; SetTimeoutQUICConnect sets them again. Requires a library
// exporting lk_client_builder_disable_timeout_quic_connect.
func (b *ClientBuilder) DisableTimeoutQUICConnect() *ClientBuilder {
	return b.disableTimeout(ffi_lk_client_builder_disable_timeout_quic_connect, "client builder disable timeout quic connect")
}

// DisableTimeoutReadIdle removes the timeout SetTimeoutReadIdle set, leaving
// the total deadline as it is. Requires a library exporting
// lk_client_builder_disable_timeout_read_idle.
func (b *ClientBuilder) DisableTimeoutReadIdle() *ClientBuilder {
	return b.disableTimeout(ffi_lk_client_builder_disable_timeout_read_idle, "client builder disable timeout read idle")
}

// disableTimeout calls disable, one of the library's setters that remove a
// deadline, recording it as unsupported when the library lacks it.
func (b *ClientBuilder) disableTimeout(disable func(builder uintptr) int32, action string) *ClientBuilder {
	if !b.check() {
		return b
	}
	if disable == nil {
		return b.setUnsupported(action)
	}
	return b.setStatus(disable(b.ptr), action)
}
