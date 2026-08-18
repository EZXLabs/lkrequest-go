package lkrequest

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// ConnectionPoolStats holds session connection pool statistics.
type ConnectionPoolStats struct {
	H3         uint
	H2         uint
	H1         uint
	Total      uint
	Max        uint
	AtCapacity bool
}

type Session struct {
	ptr    uintptr
	once   sync.Once
	client *Client
}

type SessionBuilder struct {
	ptr      uintptr
	firstErr error
	client   *Client
}

func newSessionFromPtr(ptr uintptr, client *Client) *Session {
	if ptr == 0 {
		return nil
	}

	session := &Session{
		ptr:    ptr,
		client: client,
	}
	runtime.SetFinalizer(session, (*Session).Close)
	return session
}

func (s *Session) loadPtr() uintptr {
	if s == nil {
		return 0
	}
	return atomic.LoadUintptr(&s.ptr)
}

func NewSession(client *Client) (*Session, error) {
	cp := client.loadPtr()
	if cp == 0 {
		return nil, ErrNilHandle
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_session_new(
		cp,
		uintptr(unsafe.Pointer(&cell[0])),
		uintptr(unsafe.Pointer(&cell[1])),
	)
	outSession, outErr := cell[0], cell[1]
	runtime.KeepAlive(client)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "session new"); err != nil {
		return nil, err
	}
	if outSession == 0 {
		return nil, nilHandleError("session new")
	}

	return newSessionFromPtr(outSession, client), nil
}

func NewSessionWithConfig(client *Client, proxy string, maxRedirects uint32) (*Session, error) {
	cp := client.loadPtr()
	if cp == 0 {
		return nil, ErrNilHandle
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	proxyPtr, proxyBuf := stringToCString(proxy)
	status := ffi_lk_session_new_with_config(
		cp,
		proxyPtr,
		uintptr(len(proxy)),
		maxRedirects,
		uintptr(unsafe.Pointer(&cell[0])),
		uintptr(unsafe.Pointer(&cell[1])),
	)
	outSession, outErr := cell[0], cell[1]
	runtime.KeepAlive(proxyBuf)
	runtime.KeepAlive(client)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "session new with config"); err != nil {
		return nil, err
	}
	if outSession == 0 {
		return nil, nilHandleError("session new with config")
	}

	return newSessionFromPtr(outSession, client), nil
}

func (s *Session) Clone() *Session {
	p := s.loadPtr()
	if p == 0 {
		return nil
	}

	clone := newSessionFromPtr(ffi_lk_session_clone(p), s.client)
	runtime.KeepAlive(s)
	return clone
}

func (s *Session) SetCookie(url, name, value string) error {
	p := s.loadPtr()
	if p == 0 {
		return ErrNilHandle
	}

	urlPtr, urlBuf := stringToCString(url)
	namePtr, nameBuf := stringToCString(name)
	valuePtr, valueBuf := stringToCString(value)
	status := ffi_lk_session_set_cookie(p, urlPtr, uintptr(len(url)), namePtr, uintptr(len(name)), valuePtr, uintptr(len(value)))
	runtime.KeepAlive(urlBuf)
	runtime.KeepAlive(nameBuf)
	runtime.KeepAlive(valueBuf)
	return statusError(status, "session set cookie")
}

func (s *Session) SetCookieWithAttrs(url, name, value string, attrs CookieAttrs) error {
	p := s.loadPtr()
	if p == 0 {
		return ErrNilHandle
	}

	urlPtr, urlBuf := stringToCString(url)
	namePtr, nameBuf := stringToCString(name)
	valuePtr, valueBuf := stringToCString(value)
	pathPtr, pathBuf := optionalStringToCString(attrs.Path)
	domainPtr, domainBuf := optionalStringToCString(attrs.Domain)
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_session_set_cookie_with_attrs(
		p,
		urlPtr,
		uintptr(len(url)),
		namePtr,
		uintptr(len(name)),
		valuePtr,
		uintptr(len(value)),
		pathPtr,
		uintptr(len(attrs.Path)),
		domainPtr,
		uintptr(len(attrs.Domain)),
		attrs.Secure,
		attrs.HTTPOnly,
		uintptr(unsafe.Pointer(&cell[0])),
	)
	outErr := cell[0]
	runtime.KeepAlive(urlBuf)
	runtime.KeepAlive(nameBuf)
	runtime.KeepAlive(valueBuf)
	runtime.KeepAlive(pathBuf)
	runtime.KeepAlive(domainBuf)

	if err := extractError(outErr); err != nil {
		return err
	}
	return statusError(status, "session set cookie with attrs")
}

func (s *Session) RemoveCookie(url, name string) error {
	p := s.loadPtr()
	if p == 0 {
		return ErrNilHandle
	}

	urlPtr, urlBuf := stringToCString(url)
	namePtr, nameBuf := stringToCString(name)
	status := ffi_lk_session_remove_cookie(p, urlPtr, uintptr(len(url)), namePtr, uintptr(len(name)))
	runtime.KeepAlive(urlBuf)
	runtime.KeepAlive(nameBuf)
	return statusError(status, "session remove cookie")
}

func (s *Session) GetCookie(url, name string) (string, error) {
	defer runtime.KeepAlive(s)
	p := s.loadPtr()
	if p == 0 {
		return "", ErrNilHandle
	}

	urlPtr, urlBuf := stringToCString(url)
	namePtr, nameBuf := stringToCString(name)
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_session_get_cookie(p, urlPtr, uintptr(len(url)), namePtr, uintptr(len(name)), uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])), uintptr(unsafe.Pointer(&cell[2])))
	outValuePtr, outValueLen, outErr := unsafe.Pointer(cell[0]), cell[1], cell[2]
	runtime.KeepAlive(urlBuf)
	runtime.KeepAlive(nameBuf)
	if err := extractError(outErr); err != nil {
		return "", err
	}
	if err := statusError(status, "session get cookie"); err != nil {
		return "", err
	}
	return goStringN(outValuePtr, outValueLen), nil
}

func (s *Session) GetCookiesJSON(url string) (string, error) {
	defer runtime.KeepAlive(s)
	p := s.loadPtr()
	if p == 0 {
		return "", ErrNilHandle
	}

	urlPtr, urlBuf := stringToCString(url)
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_session_get_cookies_json(p, urlPtr, uintptr(len(url)), uintptr(unsafe.Pointer(&cell[0])))
	runtime.KeepAlive(urlBuf)
	if err := statusError(status, "session get cookies json"); err != nil {
		return "", err
	}
	return goCString(unsafe.Pointer(cell[0])), nil
}

// GetCookies returns the session cookies that would be sent to url, decoded
// from the payload GetCookiesJSON returns. It is the structured counterpart to
// GetCookiesJSON, which stays available for callers that want to forward the
// raw document.
func (s *Session) GetCookies(url string) ([]Cookie, error) {
	raw, err := s.GetCookiesJSON(url)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}

	var cookies []Cookie
	if err := json.Unmarshal([]byte(raw), &cookies); err != nil {
		return nil, fmt.Errorf("lk: decode session cookies: %w", err)
	}
	return cookies, nil
}

func (s *Session) ClearCookies() error {
	p := s.loadPtr()
	if p == 0 {
		return ErrNilHandle
	}
	return statusError(ffi_lk_session_clear_cookies(p), "session clear cookies")
}

func (s *Session) Preconnect(url string) error {
	p := s.loadPtr()
	if p == 0 {
		return ErrNilHandle
	}

	urlPtr, urlBuf := stringToCString(url)
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_session_preconnect(p, urlPtr, uintptr(len(url)), uintptr(unsafe.Pointer(&cell[0])))
	outErr := cell[0]
	runtime.KeepAlive(urlBuf)
	runtime.KeepAlive(s)
	if err := extractError(outErr); err != nil {
		return err
	}
	return statusError(status, "session preconnect")
}

func (s *Session) PreconnectAsync(ctx context.Context, url string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	p := s.loadPtr()
	if p == 0 {
		return ErrNilHandle
	}

	urlPtr, urlBuf := stringToCString(url)
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_session_preconnect_async(p, urlPtr, uintptr(len(url)), uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])))
	outOp, outErr := cell[0], cell[1]
	runtime.KeepAlive(urlBuf)
	runtime.KeepAlive(s)
	if err := extractError(outErr); err != nil {
		return err
	}
	if err := statusError(status, "session preconnect async"); err != nil {
		return err
	}
	if outOp == 0 {
		return nilHandleError("session preconnect async")
	}
	defer ffi_lk_op_free(outOp)

	state, err := waitForOp(ctx, outOp)
	if err != nil {
		return err
	}

	switch state {
	case OpCompletedOK:
		return nil
	case OpCompletedErr:
		return takeOperationError(outOp, "session preconnect")
	case OpCancelled:
		return context.Canceled
	default:
		return fmt.Errorf("lk: unexpected op state %s", state)
	}
}

func (s *Session) ConnectionPoolStats() (ConnectionPoolStats, error) {
	var stats ConnectionPoolStats
	p := s.loadPtr()
	if p == 0 {
		return stats, ErrNilHandle
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_session_connection_pool_stats(p, uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])), uintptr(unsafe.Pointer(&cell[2])), uintptr(unsafe.Pointer(&cell[3])), uintptr(unsafe.Pointer(&cell[4])))
	h2, h1, total, max := cell[0], cell[1], cell[2], cell[3]
	atCapacity := cell[4] != 0
	runtime.KeepAlive(s)
	if err := statusError(status, "session connection pool stats"); err != nil {
		return stats, err
	}
	stats.H2 = uint(h2)
	stats.H1 = uint(h1)
	stats.Total = uint(total)
	if total >= h2+h1 {
		stats.H3 = uint(total - h2 - h1)
	}
	stats.Max = uint(max)
	stats.AtCapacity = atCapacity
	return stats, nil
}

func (s *Session) ConnectionPoolClear() error {
	p := s.loadPtr()
	if p == 0 {
		return ErrNilHandle
	}
	return statusError(ffi_lk_session_connection_pool_clear(p), "session connection pool clear")
}

func (s *Session) Close() {
	if s == nil {
		return
	}

	s.once.Do(func() {
		ptr := atomic.SwapUintptr(&s.ptr, 0)
		runtime.SetFinalizer(s, nil)
		if ptr != 0 {
			ffi_lk_session_free(ptr)
		}
	})
}

func NewSessionBuilder(client *Client) *SessionBuilder {
	builder := &SessionBuilder{client: client}
	cp := client.loadPtr()
	if cp == 0 {
		builder.firstErr = ErrNilHandle
		return builder
	}

	builder.ptr = ffi_lk_session_builder_new(cp)
	if builder.ptr == 0 {
		builder.firstErr = nilHandleError("session builder new")
	}

	runtime.SetFinalizer(builder, (*SessionBuilder).finalize)
	runtime.KeepAlive(client)
	return builder
}

func (b *SessionBuilder) finalize() {
	b.release()
}

func (b *SessionBuilder) release() {
	if b == nil {
		return
	}

	ptr := b.ptr
	b.ptr = 0
	runtime.SetFinalizer(b, nil)
	if ptr != 0 {
		ffi_lk_session_builder_free(ptr)
	}
}

func (b *SessionBuilder) check() bool {
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

func (b *SessionBuilder) setStatus(status int32, action string) *SessionBuilder {
	if err := statusError(status, action); err != nil && b.firstErr == nil {
		b.firstErr = err
	}
	return b
}

func (b *SessionBuilder) SetProxy(proxy string) *SessionBuilder {
	if !b.check() {
		return b
	}

	proxyPtr, proxyBuf := stringToCString(proxy)
	status := ffi_lk_session_builder_set_proxy(b.ptr, proxyPtr, uintptr(len(proxy)))
	runtime.KeepAlive(proxyBuf)
	return b.setStatus(status, "session builder set proxy")
}

func (b *SessionBuilder) AddHeaderOrder(name string) *SessionBuilder {
	if !b.check() {
		return b
	}

	namePtr, nameBuf := stringToCString(name)
	status := ffi_lk_session_builder_add_header_order(b.ptr, namePtr, uintptr(len(name)))
	runtime.KeepAlive(nameBuf)
	return b.setStatus(status, "session builder add header order")
}

func (b *SessionBuilder) AddH3HeaderOrder(name string) *SessionBuilder {
	if !b.check() {
		return b
	}

	namePtr, nameBuf := stringToCString(name)
	status := ffi_lk_session_builder_add_h3_header_order(b.ptr, namePtr, uintptr(len(name)))
	runtime.KeepAlive(nameBuf)
	return b.setStatus(status, "session builder add h3 header order")
}

func (b *SessionBuilder) AddCookieOrder(name string) *SessionBuilder {
	if !b.check() {
		return b
	}

	namePtr, nameBuf := stringToCString(name)
	status := ffi_lk_session_builder_add_cookie_order(b.ptr, namePtr, uintptr(len(name)))
	runtime.KeepAlive(nameBuf)
	return b.setStatus(status, "session builder add cookie order")
}

func (b *SessionBuilder) SetMaxRedirects(n uint32) *SessionBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(ffi_lk_session_builder_set_max_redirects(b.ptr, n), "session builder set max redirects")
}

func (b *SessionBuilder) DisableRedirects() *SessionBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(ffi_lk_session_builder_disable_redirects(b.ptr), "session builder disable redirects")
}

func (b *SessionBuilder) SetECHConfig(data []byte) *SessionBuilder {
	if !b.check() {
		return b
	}

	status := ffi_lk_session_builder_set_ech_config(b.ptr, bytesPtr(data), uintptr(len(data)))
	runtime.KeepAlive(data)
	return b.setStatus(status, "session builder set ech config")
}

func (b *SessionBuilder) SetHTTP1Only() *SessionBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(ffi_lk_session_builder_set_http1_only(b.ptr), "session builder set http1 only")
}

func (b *SessionBuilder) SetHTTP2Only() *SessionBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(ffi_lk_session_builder_set_http2_only(b.ptr), "session builder set http2 only")
}

func (b *SessionBuilder) SetHTTP3Only() *SessionBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(ffi_lk_session_builder_set_http3_only(b.ptr), "session builder set http3 only")
}

// SetHTTP3WithFallback prefers HTTP/3 but allows falling back to HTTP/2 or
// HTTP/1.1 when QUIC is unavailable. Requires a QUIC/H3-capable library.
func (b *SessionBuilder) SetHTTP3WithFallback() *SessionBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_session_builder_set_http3_with_fallback == nil {
		if b.firstErr == nil {
			b.firstErr = unsupportedError("session builder set http3 with fallback")
		}
		return b
	}
	return b.setStatus(
		ffi_lk_session_builder_set_http3_with_fallback(b.ptr),
		"session builder set http3 with fallback",
	)
}

func (b *SessionBuilder) SetDefaultAcceptEncoding(bits AcceptEncoding) *SessionBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(
		ffi_lk_session_builder_set_default_accept_encoding(b.ptr, uint8(bits)),
		"session builder set default accept encoding",
	)
}

func (b *SessionBuilder) SetMaxConnections(n uint) *SessionBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(
		ffi_lk_session_builder_set_max_connections(b.ptr, uintptr(n)),
		"session builder set max connections",
	)
}

func (b *SessionBuilder) SetIdleTimeout(ms uint64) *SessionBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(ffi_lk_session_builder_set_idle_timeout(b.ptr, ms), "session builder set idle timeout")
}

func (b *SessionBuilder) SetRetryFixed(maxRetries uint32, intervalMs uint64) *SessionBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(
		ffi_lk_session_builder_set_retry_fixed(b.ptr, maxRetries, intervalMs),
		"session builder set retry fixed",
	)
}

func (b *SessionBuilder) SetRetryExponential(maxRetries uint32, baseMs, maxMs uint64, jitter bool) *SessionBuilder {
	if !b.check() {
		return b
	}
	return b.setStatus(
		ffi_lk_session_builder_set_retry_exponential(b.ptr, maxRetries, baseMs, maxMs, boolToUintptr(jitter)),
		"session builder set retry exponential",
	)
}

func (b *SessionBuilder) Build() (*Session, error) {
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
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_session_builder_build(
		b.ptr,
		uintptr(unsafe.Pointer(&cell[0])),
		uintptr(unsafe.Pointer(&cell[1])),
	)
	outSession, outErr := cell[0], cell[1]
	runtime.KeepAlive(b.client)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "session builder build"); err != nil {
		return nil, err
	}
	if outSession == 0 {
		return nil, nilHandleError("session builder build")
	}

	return newSessionFromPtr(outSession, b.client), nil
}
