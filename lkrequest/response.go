package lkrequest

import (
	"encoding/json"
	"net/http"
	"runtime"
	"sync"
	"unsafe"
)

// responseBody buffers the response body. It is kept separate from Response so
// that the UnmarshalJSON field can close over it without capturing the
// Response itself: a bound method value (or any closure referencing the
// Response) would make the object reference itself, and the runtime never runs
// finalizers for objects that are part of a reference cycle — which would
// silently disable the SetFinalizer safety net in newResponseFromPtr.
type responseBody struct {
	ptr    uintptr
	once   sync.Once
	cached []byte
}

func (b *responseBody) get() []byte {
	if b == nil || b.ptr == 0 {
		return nil
	}
	b.once.Do(func() {
		cell := acquireFFIOutCell()
		defer ffiOutCellPool.Put(cell)
		if ffi_lk_response_body(
			b.ptr,
			uintptr(unsafe.Pointer(&cell[0])),
			uintptr(unsafe.Pointer(&cell[1])),
		) != ffiStatusOK {
			return
		}

		bodyLen := cell[1]
		if bodyLen == 0 {
			b.cached = []byte{}
			return
		}

		out := make([]byte, int(bodyLen))
		if ffi_lk_response_copy_body(
			b.ptr,
			bytesPtr(out),
			uintptr(len(out)),
			uintptr(unsafe.Pointer(&cell[2])),
		) == ffiStatusOK {
			b.cached = out[:cell[2]]
		}
	})
	return b.cached
}

// release drops the buffered body and stops further reads; called from
// Response.Close once the underlying handle is about to be freed.
func (b *responseBody) release() {
	if b == nil {
		return
	}
	b.ptr = 0
	b.cached = nil
}

type Response struct {
	ptr           uintptr
	once          sync.Once
	body          *responseBody
	UnmarshalJSON func(v any) error
}

func newResponseFromPtr(ptr uintptr) *Response {
	if ptr == 0 {
		return nil
	}

	body := &responseBody{ptr: ptr}
	resp := &Response{ptr: ptr, body: body}
	// Capture body, never resp — see the responseBody doc comment.
	resp.UnmarshalJSON = func(v any) error {
		return json.Unmarshal(body.get(), v)
	}
	runtime.SetFinalizer(resp, (*Response).Close)
	return resp
}

func (r *Response) StatusCode() int {
	if r == nil || r.ptr == 0 {
		return 0
	}
	return int(ffi_lk_response_status(r.ptr))
}

func (r *Response) Version() HttpVersion {
	if r == nil || r.ptr == 0 {
		return HttpVersionUnknown
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	if ffi_lk_response_version(r.ptr, uintptr(unsafe.Pointer(&cell[0]))) != ffiStatusOK {
		return HttpVersionUnknown
	}

	return HttpVersion(int32(uint32(cell[0])))
}

func (r *Response) URL() string {
	defer runtime.KeepAlive(r)
	if r == nil || r.ptr == 0 {
		return ""
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	if ffi_lk_response_url(r.ptr, uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1]))) != ffiStatusOK {
		return ""
	}

	return goStringN(unsafe.Pointer(cell[0]), cell[1])
}

func (r *Response) HeaderCount() int {
	if r == nil || r.ptr == 0 {
		return 0
	}
	return int(ffi_lk_response_header_count(r.ptr))
}

func (r *Response) HeaderAt(index int) (name, value string) {
	defer runtime.KeepAlive(r)
	if r == nil || r.ptr == 0 || index < 0 {
		return "", ""
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	if ffi_lk_response_header_name_at(
		r.ptr,
		uintptr(index),
		uintptr(unsafe.Pointer(&cell[0])),
		uintptr(unsafe.Pointer(&cell[1])),
	) != ffiStatusOK {
		return "", ""
	}

	if ffi_lk_response_header_value_at(
		r.ptr,
		uintptr(index),
		uintptr(unsafe.Pointer(&cell[2])),
		uintptr(unsafe.Pointer(&cell[3])),
	) != ffiStatusOK {
		return "", ""
	}

	return goStringN(unsafe.Pointer(cell[0]), cell[1]), goStringN(unsafe.Pointer(cell[2]), cell[3])
}

func (r *Response) Header(name string) string {
	defer runtime.KeepAlive(r)
	if r == nil || r.ptr == 0 {
		return ""
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	namePtr, nameBuf := stringToCString(name)
	status := ffi_lk_response_get_header_by_name(
		r.ptr,
		namePtr,
		uintptr(len(name)),
		uintptr(unsafe.Pointer(&cell[0])),
		uintptr(unsafe.Pointer(&cell[1])),
		uintptr(unsafe.Pointer(&cell[2])),
	)
	valuePtr, valueLen, outErr := unsafe.Pointer(cell[0]), cell[1], cell[2]
	runtime.KeepAlive(nameBuf)

	if outErr != 0 {
		ffi_lk_error_free(outErr)
	}
	if status != ffiStatusOK {
		return ""
	}

	return goStringN(valuePtr, valueLen)
}

func (r *Response) Headers() http.Header {
	headers := make(http.Header, r.HeaderCount())
	for i := 0; i < r.HeaderCount(); i++ {
		name, value := r.HeaderAt(i)
		if name == "" {
			continue
		}
		headers.Add(name, value)
	}
	return headers
}

func (r *Response) Body() []byte {
	if r == nil {
		return nil
	}
	return r.body.get()
}

func (r *Response) Bytes() []byte {
	return r.Body()
}

func (r *Response) String() string {
	return string(r.Body())
}

func (r *Response) ContentLength() int64 {
	if r == nil || r.ptr == 0 {
		return 0
	}
	return ffi_lk_response_content_length(r.ptr)
}

func (r *Response) DiagnosticsJSON() string {
	defer runtime.KeepAlive(r)
	if r == nil || r.ptr == 0 {
		return ""
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	if ffi_lk_response_get_diagnostics_json(r.ptr, uintptr(unsafe.Pointer(&cell[0]))) != ffiStatusOK {
		return ""
	}

	return goCString(unsafe.Pointer(cell[0]))
}

func (r *Response) Text() (string, error) {
	defer runtime.KeepAlive(r)
	if r == nil || r.ptr == 0 {
		return "", ErrNilHandle
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_response_text(r.ptr, uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])), uintptr(unsafe.Pointer(&cell[2])))
	if err := extractError(cell[2]); err != nil {
		return "", err
	}
	if err := statusError(status, "response text"); err != nil {
		return "", err
	}
	return goStringN(unsafe.Pointer(cell[0]), cell[1]), nil
}

func (r *Response) ErrorForStatus() error {
	if r == nil || r.ptr == 0 {
		return ErrNilHandle
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	ffi_lk_response_error_for_status(r.ptr, uintptr(unsafe.Pointer(&cell[0])))
	return extractError(cell[0])
}

func (r *Response) CookieCount() int {
	if r == nil || r.ptr == 0 {
		return 0
	}
	return int(ffi_lk_response_cookie_count(r.ptr))
}

func (r *Response) CookieAt(index int) (name, value string) {
	defer runtime.KeepAlive(r)
	if r == nil || r.ptr == 0 || index < 0 {
		return "", ""
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	if ffi_lk_response_cookie_at(r.ptr, uintptr(index), uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])), uintptr(unsafe.Pointer(&cell[2])), uintptr(unsafe.Pointer(&cell[3]))) != ffiStatusOK {
		return "", ""
	}
	return goStringN(unsafe.Pointer(cell[0]), cell[1]), goStringN(unsafe.Pointer(cell[2]), cell[3])
}

// NegotiatedVersion reports the HTTP version actually negotiated for the
// connection (via ALPN / Alt-Svc), which may differ from the version carried
// by the final response. Returns HttpVersionUnknown when the loaded library
// predates this accessor or the version is unavailable.
func (r *Response) NegotiatedVersion() HttpVersion {
	if r == nil || r.ptr == 0 || ffi_lk_response_negotiated_version == nil {
		return HttpVersionUnknown
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	if ffi_lk_response_negotiated_version(r.ptr, uintptr(unsafe.Pointer(&cell[0]))) != ffiStatusOK {
		return HttpVersionUnknown
	}
	return HttpVersion(int32(uint32(cell[0])))
}

// CookieNameAt returns the name of the response Set-Cookie at index, or "".
// It is the single-field counterpart to CookieAt.
func (r *Response) CookieNameAt(index int) string {
	defer runtime.KeepAlive(r)
	if r == nil || r.ptr == 0 || index < 0 || ffi_lk_response_cookie_name_at == nil {
		return ""
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	if ffi_lk_response_cookie_name_at(r.ptr, uintptr(index), uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1]))) != ffiStatusOK {
		return ""
	}
	return goStringN(unsafe.Pointer(cell[0]), cell[1])
}

// CookieValueAt returns the value of the response Set-Cookie at index, or "".
// It is the single-field counterpart to CookieAt.
func (r *Response) CookieValueAt(index int) string {
	defer runtime.KeepAlive(r)
	if r == nil || r.ptr == 0 || index < 0 || ffi_lk_response_cookie_value_at == nil {
		return ""
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	if ffi_lk_response_cookie_value_at(r.ptr, uintptr(index), uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1]))) != ffiStatusOK {
		return ""
	}
	return goStringN(unsafe.Pointer(cell[0]), cell[1])
}

func (r *Response) WasRedirected() bool {
	if r == nil || r.ptr == 0 {
		return false
	}
	return ffiBool(ffi_lk_response_was_redirected(r.ptr))
}

func (r *Response) RedirectCount() int {
	if r == nil || r.ptr == 0 {
		return 0
	}
	return int(ffi_lk_response_redirect_count(r.ptr))
}

func (r *Response) RedirectAt(index int) (url string, status int) {
	defer runtime.KeepAlive(r)
	if r == nil || r.ptr == 0 || index < 0 {
		return "", 0
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	if ffi_lk_response_redirect_at(r.ptr, uintptr(index), uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])), uintptr(unsafe.Pointer(&cell[2]))) != ffiStatusOK {
		return "", 0
	}
	return goStringN(unsafe.Pointer(cell[0]), cell[1]), int(uint16(cell[2]))
}

// RedirectURLAt returns the URL of the redirect hop at index, or "".
// It is the single-field counterpart to RedirectAt.
func (r *Response) RedirectURLAt(index int) string {
	defer runtime.KeepAlive(r)
	if r == nil || r.ptr == 0 || index < 0 || ffi_lk_response_redirect_url_at == nil {
		return ""
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	if ffi_lk_response_redirect_url_at(r.ptr, uintptr(index), uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1]))) != ffiStatusOK {
		return ""
	}
	return goStringN(unsafe.Pointer(cell[0]), cell[1])
}

// RedirectStatusAt returns the HTTP status code of the redirect hop at index,
// or 0. It is the single-field counterpart to RedirectAt.
func (r *Response) RedirectStatusAt(index int) int {
	if r == nil || r.ptr == 0 || index < 0 || ffi_lk_response_redirect_status_at == nil {
		return 0
	}
	return int(ffi_lk_response_redirect_status_at(r.ptr, uintptr(index)))
}

func (r *Response) Close() {
	if r == nil {
		return
	}

	r.once.Do(func() {
		ptr := r.ptr
		r.ptr = 0
		r.body.release()
		runtime.SetFinalizer(r, nil)
		if ptr != 0 {
			ffi_lk_response_free(ptr)
		}
	})
}
