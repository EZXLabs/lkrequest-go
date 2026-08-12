//go:build windows && amd64 && !lkcgo

package lkrequest

import _ "embed"

//go:embed lib/windows_amd64/lkrequest_ffi.dll
var embeddedLib []byte

const embeddedLibName = "lkrequest_ffi.dll"
