package lkrequest

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

// ProxyAuthHeader selects which request header carries HTTP-based proxy
// credentials. There is no interoperable answer: many deployments read
// Proxy-Authorization, while masque-go and RFC 9484 §11 use Authorization.
type ProxyAuthHeader int32

const (
	// ProxyAuthHeaderProxyAuthorization sends credentials in
	// Proxy-Authorization. This is the default.
	ProxyAuthHeaderProxyAuthorization ProxyAuthHeader = 0
	// ProxyAuthHeaderAuthorization sends credentials in Authorization.
	ProxyAuthHeaderAuthorization ProxyAuthHeader = 1
)

// ProxyConfig is a parsed proxy that can carry credentials a proxy URL cannot
// express: a static HTTP auth-scheme with already-encoded credentials
// (Preshared, PrivateToken, Bearer, ...) and the header that carries them.
//
// Setters chain; the first failure is kept and reported by URL, Identity, or
// the builder the config is handed to. Builders copy the configuration, so one
// ProxyConfig can feed several builders and must still be closed afterwards.
// A ProxyConfig is not safe for concurrent use.
type ProxyConfig struct {
	ptr      uintptr
	firstErr error
	once     sync.Once
}

// ParseProxyConfig parses a proxy URL (http, https, socks5, socks5h, and
// masque when the loaded library supports MASQUE) into a ProxyConfig.
func ParseProxyConfig(url string) (*ProxyConfig, error) {
	if ffi_lk_proxy_config_parse == nil {
		return nil, unsupportedError("proxy config parse")
	}

	urlPtr, urlBuf := stringToCString(url)
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_proxy_config_parse(urlPtr, uintptr(len(url)), uintptr(unsafe.Pointer(&cell[0])))
	runtime.KeepAlive(urlBuf)
	if err := statusError(status, "proxy config parse"); err != nil {
		return nil, err
	}
	if cell[0] == 0 {
		return nil, nilHandleError("proxy config parse")
	}

	c := &ProxyConfig{ptr: cell[0]}
	runtime.SetFinalizer(c, (*ProxyConfig).Close)
	return c, nil
}

func (c *ProxyConfig) check() bool {
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

func (c *ProxyConfig) setStatus(status int32, action string) *ProxyConfig {
	if err := statusError(status, action); err != nil && c.firstErr == nil {
		c.firstErr = err
	}
	return c
}

// handle returns the native pointer for a consumer, or the error that makes
// the config unusable.
func (c *ProxyConfig) handle() (uintptr, error) {
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

// SetUserPass replaces the credentials with a username/password pair, without
// the URL-encoding a proxy URL would need.
func (c *ProxyConfig) SetUserPass(username, password string) *ProxyConfig {
	if !c.check() {
		return c
	}

	userPtr, userBuf := stringToCString(username)
	passPtr, passBuf := stringToCString(password)
	status := ffi_lk_proxy_config_set_user_pass(
		c.ptr,
		userPtr, uintptr(len(username)),
		passPtr, uintptr(len(password)),
	)
	runtime.KeepAlive(userBuf)
	runtime.KeepAlive(passBuf)
	return c.setStatus(status, "proxy config set user pass")
}

// SetHTTPAuth authenticates with a static HTTP auth-scheme and
// already-encoded credentials, e.g. ("Preshared", "<psk>"). Fails for a SOCKS5
// proxy, whose handshake has no request headers to carry it.
func (c *ProxyConfig) SetHTTPAuth(scheme, credentials string) *ProxyConfig {
	if !c.check() {
		return c
	}

	schemePtr, schemeBuf := stringToCString(scheme)
	credPtr, credBuf := stringToCString(credentials)
	status := ffi_lk_proxy_config_set_http_auth(
		c.ptr,
		schemePtr, uintptr(len(scheme)),
		credPtr, uintptr(len(credentials)),
	)
	runtime.KeepAlive(schemeBuf)
	runtime.KeepAlive(credBuf)
	return c.setStatus(status, "proxy config set http auth")
}

// SetAuthHeader selects the header that carries HTTP-based credentials.
func (c *ProxyConfig) SetAuthHeader(header ProxyAuthHeader) *ProxyConfig {
	if !c.check() {
		return c
	}
	// The library takes this as a Rust enum by value, so an out-of-range value
	// would be undefined behavior on the far side rather than an error.
	if header != ProxyAuthHeaderProxyAuthorization && header != ProxyAuthHeaderAuthorization {
		c.firstErr = fmt.Errorf("lk: invalid proxy auth header %d", header)
		return c
	}
	return c.setStatus(ffi_lk_proxy_config_set_auth_header(c.ptr, int32(header)), "proxy config set auth header")
}

// URL renders the proxy as scheme://[user:pass@]host:port. Credentials set
// with SetHTTPAuth have no URL form and do not appear.
func (c *ProxyConfig) URL() (string, error) {
	return c.borrowString(ffi_lk_proxy_config_url, "proxy config url")
}

// Identity returns the key the library uses for this proxy's cooldown and
// blacklisting — the value ProxyPool.MarkBad expects. It is opaque and never
// contains passwords or encoded credentials.
func (c *ProxyConfig) Identity() (string, error) {
	return c.borrowString(ffi_lk_proxy_config_identity, "proxy config identity")
}

// borrowString copies a string the library lends out of the handle; the
// borrow ends at the next mutation or free, so it is copied immediately.
func (c *ProxyConfig) borrowString(fn func(config uintptr, outPtr uintptr, outLen uintptr) int32, action string) (string, error) {
	defer runtime.KeepAlive(c)
	ptr, err := c.handle()
	if err != nil {
		return "", err
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	if err := statusError(fn(ptr, uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1]))), action); err != nil {
		return "", err
	}
	return goStringN(unsafe.Pointer(cell[0]), cell[1]), nil
}

// Close frees the configuration. Builders it was handed to keep their copy.
func (c *ProxyConfig) Close() {
	if c == nil {
		return
	}

	c.once.Do(func() {
		ptr := c.ptr
		c.ptr = 0
		runtime.SetFinalizer(c, nil)
		if ptr != 0 {
			ffi_lk_proxy_config_free(ptr)
		}
	})
}

// SetProxyConfig routes the session through a proxy built with
// ParseProxyConfig. It takes precedence over SetProxy. The configuration is
// copied, so config can be closed or reused afterwards.
func (b *SessionBuilder) SetProxyConfig(config *ProxyConfig) *SessionBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_session_builder_set_proxy_config == nil {
		if b.firstErr == nil {
			b.firstErr = unsupportedError("session builder set proxy config")
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
	status := ffi_lk_session_builder_set_proxy_config(b.ptr, ptr)
	runtime.KeepAlive(config)
	return b.setStatus(status, "session builder set proxy config")
}

// AddProxyConfig adds a proxy built with ParseProxyConfig. The configuration
// is copied, so config can be closed or reused afterwards.
func (b *ProxyPoolBuilder) AddProxyConfig(config *ProxyConfig) *ProxyPoolBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_proxy_pool_builder_add_proxy_config == nil {
		if b.firstErr == nil {
			b.firstErr = unsupportedError("proxy pool builder add proxy config")
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
	status := ffi_lk_proxy_pool_builder_add_proxy_config(b.ptr, ptr)
	runtime.KeepAlive(config)
	return b.setStatus(status, "proxy pool builder add proxy config")
}
