package lkrequest

import (
	"runtime"
	"unsafe"
)

func ListPresetsJSON() (string, error) {
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_preset_list_json(uintptr(unsafe.Pointer(&cell[0])))
	if err := statusError(status, "preset list json"); err != nil {
		return "", err
	}

	return goCString(unsafe.Pointer(cell[0])), nil
}

func GetPresetDetailJSON(name string) (string, error) {
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	namePtr, nameBuf := stringToCString(name)
	status := ffi_lk_preset_get_detail_json(
		namePtr,
		uintptr(len(name)),
		uintptr(unsafe.Pointer(&cell[0])),
		uintptr(unsafe.Pointer(&cell[1])),
	)
	outJSON, outErr := unsafe.Pointer(cell[0]), cell[1]
	runtime.KeepAlive(nameBuf)

	if err := extractError(outErr); err != nil {
		return "", err
	}
	if err := statusError(status, "preset get detail json"); err != nil {
		return "", err
	}
	if outJSON == nil {
		return "", nilHandleError("preset get detail json")
	}

	return goCString(outJSON), nil
}
