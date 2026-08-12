package lkrequest

import (
	"errors"
	"unsafe"
)

var (
	ErrRequestConsumed = errors.New("lk: request already consumed")
	ErrNilHandle       = errors.New("lk: nil handle")
)

type LkError struct {
	msg         string
	code        ErrorCode
	phase       Phase
	retryable   bool
	httpStatus  int32
	diagnostics string
}

func extractError(errPtr uintptr) error {
	if errPtr == 0 {
		return nil
	}

	e := &LkError{}
	e.code = ErrorCode(ffi_lk_error_code(errPtr))
	e.phase = Phase(ffi_lk_error_phase(errPtr))
	e.retryable = ffiBool(ffi_lk_error_is_retryable(errPtr))
	e.httpStatus = ffi_lk_error_http_status(errPtr)

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	if ffi_lk_error_message(
		errPtr,
		uintptr(unsafe.Pointer(&cell[0])),
		uintptr(unsafe.Pointer(&cell[1])),
	) == ffiStatusOK {
		e.msg = goStringN(unsafe.Pointer(cell[0]), cell[1])
	}
	if e.msg == "" {
		if e.code != 0 {
			e.msg = e.code.String()
		} else {
			e.msg = "lk: ffi error"
		}
	}

	if ffi_lk_error_get_diagnostics_json(errPtr, uintptr(unsafe.Pointer(&cell[2]))) == ffiStatusOK && cell[2] != 0 {
		e.diagnostics = goCString(unsafe.Pointer(cell[2]))
	}

	ffi_lk_error_free(errPtr)
	return e
}

func (e *LkError) Error() string {
	if e == nil {
		return ErrNilHandle.Error()
	}
	return e.msg
}

func (e *LkError) Code() ErrorCode {
	if e == nil {
		return 0
	}
	return e.code
}

func (e *LkError) Phase() Phase {
	if e == nil {
		return 0
	}
	return e.phase
}

func (e *LkError) IsRetryable() bool {
	if e == nil {
		return false
	}
	return e.retryable
}

func (e *LkError) HttpStatus() int32 {
	if e == nil {
		return 0
	}
	return e.httpStatus
}

func (e *LkError) DiagnosticsJSON() string {
	if e == nil {
		return ""
	}
	return e.diagnostics
}

// Close is a no-op retained for backward compatibility.
// FFI resources are now freed eagerly when the error is created.
func (e *LkError) Close() {}
