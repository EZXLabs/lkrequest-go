package lkrequest

import (
	"runtime"
	"sync"
)

type Multipart struct {
	ptr      uintptr
	firstErr error
	once     sync.Once
}

func NewMultipart() *Multipart {
	m := &Multipart{ptr: ffi_lk_multipart_new()}
	if m.ptr == 0 {
		m.firstErr = nilHandleError("multipart new")
	}

	runtime.SetFinalizer(m, (*Multipart).finalize)
	return m
}

func (m *Multipart) finalize() {
	m.Close()
}

func (m *Multipart) check() bool {
	if m == nil {
		return false
	}
	if m.firstErr != nil {
		return false
	}
	if m.ptr == 0 {
		m.firstErr = ErrNilHandle
		return false
	}
	return true
}

func (m *Multipart) setStatus(status int32, action string) *Multipart {
	if err := statusError(status, action); err != nil && m.firstErr == nil {
		m.firstErr = err
	}
	return m
}

func (m *Multipart) AddText(name, value string) *Multipart {
	if !m.check() {
		return m
	}

	namePtr, nameBuf := stringToCString(name)
	valuePtr, valueBuf := stringToCString(value)
	status := ffi_lk_multipart_add_text(
		m.ptr,
		namePtr, uintptr(len(name)),
		valuePtr, uintptr(len(value)),
	)
	runtime.KeepAlive(nameBuf)
	runtime.KeepAlive(valueBuf)
	return m.setStatus(status, "multipart add text")
}

func (m *Multipart) AddFile(name, filename, contentType string, data []byte) *Multipart {
	if !m.check() {
		return m
	}

	namePtr, nameBuf := stringToCString(name)
	filenamePtr, filenameBuf := stringToCString(filename)
	ctPtr, ctBuf := stringToCString(contentType)
	status := ffi_lk_multipart_add_file(
		m.ptr,
		namePtr, uintptr(len(name)),
		filenamePtr, uintptr(len(filename)),
		ctPtr, uintptr(len(contentType)),
		bytesPtr(data), uintptr(len(data)),
	)
	runtime.KeepAlive(nameBuf)
	runtime.KeepAlive(filenameBuf)
	runtime.KeepAlive(ctBuf)
	runtime.KeepAlive(data)
	return m.setStatus(status, "multipart add file")
}

func (m *Multipart) Close() {
	if m == nil {
		return
	}

	m.once.Do(func() {
		ptr := m.ptr
		m.ptr = 0
		runtime.SetFinalizer(m, nil)
		if ptr != 0 {
			ffi_lk_multipart_free(ptr)
		}
	})
}

func (m *Multipart) detach() (uintptr, error) {
	if m == nil {
		return 0, ErrNilHandle
	}
	if m.firstErr != nil {
		return 0, m.firstErr
	}
	if m.ptr == 0 {
		return 0, ErrNilHandle
	}

	ptr := m.ptr
	m.ptr = 0
	runtime.SetFinalizer(m, nil)
	return ptr, nil
}
