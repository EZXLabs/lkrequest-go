package lkrequest

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

// MasqueConfig configures the outer hop of masque:// routes — the HTTP/3
// connection to the MASQUE proxy itself: its trust anchors, certificate name,
// and tunnel reclamation. None of it touches the inner connection, whose
// browser fingerprint is what the origin sees.
//
// The outer hop's trust settings are independent of the client's own: neither
// ClientBuilder.SetVerify nor the client's CA certificates apply to it, and
// nothing set here applies to the origin.
//
// MASQUE needs a library built with the masque feature; NewMasqueConfig
// reports an error otherwise. Setters chain and the first failure is reported
// by the builder the config is handed to. Builders copy the settings, so one
// MasqueConfig can feed a client and a pool probe and must still be closed
// afterwards. A MasqueConfig is not safe for concurrent use.
type MasqueConfig struct {
	ptr      uintptr
	firstErr error
	once     sync.Once
}

// NewMasqueConfig creates outer-hop settings at their defaults: verify the
// proxy against the operating system's trust store, take the certificate name
// from the proxy URL, retire a tunnel idle for 120s, and carry at most 256
// tunnels per outer connection. A public-CA proxy needs no setter at all.
func NewMasqueConfig() (*MasqueConfig, error) {
	if ffi_lk_masque_config_new == nil {
		return nil, unsupportedError("masque config")
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	// The only refusal Go can trigger is a library built without MASQUE, and
	// the library does not say why, so the cause is spelled out here.
	if status := ffi_lk_masque_config_new(uintptr(unsafe.Pointer(&cell[0]))); status != ffiStatusOK {
		return nil, fmt.Errorf("lk: MASQUE is not supported by the loaded lkrequest library: it was built without the masque feature (status=%d)", status)
	}
	if cell[0] == 0 {
		return nil, nilHandleError("masque config new")
	}

	c := &MasqueConfig{ptr: cell[0]}
	runtime.SetFinalizer(c, (*MasqueConfig).Close)
	return c, nil
}

func (c *MasqueConfig) check() bool {
	if c == nil {
		return false
	}
	if c.firstErr != nil {
		return false
	}
	if c.ptr == 0 {
		c.firstErr = ErrNilHandle
		return false
	}
	return true
}

func (c *MasqueConfig) setStatus(status int32, action string) *MasqueConfig {
	if err := statusError(status, action); err != nil && c.firstErr == nil {
		c.firstErr = err
	}
	return c
}

// handle returns the native pointer for a consumer, or the error that makes
// the config unusable.
func (c *MasqueConfig) handle() (uintptr, error) {
	if c == nil {
		return 0, ErrNilHandle
	}
	if c.firstErr != nil {
		return 0, c.firstErr
	}
	if c.ptr == 0 {
		return 0, ErrNilHandle
	}
	return c.ptr, nil
}

// AddCACertsPEM trusts every certificate in a PEM bundle for the hop to the
// proxy — the usual form a private CA comes in. Non-certificate sections are
// skipped; a malformed bundle, or one with no certificate at all, is an error.
// This takes the PEM bytes, not a path.
func (c *MasqueConfig) AddCACertsPEM(pem []byte) *MasqueConfig {
	if !c.check() {
		return c
	}
	status := ffi_lk_masque_config_add_ca_certs_pem(c.ptr, bytesPtr(pem), uintptr(len(pem)), 0)
	runtime.KeepAlive(pem)
	return c.setStatus(status, "masque config add ca certs pem")
}

// AddCACertDER trusts one DER-encoded certificate for the hop to the proxy.
// The bytes are parsed when the outer connection is set up, so invalid DER
// surfaces on the first request or probe; prefer AddCACertsPEM to validate
// up front.
func (c *MasqueConfig) AddCACertDER(der []byte) *MasqueConfig {
	if !c.check() {
		return c
	}
	status := ffi_lk_masque_config_add_ca_cert_der(c.ptr, bytesPtr(der), uintptr(len(der)))
	runtime.KeepAlive(der)
	return c.setStatus(status, "masque config add ca cert der")
}

// SetUseNativeCerts controls whether the operating system's trust store is
// used for the hop to the proxy (default: true). Disable it for a proxy behind
// a private CA so that only the anchors added above can vouch for it.
func (c *MasqueConfig) SetUseNativeCerts(enabled bool) *MasqueConfig {
	if !c.check() {
		return c
	}
	return c.setStatus(
		ffi_lk_masque_config_set_use_native_certs(c.ptr, boolToUintptr(enabled)),
		"masque config set use native certs",
	)
}

// SetServerName overrides the SNI and the certificate name checked on the hop
// to the proxy, for a proxy addressed by something its certificate does not
// name (typically an IP address). An empty name restores the default.
func (c *MasqueConfig) SetServerName(name string) *MasqueConfig {
	if !c.check() {
		return c
	}

	namePtr, nameBuf := optionalStringToCString(name)
	status := ffi_lk_masque_config_set_server_name(c.ptr, namePtr, uintptr(len(name)))
	runtime.KeepAlive(nameBuf)
	return c.setStatus(status, "masque config set server name")
}

// SetTunnelIdleTimeout retires a tunnel that has carried nothing for ms
// milliseconds (default: 120000). 0 disables idle reclamation, leaving
// SetMaxIdleTunnels as the only bound on growth.
func (c *MasqueConfig) SetTunnelIdleTimeout(ms uint64) *MasqueConfig {
	if !c.check() {
		return c
	}
	return c.setStatus(
		ffi_lk_masque_config_set_tunnel_idle_timeout(c.ptr, ms),
		"masque config set tunnel idle timeout",
	)
}

// SetMaxIdleTunnels caps the tunnels one outer connection carries, shedding
// the least recently active first (default: 256). 0 removes the cap.
func (c *MasqueConfig) SetMaxIdleTunnels(n uint) *MasqueConfig {
	if !c.check() {
		return c
	}
	return c.setStatus(
		ffi_lk_masque_config_set_max_idle_tunnels(c.ptr, uintptr(n)),
		"masque config set max idle tunnels",
	)
}

// SetVerify controls certificate verification on the hop to the proxy
// (default: true). Disabling it is for development only: whatever answers on
// the proxy's address then sees every CONNECT-UDP target, which is exactly
// what the tunnel exists to hide. Reach for AddCACertsPEM first.
func (c *MasqueConfig) SetVerify(enabled bool) *MasqueConfig {
	if !c.check() {
		return c
	}
	return c.setStatus(
		ffi_lk_masque_config_set_verify(c.ptr, boolToUintptr(enabled)),
		"masque config set verify",
	)
}

// Close frees the settings. Builders they were handed to keep their copy.
func (c *MasqueConfig) Close() {
	if c == nil {
		return
	}

	c.once.Do(func() {
		ptr := c.ptr
		c.ptr = 0
		runtime.SetFinalizer(c, nil)
		if ptr != 0 {
			ffi_lk_masque_config_free(ptr)
		}
	})
}

// SetMasqueConfig configures the outer hop of masque:// routes for clients
// built from this builder. Calling it again replaces the previous settings.
// Leaving it unset keeps the defaults described on NewMasqueConfig.
func (b *ClientBuilder) SetMasqueConfig(config *MasqueConfig) *ClientBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_client_builder_set_masque_config == nil {
		return b.setUnsupported("client builder set masque config")
	}

	ptr, err := config.handle()
	if err != nil {
		if b.firstErr == nil {
			b.firstErr = err
		}
		return b
	}
	status := ffi_lk_client_builder_set_masque_config(b.ptr, ptr)
	runtime.KeepAlive(config)
	return b.setStatus(status, "client builder set masque config")
}

// SetMasqueTunnelProbe health-checks masque:// proxies with a real CONNECT-UDP
// handshake, verified against config, instead of skipping them — a TCP connect
// proves nothing about a MASQUE proxy. Pass the config the client uses so the
// probe judges the proxy the way requests will. Requires SetHealthCheck, in
// either order; Build fails without it.
func (b *ProxyPoolBuilder) SetMasqueTunnelProbe(config *MasqueConfig) *ProxyPoolBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_proxy_pool_builder_set_masque_tunnel_probe == nil {
		if b.firstErr == nil {
			b.firstErr = unsupportedError("proxy pool builder set masque tunnel probe")
		}
		return b
	}

	ptr, err := config.handle()
	if err != nil {
		if b.firstErr == nil {
			b.firstErr = err
		}
		return b
	}
	status := ffi_lk_proxy_pool_builder_set_masque_tunnel_probe(b.ptr, ptr)
	runtime.KeepAlive(config)
	return b.setStatus(status, "proxy pool builder set masque tunnel probe")
}

// SetMasqueTunnelProbe health-checks masque:// proxies with a real CONNECT-UDP
// handshake; see ProxyPoolBuilder.SetMasqueTunnelProbe.
func (b *SessionPoolBuilder) SetMasqueTunnelProbe(config *MasqueConfig) *SessionPoolBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_session_pool_builder_set_masque_tunnel_probe == nil {
		if b.firstErr == nil {
			b.firstErr = unsupportedError("session pool builder set masque tunnel probe")
		}
		return b
	}

	ptr, err := config.handle()
	if err != nil {
		if b.firstErr == nil {
			b.firstErr = err
		}
		return b
	}
	status := ffi_lk_session_pool_builder_set_masque_tunnel_probe(b.ptr, ptr)
	runtime.KeepAlive(config)
	return b.setStatus(status, "session pool builder set masque tunnel probe")
}
