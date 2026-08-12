//go:build linux && amd64 && !lkcgo

package lkrequest

import _ "embed"

//go:embed lib/linux_amd64/liblkrequest_ffi.so
var embeddedLib []byte

const embeddedLibName = "liblkrequest_ffi.so"
