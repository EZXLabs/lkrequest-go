package lkrequest

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"sync/atomic"
	"unsafe"
)

const (
	opWaitIntervalMs    uint64 = 50
	opCancelDrainWaitMs uint64 = 10
)

type Request struct {
	ptr          uintptr
	consumed     atomic.Bool
	firstErr     error
	hasMultipart bool
	session      *Session
	method       string
	url          string
	replay       []func(*Request) *Request
}

type requestStringView struct {
	ptr uintptr
	len uintptr
}

type requestFormPair struct {
	name  requestStringView
	value requestStringView
}

type formKV struct {
	name  string
	value string
}

func newRequestFromPtr(ptr uintptr, session *Session, method, url string) *Request {
	if ptr == 0 {
		return nil
	}

	req := &Request{
		ptr:     ptr,
		session: session,
		method:  method,
		url:     url,
	}
	runtime.SetFinalizer(req, (*Request).finalize)
	return req
}

func (r *Request) finalize() {
	r.release()
}

func (r *Request) release() {
	if r == nil {
		return
	}

	ptr := r.ptr
	r.ptr = 0
	runtime.SetFinalizer(r, nil)
	if ptr != 0 {
		ffi_lk_request_free(ptr)
	}
}

// detach takes the raw handle out of the wrapper and clears the finalizer, so
// the wrapper no longer owns it. The caller becomes responsible for releasing
// it with ffi_lk_request_free: the FFI send functions only borrow the handle
// (they move the request config out of it) and never take ownership, so a
// detached handle that is not freed leaks the RequestHandle together with the
// client and session Arc references it holds.
func (r *Request) detach() uintptr {
	if r == nil {
		return 0
	}

	ptr := r.ptr
	r.ptr = 0
	runtime.SetFinalizer(r, nil)
	return ptr
}

func (r *Request) checkMutable() bool {
	if r == nil {
		return false
	}
	if r.firstErr != nil {
		return false
	}
	if r.consumed.Load() {
		return false
	}
	if r.ptr == 0 {
		r.firstErr = ErrNilHandle
		return false
	}
	return true
}

func (r *Request) setStatus(status int32, action string) bool {
	if err := statusError(status, action); err != nil {
		if r.firstErr == nil {
			r.firstErr = err
		}
		return false
	}
	return true
}

func NewRequest(session *Session, method, url string) (*Request, error) {
	sp := session.loadPtr()
	if sp == 0 {
		return nil, ErrNilHandle
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	methodPtr, methodBuf := stringToCString(method)
	urlPtr, urlBuf := stringToCString(url)
	status := ffi_lk_request_new(
		sp,
		methodPtr,
		uintptr(len(method)),
		urlPtr,
		uintptr(len(url)),
		uintptr(unsafe.Pointer(&cell[0])),
		uintptr(unsafe.Pointer(&cell[1])),
	)
	outRequest, outErr := cell[0], cell[1]
	runtime.KeepAlive(methodBuf)
	runtime.KeepAlive(urlBuf)
	runtime.KeepAlive(session)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "request new"); err != nil {
		return nil, err
	}
	if outRequest == 0 {
		return nil, nilHandleError("request new")
	}

	return newRequestFromPtr(outRequest, session, method, url), nil
}

func (r *Request) AddHeader(name, value string) *Request {
	if !r.checkMutable() {
		return r
	}

	namePtr, nameBuf := stringToCString(name)
	valuePtr, valueBuf := stringToCString(value)
	status := ffi_lk_request_add_header(r.ptr, namePtr, uintptr(len(name)), valuePtr, uintptr(len(value)))
	runtime.KeepAlive(nameBuf)
	runtime.KeepAlive(valueBuf)
	if !r.setStatus(status, "request add header") {
		return r
	}

	r.replay = append(r.replay, func(clone *Request) *Request {
		return clone.AddHeader(name, value)
	})
	return r
}

func (r *Request) AddHeaderOrder(name string) *Request {
	if !r.checkMutable() {
		return r
	}

	namePtr, nameBuf := stringToCString(name)
	status := ffi_lk_request_add_header_order(r.ptr, namePtr, uintptr(len(name)))
	runtime.KeepAlive(nameBuf)
	if !r.setStatus(status, "request add header order") {
		return r
	}

	r.replay = append(r.replay, func(clone *Request) *Request {
		return clone.AddHeaderOrder(name)
	})
	return r
}

func (r *Request) AddH3HeaderOrder(name string) *Request {
	if !r.checkMutable() {
		return r
	}

	namePtr, nameBuf := stringToCString(name)
	status := ffi_lk_request_add_h3_header_order(r.ptr, namePtr, uintptr(len(name)))
	runtime.KeepAlive(nameBuf)
	if !r.setStatus(status, "request add h3 header order") {
		return r
	}

	r.replay = append(r.replay, func(clone *Request) *Request {
		return clone.AddH3HeaderOrder(name)
	})
	return r
}

func (r *Request) AddCookieOrder(name string) *Request {
	if !r.checkMutable() {
		return r
	}

	namePtr, nameBuf := stringToCString(name)
	status := ffi_lk_request_add_cookie_order(r.ptr, namePtr, uintptr(len(name)))
	runtime.KeepAlive(nameBuf)
	if !r.setStatus(status, "request add cookie order") {
		return r
	}

	r.replay = append(r.replay, func(clone *Request) *Request {
		return clone.AddCookieOrder(name)
	})
	return r
}

func (r *Request) AddQuery(key, value string) *Request {
	if !r.checkMutable() {
		return r
	}

	keyPtr, keyBuf := stringToCString(key)
	valuePtr, valueBuf := stringToCString(value)
	status := ffi_lk_request_add_query(r.ptr, keyPtr, uintptr(len(key)), valuePtr, uintptr(len(value)))
	runtime.KeepAlive(keyBuf)
	runtime.KeepAlive(valueBuf)
	if !r.setStatus(status, "request add query") {
		return r
	}

	r.replay = append(r.replay, func(clone *Request) *Request {
		return clone.AddQuery(key, value)
	})
	return r
}

func (r *Request) SetBodyBytes(data []byte) *Request {
	if !r.checkMutable() {
		return r
	}

	body := append([]byte(nil), data...)
	status := ffi_lk_request_set_body_bytes(r.ptr, bytesPtr(body), uintptr(len(body)))
	runtime.KeepAlive(body)
	if !r.setStatus(status, "request set body bytes") {
		return r
	}

	r.replay = append(r.replay, func(clone *Request) *Request {
		return clone.SetBodyBytes(body)
	})
	return r
}

func (r *Request) SetTextBody(text string) *Request {
	if !r.checkMutable() {
		return r
	}

	textPtr, textBuf := stringToCString(text)
	status := ffi_lk_request_set_text_body(r.ptr, textPtr, uintptr(len(text)))
	runtime.KeepAlive(textBuf)
	if !r.setStatus(status, "request set text body") {
		return r
	}

	r.replay = append(r.replay, func(clone *Request) *Request {
		return clone.SetTextBody(text)
	})
	return r
}

func (r *Request) SetJSONBody(jsonStr string) *Request {
	if !r.checkMutable() {
		return r
	}

	jsonPtr, jsonBuf := stringToCString(jsonStr)
	status := ffi_lk_request_set_json_text(r.ptr, jsonPtr, uintptr(len(jsonStr)))
	runtime.KeepAlive(jsonBuf)
	if !r.setStatus(status, "request set json body") {
		return r
	}

	r.replay = append(r.replay, func(clone *Request) *Request {
		return clone.SetJSONBody(jsonStr)
	})
	return r
}

func (r *Request) SetForm(pairs map[string]string) *Request {
	ordered := orderedFormPairs(pairs)
	return r.applyFormPairs(ordered)
}

func (r *Request) applyFormPairs(pairs []formKV) *Request {
	if !r.checkMutable() {
		return r
	}

	ffiPairs := make([]requestFormPair, len(pairs))
	buffers := make([][]byte, 0, len(pairs)*2)
	for i, pair := range pairs {
		namePtr, nameBuf := stringToCString(pair.name)
		valuePtr, valueBuf := stringToCString(pair.value)
		ffiPairs[i] = requestFormPair{
			name: requestStringView{
				ptr: namePtr,
				len: uintptr(len(pair.name)),
			},
			value: requestStringView{
				ptr: valuePtr,
				len: uintptr(len(pair.value)),
			},
		}
		buffers = append(buffers, nameBuf, valueBuf)
	}

	var pairsPtr uintptr
	if len(ffiPairs) > 0 {
		pairsPtr = uintptr(unsafe.Pointer(&ffiPairs[0]))
	}

	status := ffi_lk_request_set_form(r.ptr, pairsPtr, uintptr(len(ffiPairs)))
	runtime.KeepAlive(ffiPairs)
	runtime.KeepAlive(buffers)
	if !r.setStatus(status, "request set form") {
		return r
	}

	pairsCopy := append([]formKV(nil), pairs...)
	r.replay = append(r.replay, func(clone *Request) *Request {
		return clone.applyFormPairs(pairsCopy)
	})
	return r
}

func orderedFormPairs(pairs map[string]string) []formKV {
	if len(pairs) == 0 {
		return nil
	}

	keys := make([]string, 0, len(pairs))
	for key := range pairs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	ordered := make([]formKV, 0, len(keys))
	for _, key := range keys {
		ordered = append(ordered, formKV{name: key, value: pairs[key]})
	}

	return ordered
}

func (r *Request) SetTimeout(ms uint64) *Request {
	if !r.checkMutable() {
		return r
	}

	if !r.setStatus(ffi_lk_request_set_timeout(r.ptr, ms), "request set timeout") {
		return r
	}

	r.replay = append(r.replay, func(clone *Request) *Request {
		return clone.SetTimeout(ms)
	})
	return r
}

func (r *Request) SetAutoDecompress(enabled bool) *Request {
	if !r.checkMutable() {
		return r
	}

	if !r.setStatus(
		ffi_lk_request_set_auto_decompress(r.ptr, boolToUintptr(enabled)),
		"request set auto decompress",
	) {
		return r
	}

	r.replay = append(r.replay, func(clone *Request) *Request {
		return clone.SetAutoDecompress(enabled)
	})
	return r
}

func (r *Request) SetCookie(name, value string) *Request {
	if !r.checkMutable() {
		return r
	}

	namePtr, nameBuf := stringToCString(name)
	valuePtr, valueBuf := stringToCString(value)
	status := ffi_lk_request_set_cookie(r.ptr, namePtr, uintptr(len(name)), valuePtr, uintptr(len(value)))
	runtime.KeepAlive(nameBuf)
	runtime.KeepAlive(valueBuf)
	if !r.setStatus(status, "request set cookie") {
		return r
	}

	r.replay = append(r.replay, func(clone *Request) *Request {
		return clone.SetCookie(name, value)
	})
	return r
}

func (r *Request) SetBasicAuth(username, password string) *Request {
	if !r.checkMutable() {
		return r
	}

	usernamePtr, usernameBuf := stringToCString(username)
	passwordPtr, passwordBuf := stringToCString(password)
	status := ffi_lk_request_set_basic_auth(
		r.ptr,
		usernamePtr,
		uintptr(len(username)),
		passwordPtr,
		uintptr(len(password)),
	)
	runtime.KeepAlive(usernameBuf)
	runtime.KeepAlive(passwordBuf)
	if !r.setStatus(status, "request set basic auth") {
		return r
	}

	r.replay = append(r.replay, func(clone *Request) *Request {
		return clone.SetBasicAuth(username, password)
	})
	return r
}

func (r *Request) SetBearerAuth(token string) *Request {
	if !r.checkMutable() {
		return r
	}

	tokenPtr, tokenBuf := stringToCString(token)
	status := ffi_lk_request_set_bearer_auth(r.ptr, tokenPtr, uintptr(len(token)))
	runtime.KeepAlive(tokenBuf)
	if !r.setStatus(status, "request set bearer auth") {
		return r
	}

	r.replay = append(r.replay, func(clone *Request) *Request {
		return clone.SetBearerAuth(token)
	})
	return r
}

func (r *Request) SetProxy(proxy string) *Request {
	if !r.checkMutable() {
		return r
	}

	proxyPtr, proxyBuf := stringToCString(proxy)
	status := ffi_lk_request_set_proxy(r.ptr, proxyPtr, uintptr(len(proxy)))
	runtime.KeepAlive(proxyBuf)
	if !r.setStatus(status, "request set proxy") {
		return r
	}

	r.replay = append(r.replay, func(clone *Request) *Request {
		return clone.SetProxy(proxy)
	})
	return r
}

func (r *Request) SetVersion(v HttpVersion) *Request {
	if !r.checkMutable() {
		return r
	}

	if !r.setStatus(ffi_lk_request_set_version(r.ptr, int32(v)), "request set version") {
		return r
	}

	r.replay = append(r.replay, func(clone *Request) *Request {
		return clone.SetVersion(v)
	})
	return r
}

func (r *Request) SetPreferredHTTPVersion(v PreferredHTTPVersion) *Request {
	if !r.checkMutable() {
		return r
	}
	if ffi_lk_request_set_preferred_http_version == nil {
		if r.firstErr == nil {
			r.firstErr = unsupportedError("request set preferred http version")
		}
		return r
	}

	if !r.setStatus(
		ffi_lk_request_set_preferred_http_version(r.ptr, int32(v)),
		"request set preferred http version",
	) {
		return r
	}

	r.replay = append(r.replay, func(clone *Request) *Request {
		return clone.SetPreferredHTTPVersion(v)
	})
	return r
}

func (r *Request) SetAcceptEncoding(bits AcceptEncoding) *Request {
	if !r.checkMutable() {
		return r
	}

	if !r.setStatus(
		ffi_lk_request_set_accept_encoding(r.ptr, uint8(bits)),
		"request set accept encoding",
	) {
		return r
	}

	r.replay = append(r.replay, func(clone *Request) *Request {
		return clone.SetAcceptEncoding(bits)
	})
	return r
}

func (r *Request) SetCookieOverride(name, value string) *Request {
	if !r.checkMutable() {
		return r
	}

	namePtr, nameBuf := stringToCString(name)
	valuePtr, valueBuf := stringToCString(value)
	status := ffi_lk_request_set_cookie_override(r.ptr, namePtr, uintptr(len(name)), valuePtr, uintptr(len(value)))
	runtime.KeepAlive(nameBuf)
	runtime.KeepAlive(valueBuf)
	if !r.setStatus(status, "request set cookie override") {
		return r
	}

	r.replay = append(r.replay, func(clone *Request) *Request {
		return clone.SetCookieOverride(name, value)
	})
	return r
}

func (r *Request) SetMultipart(mp *Multipart) *Request {
	if !r.checkMutable() {
		return r
	}
	if mp == nil || mp.ptr == 0 {
		r.firstErr = ErrNilHandle
		return r
	}
	if mp.firstErr != nil {
		r.firstErr = mp.firstErr
		return r
	}

	status := ffi_lk_request_set_multipart(r.ptr, mp.ptr)
	if status == ffiStatusOK {
		mp.ptr = 0
		runtime.SetFinalizer(mp, nil)
		r.hasMultipart = true
	}
	if !r.setStatus(status, "request set multipart") {
		return r
	}
	return r
}

var errCloneMultipart = errors.New("lk: cannot clone request with multipart body")

func (r *Request) Clone() (*Request, error) {
	if r == nil {
		return nil, ErrNilHandle
	}
	if r.firstErr != nil {
		return nil, r.firstErr
	}
	if r.hasMultipart {
		return nil, errCloneMultipart
	}
	if r.session == nil {
		return nil, ErrNilHandle
	}

	clone, err := NewRequest(r.session, r.method, r.url)
	if err != nil {
		return nil, err
	}

	for _, op := range r.replay {
		op(clone)
	}

	if clone.firstErr != nil {
		err := clone.firstErr
		clone.release()
		return nil, err
	}

	return clone, nil
}

func (r *Request) Send() (*Response, error) {
	if r == nil {
		return nil, ErrNilHandle
	}
	if r.firstErr != nil {
		return nil, r.firstErr
	}
	if !r.consumed.CompareAndSwap(false, true) {
		return nil, ErrRequestConsumed
	}

	ptr := r.detach()
	if ptr == 0 {
		return nil, ErrNilHandle
	}
	defer ffi_lk_request_free(ptr)

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_request_send(
		ptr,
		uintptr(unsafe.Pointer(&cell[0])),
		uintptr(unsafe.Pointer(&cell[1])),
	)
	outResponse, outErr := cell[0], cell[1]
	runtime.KeepAlive(r.session)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "request send"); err != nil {
		return nil, err
	}
	if outResponse == 0 {
		return nil, nilHandleError("request send")
	}

	return newResponseFromPtr(outResponse), nil
}

func (r *Request) SendWithContext(ctx context.Context) (*Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if r == nil {
		return nil, ErrNilHandle
	}
	if r.firstErr != nil {
		return nil, r.firstErr
	}
	if !r.consumed.CompareAndSwap(false, true) {
		return nil, ErrRequestConsumed
	}

	ptr := r.detach()
	if ptr == 0 {
		return nil, ErrNilHandle
	}
	defer ffi_lk_request_free(ptr)

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_request_send_async(
		ptr,
		uintptr(unsafe.Pointer(&cell[0])),
		uintptr(unsafe.Pointer(&cell[1])),
	)
	op, outErr := cell[0], cell[1]
	runtime.KeepAlive(r.session)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "request send async"); err != nil {
		return nil, err
	}
	if op == 0 {
		return nil, nilHandleError("request send async")
	}
	defer ffi_lk_op_free(op)

	state, err := waitForOp(ctx, op)
	if err != nil {
		return nil, err
	}

	return takeResponseFromOp(op, state)
}

func (r *Request) SendStreaming() (*StreamingResponse, error) {
	if r == nil {
		return nil, ErrNilHandle
	}
	if r.firstErr != nil {
		return nil, r.firstErr
	}
	if !r.consumed.CompareAndSwap(false, true) {
		return nil, ErrRequestConsumed
	}

	ptr := r.detach()
	if ptr == 0 {
		return nil, ErrNilHandle
	}
	defer ffi_lk_request_free(ptr)

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_request_send_streaming(
		ptr,
		uintptr(unsafe.Pointer(&cell[0])),
		uintptr(unsafe.Pointer(&cell[1])),
	)
	outStream, outErr := cell[0], cell[1]
	runtime.KeepAlive(r.session)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "request send streaming"); err != nil {
		return nil, err
	}
	if outStream == 0 {
		return nil, nilHandleError("request send streaming")
	}

	return newStreamingResponseFromPtr(outStream), nil
}

func (r *Request) SendStreamingWithContext(ctx context.Context) (*StreamingResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if r == nil {
		return nil, ErrNilHandle
	}
	if r.firstErr != nil {
		return nil, r.firstErr
	}
	if !r.consumed.CompareAndSwap(false, true) {
		return nil, ErrRequestConsumed
	}

	ptr := r.detach()
	if ptr == 0 {
		return nil, ErrNilHandle
	}
	defer ffi_lk_request_free(ptr)

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_request_send_streaming_async(
		ptr,
		uintptr(unsafe.Pointer(&cell[0])),
		uintptr(unsafe.Pointer(&cell[1])),
	)
	op, outErr := cell[0], cell[1]
	runtime.KeepAlive(r.session)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "request send streaming async"); err != nil {
		return nil, err
	}
	if op == 0 {
		return nil, nilHandleError("request send streaming async")
	}
	defer ffi_lk_op_free(op)

	state, err := waitForOp(ctx, op)
	if err != nil {
		return nil, err
	}

	return takeStreamingResponseFromOp(op, state)
}

func (r *Request) SendAsync(ctx context.Context) (<-chan *Response, <-chan error) {
	respCh := make(chan *Response, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(respCh)
		defer close(errCh)

		resp, err := r.SendWithContext(ctx)
		if err != nil {
			errCh <- err
			return
		}

		respCh <- resp
	}()

	return respCh, errCh
}

func (r *Request) SendStreamingAsync(ctx context.Context) (<-chan *StreamingResponse, <-chan error) {
	respCh := make(chan *StreamingResponse, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(respCh)
		defer close(errCh)

		resp, err := r.SendStreamingWithContext(ctx)
		if err != nil {
			errCh <- err
			return
		}

		respCh <- resp
	}()

	return respCh, errCh
}

func waitForOp(ctx context.Context, op uintptr) (OpState, error) {
	for {
		if err := ctx.Err(); err != nil {
			_ = ffi_lk_op_cancel(op)
			drainOp(op)
			return 0, err
		}

		state := OpState(ffi_lk_op_wait(op, opWaitIntervalMs))
		if state == OpInProgress {
			continue
		}
		return state, nil
	}
}

func drainOp(op uintptr) {
	for {
		state := OpState(ffi_lk_op_wait(op, opCancelDrainWaitMs))
		if state != OpInProgress {
			return
		}
	}
}

func takeResponseFromOp(op uintptr, state OpState) (*Response, error) {
	switch state {
	case OpCompletedOK:
		cell := acquireFFIOutCell()
		defer ffiOutCellPool.Put(cell)
		status := ffi_lk_op_take_response(
			op,
			uintptr(unsafe.Pointer(&cell[0])),
			uintptr(unsafe.Pointer(&cell[1])),
		)
		outResponse, outErr := cell[0], cell[1]
		if err := extractError(outErr); err != nil {
			return nil, err
		}
		if err := statusError(status, "op take response"); err != nil {
			return nil, err
		}
		if outResponse == 0 {
			return nil, nilHandleError("op take response")
		}
		return newResponseFromPtr(outResponse), nil
	case OpCompletedErr:
		return nil, takeOperationError(op, "request send")
	case OpCancelled:
		return nil, context.Canceled
	case OpConsumed:
		return nil, fmt.Errorf("lk: op already consumed")
	default:
		return nil, fmt.Errorf("lk: unexpected op state %s", state)
	}
}

func takeStreamingResponseFromOp(op uintptr, state OpState) (*StreamingResponse, error) {
	switch state {
	case OpCompletedOK:
		cell := acquireFFIOutCell()
		defer ffiOutCellPool.Put(cell)
		status := ffi_lk_op_take_streaming_response(
			op,
			uintptr(unsafe.Pointer(&cell[0])),
			uintptr(unsafe.Pointer(&cell[1])),
		)
		outStream, outErr := cell[0], cell[1]
		if err := extractError(outErr); err != nil {
			return nil, err
		}
		if err := statusError(status, "op take streaming response"); err != nil {
			return nil, err
		}
		if outStream == 0 {
			return nil, nilHandleError("op take streaming response")
		}
		return newStreamingResponseFromPtr(outStream), nil
	case OpCompletedErr:
		return nil, takeOperationError(op, "request send streaming")
	case OpCancelled:
		return nil, context.Canceled
	case OpConsumed:
		return nil, fmt.Errorf("lk: op already consumed")
	default:
		return nil, fmt.Errorf("lk: unexpected op state %s", state)
	}
}

func takeOperationError(op uintptr, action string) error {
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_op_take_error(op, uintptr(unsafe.Pointer(&cell[0])))
	outErr := cell[0]
	if err := extractError(outErr); err != nil {
		return err
	}
	if err := statusError(status, action); err != nil {
		return err
	}

	return fmt.Errorf("lk: %s failed", action)
}
