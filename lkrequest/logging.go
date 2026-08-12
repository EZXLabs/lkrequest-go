package lkrequest

import (
	"errors"
	"runtime"
	"sync/atomic"
)

// ErrLogAlreadyInitialized is returned when logging has already been set up.
//
// The native library installs its subscriber once per process: later calls
// report success but leave the original configuration in place. Reporting the
// no-op instead keeps a second InitLog/InitLogCallback from looking like it
// took effect.
var ErrLogAlreadyInitialized = errors.New("lk: logging is already initialized")

// logInitialized guards both InitLog and InitLogCallback, which share the same
// one-shot initializer on the native side.
var logInitialized atomic.Bool

func InitLog(level string, filePath string) error {
	if !logInitialized.CompareAndSwap(false, true) {
		return ErrLogAlreadyInitialized
	}

	levelPtr, levelBuf := stringToCString(level)
	filePathPtr, filePathBuf := stringToCString(filePath)

	status := ffi_lk_log_init(levelPtr, filePathPtr)
	runtime.KeepAlive(levelBuf)
	runtime.KeepAlive(filePathBuf)

	err := statusError(status, "log init")
	if err != nil {
		logInitialized.Store(false) // let the caller retry
	}
	return err
}
