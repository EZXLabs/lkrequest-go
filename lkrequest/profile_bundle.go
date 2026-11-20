package lkrequest

import (
	"runtime"
	"unsafe"
)

// SetTLSProfileJSON overrides the client's TCP TLS fingerprint with a native
// TLS profile: a JSON document of the form of the "tls" layer
// FingerprintInfoJSON reports. The override takes precedence over SetPreset
// and SetPresetJSON whichever is called first; the other layers keep the
// preset's. An empty profileJSON removes it, restoring the preset's TLS
// profile. Build reports a profile that is not valid TLS, input over 1 MiB,
// and a strict ClientHello mirror template, which this API does not accept.
// Requires a library exporting lk_client_builder_set_tls_profile_json.
func (b *ClientBuilder) SetTLSProfileJSON(profileJSON string) *ClientBuilder {
	return b.setProfileJSON(ffi_lk_client_builder_set_tls_profile_json, profileJSON, "client builder set tls profile json")
}

// SetH2ProfileJSON overrides the client's HTTP/2 fingerprint with a native
// H2 profile: a JSON document of the form of the "h2" layer
// FingerprintInfoJSON reports. The override takes precedence over SetPreset,
// SetPresetJSON and SetH2Preset whichever is called first, and leaves header
// order alone. The profile holds data fields only, so a browser preset's
// HTTP/2 runtime behaviour — how frames are grouped into writes, flow control
// and HPACK encoding — gives way to the baseline, and a "behavior" field is
// rejected; a profile bundle loaded with SetPresetJSON carries behaviour too.
// An empty profileJSON removes the override, restoring the H2 profile of
// SetH2Preset if one was set, else of the preset. Build reports an invalid
// profile and input over 1 MiB. Requires a library exporting
// lk_client_builder_set_h2_profile_json.
func (b *ClientBuilder) SetH2ProfileJSON(profileJSON string) *ClientBuilder {
	return b.setProfileJSON(ffi_lk_client_builder_set_h2_profile_json, profileJSON, "client builder set h2 profile json")
}

// SetQUICTLSProfileJSON sets the TLS profile of QUIC connections apart from
// the TCP one, in the JSON form SetTLSProfileJSON takes; the profile must
// enable TLS 1.3. It takes precedence over presets and bundles whichever is
// called first, and neither enables HTTP/3 nor changes the QUIC transport
// profile. An empty profileJSON removes the override, restoring the preset's
// QUIC TLS profile. With ClearQUICFingerprint the last call wins. Requires a
// QUIC/H3-capable library exporting lk_client_builder_set_quic_tls_profile_json.
func (b *ClientBuilder) SetQUICTLSProfileJSON(profileJSON string) *ClientBuilder {
	return b.setProfileJSON(ffi_lk_client_builder_set_quic_tls_profile_json, profileJSON, "client builder set quic tls profile json")
}

// ClearQUICFingerprint makes QUIC connections use the client's effective TCP
// TLS profile, even where the preset carries a separate QUIC TLS profile, and
// FingerprintInfoJSON then reports "quic_tls" as null. SetQUICTLSProfileJSON("")
// instead restores the preset's QUIC TLS profile; between the two the last
// call wins. Requires a QUIC/H3-capable library exporting
// lk_client_builder_clear_quic_fingerprint.
func (b *ClientBuilder) ClearQUICFingerprint() *ClientBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_client_builder_clear_quic_fingerprint == nil {
		return b.setUnsupported("client builder clear quic fingerprint")
	}
	return b.setStatus(ffi_lk_client_builder_clear_quic_fingerprint(b.ptr), "client builder clear quic fingerprint")
}

// SetPresetJSON applies a complete profile bundle in place of a named preset:
// a JSON document with "format": "lkrequest-profile-bundle" and
// "schema_version": 2 carrying every layer a preset does — TCP and QUIC TLS,
// HTTP/2 with its runtime behaviour, QUIC and HTTP/3, header and cookie order,
// and protocol policy. ExportPresetJSON writes one to start from. A later
// SetPreset replaces the bundle, and an empty bundleJSON removes it, restoring
// the named preset. SetTLSProfileJSON, SetH2ProfileJSON, SetH2Preset,
// SetQUICTLSProfileJSON and ClearQUICFingerprint take precedence over its
// layers whichever is called first. Loading performs no I/O and cannot change
// proxy, CA or verification settings. Build reports a document the library
// rejects as an *LkError, with ErrResourceLimitExceeded past the 1 MiB input
// limit and ErrInvalidConfig otherwise; its DiagnosticsJSON holds a
// "preset_error" object with the failure's code, JSON Pointer path and
// message. Requires a library exporting lk_client_builder_set_preset_json.
func (b *ClientBuilder) SetPresetJSON(bundleJSON string) *ClientBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_client_builder_set_preset_json == nil {
		return b.setUnsupported("client builder set preset json")
	}

	jsonPtr, jsonBuf := optionalStringToCString(bundleJSON)
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_client_builder_set_preset_json(b.ptr, jsonPtr, uintptr(len(bundleJSON)), uintptr(unsafe.Pointer(&cell[0])))
	runtime.KeepAlive(jsonBuf)
	return b.setResult(status, cell[0], "client builder set preset json")
}

// SetPresetFile is SetPresetJSON for a bundle read from the file at path.
// Reading stops past the 1 MiB input limit, and the document cannot pull in
// other files. Build reports a file that cannot be read as an *LkError with
// ErrIO, and a document the library rejects as SetPresetJSON does. Requires a
// library exporting lk_client_builder_set_preset_file.
func (b *ClientBuilder) SetPresetFile(path string) *ClientBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_client_builder_set_preset_file == nil {
		return b.setUnsupported("client builder set preset file")
	}

	pathPtr, pathBuf := stringToCString(path)
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_client_builder_set_preset_file(b.ptr, pathPtr, uintptr(len(path)), uintptr(unsafe.Pointer(&cell[0])))
	runtime.KeepAlive(pathBuf)
	return b.setResult(status, cell[0], "client builder set preset file")
}

// ExportPresetJSON returns built-in preset name as a complete profile bundle,
// the document SetPresetJSON and SetPresetFile load: loading it unchanged
// reproduces the preset. Unlike the snapshot GetPresetDetailJSON returns, it
// can be edited and loaded back. It performs no I/O. Requires a library
// exporting lk_preset_export_json.
func ExportPresetJSON(name string) (string, error) {
	if ffi_lk_preset_export_json == nil {
		return "", unsupportedError("preset export json")
	}

	// The first call measures the document, the second copies it and a NUL.
	length, err := exportPresetJSON(name, nil)
	if err != nil {
		return "", err
	}
	out := make([]byte, length+1)
	if length, err = exportPresetJSON(name, out); err != nil {
		return "", err
	}
	return string(out[:length]), nil
}

// exportPresetJSON makes one lk_preset_export_json call, copying the document
// into out unless out is nil, and returns the document's length.
func exportPresetJSON(name string, out []byte) (int, error) {
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	namePtr, nameBuf := stringToCString(name)
	status := ffi_lk_preset_export_json(
		namePtr,
		uintptr(len(name)),
		bytesPtr(out),
		uintptr(len(out)),
		uintptr(unsafe.Pointer(&cell[0])),
		uintptr(unsafe.Pointer(&cell[1])),
	)
	runtime.KeepAlive(nameBuf)
	runtime.KeepAlive(out)
	length, outErr := cell[0], cell[1]

	if err := extractError(outErr); err != nil {
		return 0, err
	}
	if err := statusError(status, "preset export json"); err != nil {
		return 0, err
	}
	return int(length), nil
}

// setProfileJSON passes profileJSON to one of the library's profile setters.
// An empty string goes as a null pointer and zero length, which the library
// takes as removing the override.
func (b *ClientBuilder) setProfileJSON(set func(builder uintptr, jsonPtr uintptr, jsonLen uintptr) int32, profileJSON, action string) *ClientBuilder {
	if !b.check() {
		return b
	}
	if set == nil {
		return b.setUnsupported(action)
	}

	jsonPtr, jsonBuf := optionalStringToCString(profileJSON)
	status := set(b.ptr, jsonPtr, uintptr(len(profileJSON)))
	runtime.KeepAlive(jsonBuf)
	return b.setStatus(status, action)
}

// setResult records the error a library call reported through its error out
// parameter or, without one, the error its status implies.
func (b *ClientBuilder) setResult(status int32, outErr uintptr, action string) *ClientBuilder {
	if err := extractError(outErr); err != nil {
		if b.firstErr == nil {
			b.firstErr = err
		}
		return b
	}
	return b.setStatus(status, action)
}
