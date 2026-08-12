//go:build darwin && arm64 && !lkcgo

package lkrequest

import _ "embed"

//go:embed lib/darwin_arm64/liblkrequest_ffi.dylib
var embeddedLib []byte

const embeddedLibName = "liblkrequest_ffi.dylib"
