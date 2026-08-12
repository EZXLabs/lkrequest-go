package lkrequest

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

const ffiStatusOK int32 = 0

func boolToUintptr(v bool) uintptr {
	if v {
		return 1
	}
	return 0
}

// ffiBool interprets the return value of a C function that returns `bool`.
//
// On x86-64 a bool-returning function only sets the low byte (AL) of the
// return register; the upper bits are undefined and an optimized library
// build may leave stale garbage there. The purego backend reads the whole
// register as a uintptr, so a plain `!= 0` check would misread a `false`
// (low byte 0) as true whenever the high bits are dirty. Masking to the low
// byte yields the real bool. The cgo backend already returns a clean 0/1, so
// the mask is a no-op there.
func ffiBool(v uintptr) bool {
	return v&0xFF != 0
}

func stringToCString(s string) (uintptr, []byte) {
	buf := append([]byte(s), 0)
	return uintptr(unsafe.Pointer(&buf[0])), buf
}

func optionalStringToCString(s string) (uintptr, []byte) {
	if s == "" {
		return 0, nil
	}
	return stringToCString(s)
}

func bytesPtr(b []byte) uintptr {
	if len(b) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(unsafe.SliceData(b)))
}

func goStringN(ptr unsafe.Pointer, n uintptr) string {
	if ptr == nil || n == 0 {
		return ""
	}
	buf := make([]byte, int(n))
	copy(buf, unsafe.Slice((*byte)(ptr), int(n)))
	return string(buf)
}

func goCString(ptr unsafe.Pointer) string {
	if ptr == nil {
		return ""
	}

	const maxLen = 1 << 24 // 16 MiB safety limit
	n := 0
	for n < maxLen && *(*byte)(unsafe.Add(ptr, n)) != 0 {
		n++
	}

	buf := make([]byte, n)
	copy(buf, unsafe.Slice((*byte)(ptr), n))
	return string(buf)
}

func statusError(status int32, action string) error {
	if status == ffiStatusOK {
		return nil
	}
	return fmt.Errorf("lk: %s failed (status=%d)", action, status)
}

func nilHandleError(action string) error {
	return fmt.Errorf("lk: %s returned nil handle", action)
}

// unsupportedError is returned when an FFI symbol is absent from the loaded
// library, i.e. the embedded/linked lkrequest build predates the feature.
func unsupportedError(action string) error {
	return fmt.Errorf("lk: %s is not supported by the loaded lkrequest library", action)
}

// ffiOutCellPool provides heap-allocated storage for up to eight FFI out
// parameters.
//
// Stack addresses are unsafe for these out parameters because Rust may
// synchronously call back into Go on the calling thread for logging, provider,
// or DNS resolver callbacks. If callback code grows the goroutine stack, Go
// copies and moves the stack (copystack), while C still holds the stale raw
// address. Subsequent writes land on the old stack and are silently lost,
// appearing as zero-valued out parameters such as nil handles. Go heap objects
// are not moved by the GC, so the out cells must live on the heap.
//
// Callers should defer returning the cell to the pool immediately after
// acquireFFIOutCell, then copy slot values into locals before handling errors.
// For out types smaller than eight bytes, such as int32, uint16, or bool, C
// writes the low bytes while the rest of the zeroed slot remains clear, so
// truncating conversions are safe.
var ffiOutCellPool = sync.Pool{New: func() any { return new([8]uintptr) }}

// acquireFFIOutCell returns a zeroed out-parameter cell from the pool.
func acquireFFIOutCell() *[8]uintptr {
	cell := ffiOutCellPool.Get().(*[8]uintptr)
	*cell = [8]uintptr{}
	return cell
}

// ffiPinnedCopy copies v to the heap, pins it, and returns the copy together
// with its unpin function. Callers use it when an FFI function receives a
// struct address, such as SOCKS5 probe configuration or Windows provider and
// DNS resolver structs, so stack movement cannot invalidate the C-visible
// address. T must not contain Go pointers; it may only contain integers, C
// pointers, and other values permitted by the FFI pointer rules.
func ffiPinnedCopy[T any](v T) (*T, func()) {
	p := new(T)
	*p = v
	var pin runtime.Pinner
	pin.Pin(p)
	return p, pin.Unpin
}
