package lkrequest

import (
	"runtime"
	"unsafe"
)

// AttemptLimitScope names the level an attempt limit in force comes from.
type AttemptLimitScope int32

const (
	// AttemptLimitUnset means that no level sets the limit, so none applies.
	AttemptLimitUnset AttemptLimitScope = 0
	// AttemptLimitClient means that the client's limit is in force.
	AttemptLimitClient AttemptLimitScope = 1
	// AttemptLimitSession means that the session's limit is in force.
	AttemptLimitSession AttemptLimitScope = 2
	// AttemptLimitRequest means that the request's limit is in force.
	AttemptLimitRequest AttemptLimitScope = 3
	// AttemptLimitUnknown means that the library reports a level this binding
	// does not know.
	AttemptLimitUnknown AttemptLimitScope = 4
)

func (s AttemptLimitScope) String() string {
	switch s {
	case AttemptLimitUnset:
		return "AttemptLimitUnset"
	case AttemptLimitClient:
		return "AttemptLimitClient"
	case AttemptLimitSession:
		return "AttemptLimitSession"
	case AttemptLimitRequest:
		return "AttemptLimitRequest"
	default:
		return "AttemptLimitUnknown"
	}
}

// AttemptLimit is an attempt limit as one level reports it: a client, a
// session or a request, or the builder of one.
type AttemptLimit struct {
	// Configured is the limit set at this level, or nil when the level sets
	// none and inherits its parent's.
	Configured *uint64
	// Effective is the limit in force at this level, the tightest of its own
	// and its parents', or nil when no level sets one.
	Effective *uint64
	// Scope is the level Effective comes from.
	Scope AttemptLimitScope
}

// ffiAttemptLimitView mirrors lk_attempt_limit_view_t. Its 40 bytes fit the
// first five slots of an FFI out cell.
type ffiAttemptLimitView struct {
	configuredPresent bool
	configured        uint64
	effectivePresent  bool
	effective         uint64
	scope             int32
}

// limit converts the view into an AttemptLimit.
func (v *ffiAttemptLimitView) limit() AttemptLimit {
	limit := AttemptLimit{Scope: AttemptLimitScope(v.scope)}
	if v.configuredPresent {
		configured := v.configured
		limit.Configured = &configured
	}
	if v.effectivePresent {
		effective := v.effective
		limit.Effective = &effective
	}
	return limit
}

// SetConnectAttemptLimit caps the TCP and QUIC connections each request of
// the client may start, failed starts and the losers of a connection race
// included; reusing a pooled connection, or one another request is opening,
// starts none. With 0 a request starts no connection at all. It is unset by
// default and sets no deadline; sessions and requests inherit it and may only
// tighten it. Requires a library exporting
// lk_client_builder_set_connect_attempt_limit.
func (b *ClientBuilder) SetConnectAttemptLimit(limit uint64) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.keepErr(setAttemptLimit(ffi_lk_client_builder_set_connect_attempt_limit, b.ptr, limit, "client builder set connect attempt limit"))
}

// SetExtraHTTPAttemptLimit caps the HTTP submissions each request of the
// client may make beyond its first while recovering from a failure, through
// retries, replays and protocol fallbacks. The first submission and the
// redirects a request follows do not count, and the limit enables no recovery
// the client's settings leave off. With 0 a request is never sent again. It is
// unset by default; sessions and requests inherit it and may only tighten it.
// Requires a library exporting lk_client_builder_set_extra_http_attempt_limit.
func (b *ClientBuilder) SetExtraHTTPAttemptLimit(limit uint64) *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.keepErr(setAttemptLimit(ffi_lk_client_builder_set_extra_http_attempt_limit, b.ptr, limit, "client builder set extra http attempt limit"))
}

// ClearConnectAttemptLimit removes the client's connect attempt limit.
// Requires a library exporting lk_client_builder_clear_connect_attempt_limit.
func (b *ClientBuilder) ClearConnectAttemptLimit() *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.keepErr(clearAttemptLimit(ffi_lk_client_builder_clear_connect_attempt_limit, b.ptr, "client builder clear connect attempt limit"))
}

// ClearExtraHTTPAttemptLimit removes the client's extra HTTP attempt limit.
// Requires a library exporting
// lk_client_builder_clear_extra_http_attempt_limit.
func (b *ClientBuilder) ClearExtraHTTPAttemptLimit() *ClientBuilder {
	if !b.check() {
		return b
	}
	return b.keepErr(clearAttemptLimit(ffi_lk_client_builder_clear_extra_http_attempt_limit, b.ptr, "client builder clear extra http attempt limit"))
}

// ConnectAttemptLimit reports the connect attempt limit the builder sets.
func (b *ClientBuilder) ConnectAttemptLimit() (AttemptLimit, error) {
	if b == nil || b.ptr == 0 {
		return AttemptLimit{}, ErrNilHandle
	}
	return getAttemptLimit(ffi_lk_client_builder_get_connect_attempt_limit, b.ptr, "client builder get connect attempt limit")
}

// ExtraHTTPAttemptLimit reports the extra HTTP attempt limit the builder
// sets.
func (b *ClientBuilder) ExtraHTTPAttemptLimit() (AttemptLimit, error) {
	if b == nil || b.ptr == 0 {
		return AttemptLimit{}, ErrNilHandle
	}
	return getAttemptLimit(ffi_lk_client_builder_get_extra_http_attempt_limit, b.ptr, "client builder get extra http attempt limit")
}

// keepErr records err, if it is the builder's first error, for Build to
// report.
func (b *ClientBuilder) keepErr(err error) *ClientBuilder {
	if err != nil && b.firstErr == nil {
		b.firstErr = err
	}
	return b
}

// ConnectAttemptLimit reports the client's connect attempt limit.
func (c *Client) ConnectAttemptLimit() (AttemptLimit, error) {
	defer runtime.KeepAlive(c)
	p := c.loadPtr()
	if p == 0 {
		return AttemptLimit{}, ErrNilHandle
	}
	return getAttemptLimit(ffi_lk_client_get_connect_attempt_limit, p, "client get connect attempt limit")
}

// ExtraHTTPAttemptLimit reports the client's extra HTTP attempt limit.
func (c *Client) ExtraHTTPAttemptLimit() (AttemptLimit, error) {
	defer runtime.KeepAlive(c)
	p := c.loadPtr()
	if p == 0 {
		return AttemptLimit{}, ErrNilHandle
	}
	return getAttemptLimit(ffi_lk_client_get_extra_http_attempt_limit, p, "client get extra http attempt limit")
}

// SetConnectAttemptLimit caps the connections each request of the session may
// start, as ClientBuilder.SetConnectAttemptLimit does for a client. A limit
// looser than the client's is an error, reported by Build. Requires a library
// exporting lk_session_builder_set_connect_attempt_limit.
func (b *SessionBuilder) SetConnectAttemptLimit(limit uint64) *SessionBuilder {
	if !b.check() {
		return b
	}
	return b.keepErr(setAttemptLimit(ffi_lk_session_builder_set_connect_attempt_limit, b.ptr, limit, "session builder set connect attempt limit"))
}

// SetExtraHTTPAttemptLimit caps the extra HTTP submissions of each request of
// the session, as ClientBuilder.SetExtraHTTPAttemptLimit does for a client. A
// limit looser than the client's is an error, reported by Build. Requires a
// library exporting lk_session_builder_set_extra_http_attempt_limit.
func (b *SessionBuilder) SetExtraHTTPAttemptLimit(limit uint64) *SessionBuilder {
	if !b.check() {
		return b
	}
	return b.keepErr(setAttemptLimit(ffi_lk_session_builder_set_extra_http_attempt_limit, b.ptr, limit, "session builder set extra http attempt limit"))
}

// ClearConnectAttemptLimit removes the session's connect attempt limit, so
// that the session inherits the client's. Requires a library exporting
// lk_session_builder_clear_connect_attempt_limit.
func (b *SessionBuilder) ClearConnectAttemptLimit() *SessionBuilder {
	if !b.check() {
		return b
	}
	return b.keepErr(clearAttemptLimit(ffi_lk_session_builder_clear_connect_attempt_limit, b.ptr, "session builder clear connect attempt limit"))
}

// ClearExtraHTTPAttemptLimit removes the session's extra HTTP attempt limit,
// so that the session inherits the client's. Requires a library exporting
// lk_session_builder_clear_extra_http_attempt_limit.
func (b *SessionBuilder) ClearExtraHTTPAttemptLimit() *SessionBuilder {
	if !b.check() {
		return b
	}
	return b.keepErr(clearAttemptLimit(ffi_lk_session_builder_clear_extra_http_attempt_limit, b.ptr, "session builder clear extra http attempt limit"))
}

// ConnectAttemptLimit reports the connect attempt limit the builder sets and
// the one in force for the session it builds.
func (b *SessionBuilder) ConnectAttemptLimit() (AttemptLimit, error) {
	if b == nil || b.ptr == 0 {
		return AttemptLimit{}, ErrNilHandle
	}
	return getAttemptLimit(ffi_lk_session_builder_get_connect_attempt_limit, b.ptr, "session builder get connect attempt limit")
}

// ExtraHTTPAttemptLimit reports the extra HTTP attempt limit the builder sets
// and the one in force for the session it builds.
func (b *SessionBuilder) ExtraHTTPAttemptLimit() (AttemptLimit, error) {
	if b == nil || b.ptr == 0 {
		return AttemptLimit{}, ErrNilHandle
	}
	return getAttemptLimit(ffi_lk_session_builder_get_extra_http_attempt_limit, b.ptr, "session builder get extra http attempt limit")
}

// keepErr records err, if it is the builder's first error, for Build to
// report.
func (b *SessionBuilder) keepErr(err error) *SessionBuilder {
	if err != nil && b.firstErr == nil {
		b.firstErr = err
	}
	return b
}

// ConnectAttemptLimit reports the session's connect attempt limit.
func (s *Session) ConnectAttemptLimit() (AttemptLimit, error) {
	defer runtime.KeepAlive(s)
	p := s.loadPtr()
	if p == 0 {
		return AttemptLimit{}, ErrNilHandle
	}
	return getAttemptLimit(ffi_lk_session_get_connect_attempt_limit, p, "session get connect attempt limit")
}

// ExtraHTTPAttemptLimit reports the session's extra HTTP attempt limit.
func (s *Session) ExtraHTTPAttemptLimit() (AttemptLimit, error) {
	defer runtime.KeepAlive(s)
	p := s.loadPtr()
	if p == 0 {
		return AttemptLimit{}, ErrNilHandle
	}
	return getAttemptLimit(ffi_lk_session_get_extra_http_attempt_limit, p, "session get extra http attempt limit")
}

// SetConnectAttemptLimit caps the connections the request may start, as
// ClientBuilder.SetConnectAttemptLimit does for each request of a client. A
// limit looser than its session's is an error, reported by Send. Requires a
// library exporting lk_request_set_connect_attempt_limit.
func (r *Request) SetConnectAttemptLimit(limit uint64) *Request {
	if !r.checkMutable() {
		return r
	}
	if r.keepErr(setAttemptLimit(ffi_lk_request_set_connect_attempt_limit, r.ptr, limit, "request set connect attempt limit")) {
		r.replay = append(r.replay, func(clone *Request) *Request {
			return clone.SetConnectAttemptLimit(limit)
		})
	}
	return r
}

// SetExtraHTTPAttemptLimit caps the extra HTTP submissions of the request, as
// ClientBuilder.SetExtraHTTPAttemptLimit does for each request of a client. A
// limit looser than its session's is an error, reported by Send. Requires a
// library exporting lk_request_set_extra_http_attempt_limit.
func (r *Request) SetExtraHTTPAttemptLimit(limit uint64) *Request {
	if !r.checkMutable() {
		return r
	}
	if r.keepErr(setAttemptLimit(ffi_lk_request_set_extra_http_attempt_limit, r.ptr, limit, "request set extra http attempt limit")) {
		r.replay = append(r.replay, func(clone *Request) *Request {
			return clone.SetExtraHTTPAttemptLimit(limit)
		})
	}
	return r
}

// ClearConnectAttemptLimit removes the request's connect attempt limit, so
// that the request inherits its session's. Requires a library exporting
// lk_request_clear_connect_attempt_limit.
func (r *Request) ClearConnectAttemptLimit() *Request {
	if !r.checkMutable() {
		return r
	}
	if r.keepErr(clearAttemptLimit(ffi_lk_request_clear_connect_attempt_limit, r.ptr, "request clear connect attempt limit")) {
		r.replay = append(r.replay, (*Request).ClearConnectAttemptLimit)
	}
	return r
}

// ClearExtraHTTPAttemptLimit removes the request's extra HTTP attempt limit,
// so that the request inherits its session's. Requires a library exporting
// lk_request_clear_extra_http_attempt_limit.
func (r *Request) ClearExtraHTTPAttemptLimit() *Request {
	if !r.checkMutable() {
		return r
	}
	if r.keepErr(clearAttemptLimit(ffi_lk_request_clear_extra_http_attempt_limit, r.ptr, "request clear extra http attempt limit")) {
		r.replay = append(r.replay, (*Request).ClearExtraHTTPAttemptLimit)
	}
	return r
}

// ConnectAttemptLimit reports the request's connect attempt limit. It fails
// once the request has been sent.
func (r *Request) ConnectAttemptLimit() (AttemptLimit, error) {
	if err := r.checkReadable(); err != nil {
		return AttemptLimit{}, err
	}
	return getAttemptLimit(ffi_lk_request_get_connect_attempt_limit, r.ptr, "request get connect attempt limit")
}

// ExtraHTTPAttemptLimit reports the request's extra HTTP attempt limit. It
// fails once the request has been sent.
func (r *Request) ExtraHTTPAttemptLimit() (AttemptLimit, error) {
	if err := r.checkReadable(); err != nil {
		return AttemptLimit{}, err
	}
	return getAttemptLimit(ffi_lk_request_get_extra_http_attempt_limit, r.ptr, "request get extra http attempt limit")
}

// EnableAttemptSummary has the library record what the request's attempts
// were — its HTTP submissions and connection starts and how each ended — for
// AttemptSummaryJSON of its Response, StreamingResponse or LkError. It is off
// by default, and changes neither the request's diagnostics nor its retries.
// Requires a library exporting lk_request_enable_attempt_summary.
func (r *Request) EnableAttemptSummary(enabled bool) *Request {
	if !r.checkMutable() {
		return r
	}
	if ffi_lk_request_enable_attempt_summary == nil {
		r.keepErr(unsupportedError("request enable attempt summary"))
		return r
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_request_enable_attempt_summary(r.ptr, boolToUintptr(enabled), uintptr(unsafe.Pointer(&cell[0])))
	err := extractError(cell[0])
	if err == nil {
		err = statusError(status, "request enable attempt summary")
	}
	if r.keepErr(err) {
		r.replay = append(r.replay, func(clone *Request) *Request {
			return clone.EnableAttemptSummary(enabled)
		})
	}
	return r
}

// keepErr records err, if it is the request's first error, for Send to
// report, and reports whether err is nil.
func (r *Request) keepErr(err error) bool {
	if err == nil {
		return true
	}
	if r.firstErr == nil {
		r.firstErr = err
	}
	return false
}

// checkReadable returns the error of reading from a request that has been
// sent or has no handle.
func (r *Request) checkReadable() error {
	if r == nil || r.ptr == 0 {
		return ErrNilHandle
	}
	if r.consumed.Load() {
		return ErrRequestConsumed
	}
	return nil
}

// AttemptSummaryJSON returns, as JSON, the summary of the attempts made by
// the request this response answers, or "" when the request did not enable
// one with Request.EnableAttemptSummary.
func (r *Response) AttemptSummaryJSON() string {
	defer runtime.KeepAlive(r)
	if r == nil || r.ptr == 0 {
		return ""
	}
	return copyAttemptSummary(ffi_lk_response_copy_attempt_summary_json, r.ptr)
}

// AttemptSummaryJSON returns, as JSON, the summary of the attempts made by
// the request this response answers, or "" when the request did not enable
// one with Request.EnableAttemptSummary. Its counts are final once the
// headers have arrived.
func (s *StreamingResponse) AttemptSummaryJSON() string {
	defer runtime.KeepAlive(s)
	if s == nil || s.ptr == 0 {
		return ""
	}
	return copyAttemptSummary(ffi_lk_streaming_response_copy_attempt_summary_json, s.ptr)
}

// AttemptSummaryJSON returns, as JSON, the summary of the attempts made by
// the request that failed with this error, or "" when the request did not
// enable one with Request.EnableAttemptSummary.
func (e *LkError) AttemptSummaryJSON() string {
	if e == nil {
		return ""
	}
	return e.attemptSummary
}

// setAttemptLimit calls set, the attempt-limit setter of handle's level, and
// returns the error the library reports.
func setAttemptLimit(set func(handle uintptr, limit uint64, outErr uintptr) int32, handle uintptr, limit uint64, action string) error {
	if set == nil {
		return unsupportedError(action)
	}
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := set(handle, limit, uintptr(unsafe.Pointer(&cell[0])))
	if err := extractError(cell[0]); err != nil {
		return err
	}
	return statusError(status, action)
}

// clearAttemptLimit calls clear, the attempt-limit clearer of handle's level,
// and returns the error the library reports.
func clearAttemptLimit(clear func(handle uintptr, outErr uintptr) int32, handle uintptr, action string) error {
	if clear == nil {
		return unsupportedError(action)
	}
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := clear(handle, uintptr(unsafe.Pointer(&cell[0])))
	if err := extractError(cell[0]); err != nil {
		return err
	}
	return statusError(status, action)
}

// getAttemptLimit reads an attempt limit through get, the attempt-limit
// getter of handle's level. The view occupies the first five slots of the out
// cell and the error the sixth.
func getAttemptLimit(get func(handle uintptr, outLimit uintptr, outErr uintptr) int32, handle uintptr, action string) (AttemptLimit, error) {
	if get == nil {
		return AttemptLimit{}, unsupportedError(action)
	}
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	view := (*ffiAttemptLimitView)(unsafe.Pointer(&cell[0]))
	status := get(handle, uintptr(unsafe.Pointer(view)), uintptr(unsafe.Pointer(&cell[5])))
	limit := view.limit()
	if err := extractError(cell[5]); err != nil {
		return AttemptLimit{}, err
	}
	if err := statusError(status, action); err != nil {
		return AttemptLimit{}, err
	}
	return limit, nil
}

// copyAttemptSummary copies the attempt summary of handle through copyJSON,
// the summary copier of its type, growing the buffer while the summary of a
// request still in flight grows. It returns "" when there is none.
func copyAttemptSummary(copyJSON func(handle uintptr, buffer uintptr, capacity uintptr, outRequired uintptr, outErr uintptr) int32, handle uintptr) string {
	if copyJSON == nil {
		return ""
	}
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)

	var buf []byte
	for i := 0; i < 4; i++ {
		cell[0], cell[1] = 0, 0
		status := copyJSON(handle, bytesPtr(buf), uintptr(len(buf)), uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])))
		required := int(cell[0])
		if cell[1] != 0 {
			ffi_lk_error_free(cell[1])
		}
		if status == ffiStatusOK && required <= len(buf) {
			return string(buf[:required])
		}
		if required <= len(buf) {
			return ""
		}
		buf = make([]byte, required)
	}
	return ""
}
