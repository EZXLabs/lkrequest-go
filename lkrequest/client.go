package lkrequest

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

type Client struct {
	ptr  uintptr
	once sync.Once
}

type ClientBuilder struct {
	ptr      uintptr
	firstErr error
}

func newClientFromPtr(ptr uintptr) *Client {
	if ptr == 0 {
		return nil
	}

	client := &Client{ptr: ptr}
	runtime.SetFinalizer(client, (*Client).Close)
	return client
}

func (c *Client) loadPtr() uintptr {
	if c == nil {
		return 0
	}
	return atomic.LoadUintptr(&c.ptr)
}

func NewClient(preset string) (*Client, error) {
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	presetPtr, presetBuf := stringToCString(preset)
	status := ffi_lk_client_new(
		presetPtr,
		uintptr(unsafe.Pointer(&cell[0])),
		uintptr(unsafe.Pointer(&cell[1])),
	)
	outClient, outErr := cell[0], cell[1]
	runtime.KeepAlive(presetBuf)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "client new"); err != nil {
		return nil, err
	}
	if outClient == 0 {
		return nil, nilHandleError("client new")
	}

	return newClientFromPtr(outClient), nil
}

func NewDefaultClient() (*Client, error) {
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_client_new_default(
		uintptr(unsafe.Pointer(&cell[0])),
		uintptr(unsafe.Pointer(&cell[1])),
	)
	outClient, outErr := cell[0], cell[1]

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "client new default"); err != nil {
		return nil, err
	}
	if outClient == 0 {
		// Some embedded builds do not expose a preset-backed default client even
		// though the plain builder path is available. Fall back so the public
		// default constructor still produces a usable client.
		return NewClientBuilder().Build()
	}

	return newClientFromPtr(outClient), nil
}

func (c *Client) Clone() *Client {
	p := c.loadPtr()
	if p == 0 {
		return nil
	}

	clone := newClientFromPtr(ffi_lk_client_clone(p))
	runtime.KeepAlive(c)
	return clone
}

func (c *Client) FingerprintInfoJSON() (string, error) {
	// Deferred rather than placed after the FFI call: the JSON pointer below is
	// borrowed from the client handle and is only read by goStringN at the very
	// end, so the client must stay reachable until this function returns.
	defer runtime.KeepAlive(c)

	p := c.loadPtr()
	if p == 0 {
		return "", ErrNilHandle
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_client_fingerprint_info_json(
		p,
		uintptr(unsafe.Pointer(&cell[0])),
		uintptr(unsafe.Pointer(&cell[1])),
		uintptr(unsafe.Pointer(&cell[2])),
	)
	outJSONPtr, outJSONLen, outErr := unsafe.Pointer(cell[0]), cell[1], cell[2]

	if err := extractError(outErr); err != nil {
		return "", err
	}
	if err := statusError(status, "client fingerprint info json"); err != nil {
		return "", err
	}
	if outJSONPtr == nil {
		return "", nilHandleError("client fingerprint info json")
	}

	return goStringN(outJSONPtr, outJSONLen), nil
}

func (c *Client) Close() {
	if c == nil {
		return
	}

	c.once.Do(func() {
		ptr := atomic.SwapUintptr(&c.ptr, 0)
		runtime.SetFinalizer(c, nil)
		if ptr != 0 {
			ffi_lk_client_free(ptr)
		}
	})
}

func NewClientBuilder() *ClientBuilder {
	builder := &ClientBuilder{ptr: ffi_lk_client_builder_new()}
	if builder.ptr == 0 {
		builder.firstErr = nilHandleError("client builder new")
	}

	runtime.SetFinalizer(builder, (*ClientBuilder).finalize)
	return builder
}

func (b *ClientBuilder) finalize() {
	b.release()
}

func (b *ClientBuilder) release() {
	if b == nil {
		return
	}

	ptr := b.ptr
	b.ptr = 0
	runtime.SetFinalizer(b, nil)
	if ptr != 0 {
		ffi_lk_client_builder_free(ptr)
	}
}

func (b *ClientBuilder) check() bool {
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

func (b *ClientBuilder) setStatus(status int32, action string) *ClientBuilder {
	if err := statusError(status, action); err != nil && b.firstErr == nil {
		b.firstErr = err
	}
	return b
}

func (b *ClientBuilder) SetPreset(name string) *ClientBuilder {
	if !b.check() {
		return b
	}

	namePtr, nameBuf := stringToCString(name)
	status := ffi_lk_client_builder_set_preset(b.ptr, namePtr)
	runtime.KeepAlive(nameBuf)
	return b.setStatus(status, "client builder set preset")
}

func (b *ClientBuilder) SetH2Preset(name string) *ClientBuilder {
	if !b.check() {
		return b
	}

	namePtr, nameBuf := stringToCString(name)
	status := ffi_lk_client_builder_set_h2_preset(b.ptr, namePtr)
	runtime.KeepAlive(nameBuf)
	return b.setStatus(status, "client builder set h2 preset")
}

func (b *ClientBuilder) SetVerify(enabled bool) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(
		ffi_lk_client_builder_set_verify(b.ptr, boolToUintptr(enabled)),
		"client builder set verify",
	)
}

func (b *ClientBuilder) SetTimeoutDNS(ms uint64) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(ffi_lk_client_builder_set_timeout_dns(b.ptr, ms), "client builder set timeout dns")
}

func (b *ClientBuilder) SetTimeoutTCPConnect(ms uint64) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(
		ffi_lk_client_builder_set_timeout_tcp_connect(b.ptr, ms),
		"client builder set timeout tcp connect",
	)
}

func (b *ClientBuilder) SetTimeoutTLSHandshake(ms uint64) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(
		ffi_lk_client_builder_set_timeout_tls_handshake(b.ptr, ms),
		"client builder set timeout tls handshake",
	)
}

func (b *ClientBuilder) SetTimeoutTTFB(ms uint64) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(ffi_lk_client_builder_set_timeout_ttfb(b.ptr, ms), "client builder set timeout ttfb")
}

func (b *ClientBuilder) SetTimeoutTotal(ms uint64) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(ffi_lk_client_builder_set_timeout_total(b.ptr, ms), "client builder set timeout total")
}

func (b *ClientBuilder) AddDefaultHeader(name, value string) *ClientBuilder {
	if !b.check() {
		return b
	}

	namePtr, nameBuf := stringToCString(name)
	valuePtr, valueBuf := stringToCString(value)
	status := ffi_lk_client_builder_add_default_header(
		b.ptr,
		namePtr,
		uintptr(len(name)),
		valuePtr,
		uintptr(len(value)),
	)
	runtime.KeepAlive(nameBuf)
	runtime.KeepAlive(valueBuf)
	return b.setStatus(status, "client builder add default header")
}

func (b *ClientBuilder) AddHeaderOrder(name string) *ClientBuilder {
	if !b.check() {
		return b
	}

	namePtr, nameBuf := stringToCString(name)
	status := ffi_lk_client_builder_add_header_order(b.ptr, namePtr, uintptr(len(name)))
	runtime.KeepAlive(nameBuf)
	return b.setStatus(status, "client builder add header order")
}

func (b *ClientBuilder) AddH3HeaderOrder(name string) *ClientBuilder {
	if !b.check() {
		return b
	}

	namePtr, nameBuf := stringToCString(name)
	status := ffi_lk_client_builder_add_h3_header_order(b.ptr, namePtr, uintptr(len(name)))
	runtime.KeepAlive(nameBuf)
	return b.setStatus(status, "client builder add h3 header order")
}

func (b *ClientBuilder) AddCookieOrder(name string) *ClientBuilder {
	if !b.check() {
		return b
	}

	namePtr, nameBuf := stringToCString(name)
	status := ffi_lk_client_builder_add_cookie_order(b.ptr, namePtr, uintptr(len(name)))
	runtime.KeepAlive(nameBuf)
	return b.setStatus(status, "client builder add cookie order")
}

func (b *ClientBuilder) AddCACertFile(path string) *ClientBuilder {
	if !b.check() {
		return b
	}

	pathPtr, pathBuf := stringToCString(path)
	status := ffi_lk_client_builder_add_ca_cert_file(b.ptr, pathPtr)
	runtime.KeepAlive(pathBuf)
	return b.setStatus(status, "client builder add ca cert file")
}

func (b *ClientBuilder) AddCACertMemory(data []byte) *ClientBuilder {
	if !b.check() {
		return b
	}

	status := ffi_lk_client_builder_add_ca_cert_memory(b.ptr, bytesPtr(data), uintptr(len(data)))
	runtime.KeepAlive(data)
	return b.setStatus(status, "client builder add ca cert memory")
}

func (b *ClientBuilder) SetECHConfig(data []byte) *ClientBuilder {
	if !b.check() {
		return b
	}

	status := ffi_lk_client_builder_set_ech_config(b.ptr, bytesPtr(data), uintptr(len(data)))
	runtime.KeepAlive(data)
	return b.setStatus(status, "client builder set ech config")
}

func (b *ClientBuilder) SetMaxOutstandingOps(n uint) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(
		ffi_lk_client_builder_set_max_outstanding_ops(b.ptr, uintptr(n)),
		"client builder set max outstanding ops",
	)
}

func (b *ClientBuilder) SetMaxResponseBodySize(n uint) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(
		ffi_lk_client_builder_set_max_response_body_size(b.ptr, uintptr(n)),
		"client builder set max response body size",
	)
}

func (b *ClientBuilder) SetMaxHeaderCount(n uint) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(
		ffi_lk_client_builder_set_max_header_count(b.ptr, uintptr(n)),
		"client builder set max header count",
	)
}

func (b *ClientBuilder) SetMaxHeaderSize(n uint) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(
		ffi_lk_client_builder_set_max_header_size(b.ptr, uintptr(n)),
		"client builder set max header size",
	)
}

func (b *ClientBuilder) SetMaxHeadersTotalSize(n uint) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(
		ffi_lk_client_builder_set_max_headers_total_size(b.ptr, uintptr(n)),
		"client builder set max headers total size",
	)
}

func (b *ClientBuilder) SetMinTransferRate(bytesPerSec uint, windowMs uint64) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(
		ffi_lk_client_builder_set_min_transfer_rate(b.ptr, uintptr(bytesPerSec), windowMs),
		"client builder set min transfer rate",
	)
}

func (b *ClientBuilder) SetMaxConnectionsPerSession(n uint) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(
		ffi_lk_client_builder_set_max_connections_per_session(b.ptr, uintptr(n)),
		"client builder set max connections per session",
	)
}

func (b *ClientBuilder) SetTCPFingerprintJA4T(ja4t string) *ClientBuilder {
	if !b.check() {
		return b
	}

	ja4tPtr, ja4tBuf := stringToCString(ja4t)
	status := ffi_lk_client_builder_set_tcp_fingerprint_ja4t(b.ptr, ja4tPtr, uintptr(len(ja4t)))
	runtime.KeepAlive(ja4tBuf)
	return b.setStatus(status, "client builder set tcp fingerprint ja4t")
}

func (b *ClientBuilder) SetKeylogFile(path string) *ClientBuilder {
	if !b.check() {
		return b
	}

	pathPtr, pathBuf := stringToCString(path)
	status := ffi_lk_client_builder_set_keylog_file(b.ptr, pathPtr, uintptr(len(path)))
	runtime.KeepAlive(pathBuf)
	return b.setStatus(status, "client builder set keylog file")
}

func (b *ClientBuilder) SetFallbackH2ToH1(enabled bool) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(
		ffi_lk_client_builder_set_fallback_h2_to_h1(b.ptr, boolToUintptr(enabled)),
		"client builder set fallback h2 to h1",
	)
}

func (b *ClientBuilder) SetFallbackProxyToDirect(enabled bool) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(
		ffi_lk_client_builder_set_fallback_proxy_to_direct(b.ptr, boolToUintptr(enabled)),
		"client builder set fallback proxy to direct",
	)
}

func (b *ClientBuilder) SetRetryOnConnectionClose(enabled bool) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(
		ffi_lk_client_builder_set_retry_on_connection_close(b.ptr, boolToUintptr(enabled)),
		"client builder set retry on connection close",
	)
}

func (b *ClientBuilder) SetDNS(config DnsConfig) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(
		ffi_lk_client_builder_set_dns(b.ptr, int32(config)),
		"client builder set dns",
	)
}

func (b *ClientBuilder) SetDNSCustom(addr string) *ClientBuilder {
	if !b.check() {
		return b
	}

	addrPtr, addrBuf := stringToCString(addr)
	status := ffi_lk_client_builder_set_dns_custom(b.ptr, addrPtr, uintptr(len(addr)))
	runtime.KeepAlive(addrBuf)
	return b.setStatus(status, "client builder set dns custom")
}

func (b *ClientBuilder) SetUseNativeCerts(enabled bool) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(
		ffi_lk_client_builder_set_use_native_certs(b.ptr, boolToUintptr(enabled)),
		"client builder set use native certs",
	)
}

// setUnsupported records firstErr when an optional FFI symbol is missing from
// the loaded library, keeping the builder chainable.
func (b *ClientBuilder) setUnsupported(action string) *ClientBuilder {
	if b.firstErr == nil {
		b.firstErr = unsupportedError(action)
	}
	return b
}

// DisableHTTP3 opts the client out of QUIC/HTTP3, even when a fingerprint
// preset would otherwise advertise it. Requires a QUIC/H3-capable library.
func (b *ClientBuilder) DisableHTTP3() *ClientBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_client_builder_disable_http3 == nil {
		return b.setUnsupported("client builder disable http3")
	}
	return b.setStatus(ffi_lk_client_builder_disable_http3(b.ptr), "client builder disable http3")
}

// SetTimeoutQUICConnect sets the QUIC connection (handshake) timeout in
// milliseconds. Requires a QUIC/H3-capable library.
func (b *ClientBuilder) SetTimeoutQUICConnect(ms uint64) *ClientBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_client_builder_set_timeout_quic_connect == nil {
		return b.setUnsupported("client builder set timeout quic connect")
	}
	return b.setStatus(
		ffi_lk_client_builder_set_timeout_quic_connect(b.ptr, ms),
		"client builder set timeout quic connect",
	)
}

// SetQUICProfileJSON applies a QUIC transport-parameter fingerprint profile,
// encoded as JSON. Requires a QUIC/H3-capable library.
func (b *ClientBuilder) SetQUICProfileJSON(profileJSON string) *ClientBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_client_builder_set_quic_profile_json == nil {
		return b.setUnsupported("client builder set quic profile json")
	}

	jsonPtr, jsonBuf := stringToCString(profileJSON)
	status := ffi_lk_client_builder_set_quic_profile_json(b.ptr, jsonPtr, uintptr(len(profileJSON)))
	runtime.KeepAlive(jsonBuf)
	return b.setStatus(status, "client builder set quic profile json")
}

// SetSessionResumptionJSON configures TLS/QUIC session resumption behavior,
// encoded as JSON. Requires a QUIC/H3-capable library.
func (b *ClientBuilder) SetSessionResumptionJSON(configJSON string) *ClientBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_client_builder_set_session_resumption_json == nil {
		return b.setUnsupported("client builder set session resumption json")
	}

	jsonPtr, jsonBuf := stringToCString(configJSON)
	status := ffi_lk_client_builder_set_session_resumption_json(b.ptr, jsonPtr, uintptr(len(configJSON)))
	runtime.KeepAlive(jsonBuf)
	return b.setStatus(status, "client builder set session resumption json")
}

// SetRandomize selects the fingerprint randomization tier. RandomizeOff and
// RandomizeExtensionOrder are always available; RandomizeRecombine and
// RandomizeFull are synthetic modes that require a library built with the
// synthetic-fp feature (otherwise the call records an error). layers is a
// FingerprintLayers bitmask choosing which layers the synthetic modes vary;
// FingerprintLayersAll (the zero value) means all layers, the safe default.
// layers is ignored for RandomizeOff and RandomizeExtensionOrder. Requires a
// library exporting lk_client_builder_set_randomize.
func (b *ClientBuilder) SetRandomize(mode RandomizeMode, layers FingerprintLayers) *ClientBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_client_builder_set_randomize == nil {
		return b.setUnsupported("client builder set randomize")
	}
	return b.setStatus(
		ffi_lk_client_builder_set_randomize(b.ptr, int32(mode), uint32(layers)),
		"client builder set randomize",
	)
}

func (b *ClientBuilder) Build() (*Client, error) {
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
	status := ffi_lk_client_builder_build(
		b.ptr,
		uintptr(unsafe.Pointer(&cell[0])),
		uintptr(unsafe.Pointer(&cell[1])),
	)
	outClient, outErr := cell[0], cell[1]
	ffiOutCellPool.Put(cell)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "client builder build"); err != nil {
		return nil, err
	}
	if outClient == 0 {
		return nil, nilHandleError("client builder build")
	}

	return newClientFromPtr(outClient), nil
}
