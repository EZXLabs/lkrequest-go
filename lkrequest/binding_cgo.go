//go:build lkcgo

package lkrequest

// #cgo CFLAGS: -I${SRCDIR}/include
// #cgo windows,amd64  LDFLAGS: -L${SRCDIR}/lib/windows_amd64 -llkrequest_ffi -lws2_32 -lbcrypt -luserenv -lntdll
// #cgo linux,amd64    LDFLAGS: -L${SRCDIR}/lib/linux_amd64 -llkrequest_ffi -lpthread -ldl -lm
// #cgo darwin,arm64   LDFLAGS: -L${SRCDIR}/lib/darwin_arm64 -llkrequest_ffi -lpthread -ldl -lm -framework Security -framework CoreFoundation
// #include "lkrequest.h"
// #include <stdlib.h>
import "C"

import "unsafe"

func cBool(v uintptr) C.bool {
	if v != 0 {
		return C.bool(true)
	}
	return C.bool(false)
}

func init() {
	ffi_lk_abi_version = func() uint32 {
		return uint32(C.lk_abi_version())
	}

	ffi_lk_library_version = func() unsafe.Pointer {
		return unsafe.Pointer(C.lk_library_version())
	}

	ffi_lk_feature_supported = func(name uintptr) uintptr {
		return boolToUintptr(bool(C.lk_feature_supported((*C.char)(unsafe.Pointer(name)))))
	}

	ffi_lk_metrics_snapshot = func(out uintptr) int32 {
		return int32(C.lk_metrics_snapshot((*C.lk_metrics_snapshot_t)(unsafe.Pointer(out))))
	}

	ffi_lk_log_init = func(levelPtr uintptr, filePathPtr uintptr) int32 {
		return int32(C.lk_log_init((*C.char)(unsafe.Pointer(levelPtr)), (*C.char)(unsafe.Pointer(filePathPtr))))
	}

	ffi_lk_preset_list_json = func(outJSONPtr uintptr) int32 {
		return int32(C.lk_preset_list_json((**C.char)(unsafe.Pointer(outJSONPtr))))
	}

	ffi_lk_preset_get_detail_json = func(namePtr uintptr, nameLen uintptr, outJSONPtr uintptr, outErr uintptr) int32 {
		return int32(C.lk_preset_get_detail_json(
			(*C.char)(unsafe.Pointer(namePtr)),
			C.size_t(nameLen),
			(**C.char)(unsafe.Pointer(outJSONPtr)),
			(**C.lk_error_t)(unsafe.Pointer(outErr)),
		))
	}

	ffi_lk_client_builder_add_ca_cert_file = func(builder uintptr, path uintptr) int32 {
		return int32(C.lk_client_builder_add_ca_cert_file((*C.lk_client_builder_t)(unsafe.Pointer(builder)), (*C.char)(unsafe.Pointer(path))))
	}

	ffi_lk_client_builder_add_ca_cert_memory = func(builder uintptr, dataPtr uintptr, dataLen uintptr) int32 {
		return int32(C.lk_client_builder_add_ca_cert_memory((*C.lk_client_builder_t)(unsafe.Pointer(builder)), (*C.uint8_t)(unsafe.Pointer(dataPtr)), C.size_t(dataLen)))
	}

	ffi_lk_client_builder_add_cookie_order = func(builder uintptr, namePtr uintptr, nameLen uintptr) int32 {
		return int32(C.lk_client_builder_add_cookie_order((*C.lk_client_builder_t)(unsafe.Pointer(builder)), (*C.char)(unsafe.Pointer(namePtr)), C.size_t(nameLen)))
	}

	ffi_lk_client_builder_add_default_header = func(builder uintptr, namePtr uintptr, nameLen uintptr, valuePtr uintptr, valueLen uintptr) int32 {
		return int32(C.lk_client_builder_add_default_header(
			(*C.lk_client_builder_t)(unsafe.Pointer(builder)),
			(*C.char)(unsafe.Pointer(namePtr)),
			C.size_t(nameLen),
			(*C.char)(unsafe.Pointer(valuePtr)),
			C.size_t(valueLen),
		))
	}

	ffi_lk_client_builder_add_h3_header_order = func(builder uintptr, namePtr uintptr, nameLen uintptr) int32 {
		return int32(C.lk_client_builder_add_h3_header_order((*C.lk_client_builder_t)(unsafe.Pointer(builder)), (*C.char)(unsafe.Pointer(namePtr)), C.size_t(nameLen)))
	}

	ffi_lk_client_builder_add_header_order = func(builder uintptr, namePtr uintptr, nameLen uintptr) int32 {
		return int32(C.lk_client_builder_add_header_order((*C.lk_client_builder_t)(unsafe.Pointer(builder)), (*C.char)(unsafe.Pointer(namePtr)), C.size_t(nameLen)))
	}

	ffi_lk_client_builder_build = func(builder uintptr, outClient uintptr, outErr uintptr) int32 {
		return int32(C.lk_client_builder_build((*C.lk_client_builder_t)(unsafe.Pointer(builder)), (**C.lk_client_t)(unsafe.Pointer(outClient)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_client_builder_free = func(builder uintptr) {
		C.lk_client_builder_free((*C.lk_client_builder_t)(unsafe.Pointer(builder)))
	}

	ffi_lk_client_builder_new = func() uintptr {
		return uintptr(unsafe.Pointer(C.lk_client_builder_new()))
	}

	ffi_lk_client_builder_set_ech_config = func(builder uintptr, dataPtr uintptr, dataLen uintptr) int32 {
		return int32(C.lk_client_builder_set_ech_config((*C.lk_client_builder_t)(unsafe.Pointer(builder)), (*C.uint8_t)(unsafe.Pointer(dataPtr)), C.size_t(dataLen)))
	}

	ffi_lk_client_builder_set_fallback_h2_to_h1 = func(builder uintptr, enabled uintptr) int32 {
		return int32(C.lk_client_builder_set_fallback_h2_to_h1((*C.lk_client_builder_t)(unsafe.Pointer(builder)), cBool(enabled)))
	}

	ffi_lk_client_builder_set_fallback_proxy_to_direct = func(builder uintptr, enabled uintptr) int32 {
		return int32(C.lk_client_builder_set_fallback_proxy_to_direct((*C.lk_client_builder_t)(unsafe.Pointer(builder)), cBool(enabled)))
	}

	ffi_lk_client_builder_set_h2_preset = func(builder uintptr, presetName uintptr) int32 {
		return int32(C.lk_client_builder_set_h2_preset((*C.lk_client_builder_t)(unsafe.Pointer(builder)), (*C.char)(unsafe.Pointer(presetName))))
	}

	ffi_lk_client_builder_set_keylog_file = func(builder uintptr, pathPtr uintptr, pathLen uintptr) int32 {
		return int32(C.lk_client_builder_set_keylog_file((*C.lk_client_builder_t)(unsafe.Pointer(builder)), (*C.char)(unsafe.Pointer(pathPtr)), C.size_t(pathLen)))
	}

	ffi_lk_client_builder_set_max_connections_per_session = func(builder uintptr, maxConnections uintptr) int32 {
		return int32(C.lk_client_builder_set_max_connections_per_session((*C.lk_client_builder_t)(unsafe.Pointer(builder)), C.size_t(maxConnections)))
	}

	ffi_lk_client_builder_set_max_header_count = func(builder uintptr, count uintptr) int32 {
		return int32(C.lk_client_builder_set_max_header_count((*C.lk_client_builder_t)(unsafe.Pointer(builder)), C.size_t(count)))
	}

	ffi_lk_client_builder_set_max_header_size = func(builder uintptr, size uintptr) int32 {
		return int32(C.lk_client_builder_set_max_header_size((*C.lk_client_builder_t)(unsafe.Pointer(builder)), C.size_t(size)))
	}

	ffi_lk_client_builder_set_max_headers_total_size = func(builder uintptr, size uintptr) int32 {
		return int32(C.lk_client_builder_set_max_headers_total_size((*C.lk_client_builder_t)(unsafe.Pointer(builder)), C.size_t(size)))
	}

	ffi_lk_client_builder_set_max_outstanding_ops = func(builder uintptr, maxOutstandingOps uintptr) int32 {
		return int32(C.lk_client_builder_set_max_outstanding_ops((*C.lk_client_builder_t)(unsafe.Pointer(builder)), C.size_t(maxOutstandingOps)))
	}

	ffi_lk_client_builder_set_max_response_body_size = func(builder uintptr, size uintptr) int32 {
		return int32(C.lk_client_builder_set_max_response_body_size((*C.lk_client_builder_t)(unsafe.Pointer(builder)), C.size_t(size)))
	}

	ffi_lk_client_builder_set_min_transfer_rate = func(builder uintptr, bytesPerSec uintptr, windowMs uint64) int32 {
		return int32(C.lk_client_builder_set_min_transfer_rate((*C.lk_client_builder_t)(unsafe.Pointer(builder)), C.size_t(bytesPerSec), C.uint64_t(windowMs)))
	}

	ffi_lk_client_builder_set_preset = func(builder uintptr, presetName uintptr) int32 {
		return int32(C.lk_client_builder_set_preset((*C.lk_client_builder_t)(unsafe.Pointer(builder)), (*C.char)(unsafe.Pointer(presetName))))
	}

	ffi_lk_client_builder_set_retry_on_connection_close = func(builder uintptr, enabled uintptr) int32 {
		return int32(C.lk_client_builder_set_retry_on_connection_close((*C.lk_client_builder_t)(unsafe.Pointer(builder)), cBool(enabled)))
	}

	ffi_lk_client_builder_set_tcp_fingerprint_ja4t = func(builder uintptr, ja4tPtr uintptr, ja4tLen uintptr) int32 {
		return int32(C.lk_client_builder_set_tcp_fingerprint_ja4t((*C.lk_client_builder_t)(unsafe.Pointer(builder)), (*C.char)(unsafe.Pointer(ja4tPtr)), C.size_t(ja4tLen)))
	}

	ffi_lk_client_builder_set_timeout_dns = func(builder uintptr, timeoutMs uint64) int32 {
		return int32(C.lk_client_builder_set_timeout_dns((*C.lk_client_builder_t)(unsafe.Pointer(builder)), C.uint64_t(timeoutMs)))
	}

	ffi_lk_client_builder_set_timeout_tcp_connect = func(builder uintptr, timeoutMs uint64) int32 {
		return int32(C.lk_client_builder_set_timeout_tcp_connect((*C.lk_client_builder_t)(unsafe.Pointer(builder)), C.uint64_t(timeoutMs)))
	}

	ffi_lk_client_builder_set_timeout_tls_handshake = func(builder uintptr, timeoutMs uint64) int32 {
		return int32(C.lk_client_builder_set_timeout_tls_handshake((*C.lk_client_builder_t)(unsafe.Pointer(builder)), C.uint64_t(timeoutMs)))
	}

	ffi_lk_client_builder_set_timeout_total = func(builder uintptr, timeoutMs uint64) int32 {
		return int32(C.lk_client_builder_set_timeout_total((*C.lk_client_builder_t)(unsafe.Pointer(builder)), C.uint64_t(timeoutMs)))
	}

	ffi_lk_client_builder_set_timeout_ttfb = func(builder uintptr, timeoutMs uint64) int32 {
		return int32(C.lk_client_builder_set_timeout_ttfb((*C.lk_client_builder_t)(unsafe.Pointer(builder)), C.uint64_t(timeoutMs)))
	}

	ffi_lk_client_builder_set_verify = func(builder uintptr, enabled uintptr) int32 {
		return int32(C.lk_client_builder_set_verify((*C.lk_client_builder_t)(unsafe.Pointer(builder)), cBool(enabled)))
	}

	ffi_lk_client_new = func(presetName uintptr, outClient uintptr, outErr uintptr) int32 {
		return int32(C.lk_client_new((*C.char)(unsafe.Pointer(presetName)), (**C.lk_client_t)(unsafe.Pointer(outClient)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_client_new_default = func(outClient uintptr, outErr uintptr) int32 {
		return int32(C.lk_client_new_default((**C.lk_client_t)(unsafe.Pointer(outClient)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_client_clone = func(client uintptr) uintptr {
		return uintptr(unsafe.Pointer(C.lk_client_clone((*C.lk_client_t)(unsafe.Pointer(client)))))
	}

	ffi_lk_client_fingerprint_info_json = func(client uintptr, outJSONPtr uintptr, outJSONLen uintptr, outErr uintptr) int32 {
		return int32(C.lk_client_fingerprint_info_json(
			(*C.lk_client_t)(unsafe.Pointer(client)),
			(**C.char)(unsafe.Pointer(outJSONPtr)),
			(*C.size_t)(unsafe.Pointer(outJSONLen)),
			(**C.lk_error_t)(unsafe.Pointer(outErr)),
		))
	}

	ffi_lk_client_free = func(client uintptr) {
		C.lk_client_free((*C.lk_client_t)(unsafe.Pointer(client)))
	}

	ffi_lk_session_builder_build = func(builder uintptr, outSession uintptr, outErr uintptr) int32 {
		return int32(C.lk_session_builder_build((*C.lk_session_builder_t)(unsafe.Pointer(builder)), (**C.lk_session_t)(unsafe.Pointer(outSession)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_session_builder_free = func(builder uintptr) {
		C.lk_session_builder_free((*C.lk_session_builder_t)(unsafe.Pointer(builder)))
	}

	ffi_lk_session_builder_new = func(client uintptr) uintptr {
		return uintptr(unsafe.Pointer(C.lk_session_builder_new((*C.lk_client_t)(unsafe.Pointer(client)))))
	}

	ffi_lk_session_builder_set_default_accept_encoding = func(builder uintptr, encodingBits uint8) int32 {
		return int32(C.lk_session_builder_set_default_accept_encoding((*C.lk_session_builder_t)(unsafe.Pointer(builder)), C.uint8_t(encodingBits)))
	}

	ffi_lk_session_builder_disable_redirects = func(builder uintptr) int32 {
		return int32(C.lk_session_builder_disable_redirects((*C.lk_session_builder_t)(unsafe.Pointer(builder))))
	}

	ffi_lk_session_builder_set_ech_config = func(builder uintptr, dataPtr uintptr, dataLen uintptr) int32 {
		return int32(C.lk_session_builder_set_ech_config((*C.lk_session_builder_t)(unsafe.Pointer(builder)), (*C.uint8_t)(unsafe.Pointer(dataPtr)), C.size_t(dataLen)))
	}

	ffi_lk_session_builder_set_http1_only = func(builder uintptr) int32 {
		return int32(C.lk_session_builder_set_http1_only((*C.lk_session_builder_t)(unsafe.Pointer(builder))))
	}

	ffi_lk_session_builder_set_http2_only = func(builder uintptr) int32 {
		return int32(C.lk_session_builder_set_http2_only((*C.lk_session_builder_t)(unsafe.Pointer(builder))))
	}

	ffi_lk_session_builder_set_http3_only = func(builder uintptr) int32 {
		return int32(C.lk_session_builder_set_http3_only((*C.lk_session_builder_t)(unsafe.Pointer(builder))))
	}

	ffi_lk_session_builder_set_idle_timeout = func(builder uintptr, idleTimeoutMs uint64) int32 {
		return int32(C.lk_session_builder_set_idle_timeout((*C.lk_session_builder_t)(unsafe.Pointer(builder)), C.uint64_t(idleTimeoutMs)))
	}

	ffi_lk_session_builder_set_max_connections = func(builder uintptr, maxConnections uintptr) int32 {
		return int32(C.lk_session_builder_set_max_connections((*C.lk_session_builder_t)(unsafe.Pointer(builder)), C.size_t(maxConnections)))
	}

	ffi_lk_session_builder_set_max_redirects = func(builder uintptr, maxRedirects uint32) int32 {
		return int32(C.lk_session_builder_set_max_redirects((*C.lk_session_builder_t)(unsafe.Pointer(builder)), C.uint32_t(maxRedirects)))
	}

	ffi_lk_session_builder_set_proxy = func(builder uintptr, proxyPtr uintptr, proxyLen uintptr) int32 {
		return int32(C.lk_session_builder_set_proxy((*C.lk_session_builder_t)(unsafe.Pointer(builder)), (*C.char)(unsafe.Pointer(proxyPtr)), C.size_t(proxyLen)))
	}

	ffi_lk_session_builder_set_retry_exponential = func(builder uintptr, maxRetries uint32, baseDelayMs uint64, maxDelayMs uint64, jitter uintptr) int32 {
		return int32(C.lk_session_builder_set_retry_exponential(
			(*C.lk_session_builder_t)(unsafe.Pointer(builder)),
			C.uint32_t(maxRetries),
			C.uint64_t(baseDelayMs),
			C.uint64_t(maxDelayMs),
			cBool(jitter),
		))
	}

	ffi_lk_session_builder_set_retry_fixed = func(builder uintptr, maxRetries uint32, intervalMs uint64) int32 {
		return int32(C.lk_session_builder_set_retry_fixed((*C.lk_session_builder_t)(unsafe.Pointer(builder)), C.uint32_t(maxRetries), C.uint64_t(intervalMs)))
	}

	ffi_lk_session_builder_add_cookie_order = func(builder uintptr, namePtr uintptr, nameLen uintptr) int32 {
		return int32(C.lk_session_builder_add_cookie_order((*C.lk_session_builder_t)(unsafe.Pointer(builder)), (*C.char)(unsafe.Pointer(namePtr)), C.size_t(nameLen)))
	}

	ffi_lk_session_builder_add_h3_header_order = func(builder uintptr, namePtr uintptr, nameLen uintptr) int32 {
		return int32(C.lk_session_builder_add_h3_header_order((*C.lk_session_builder_t)(unsafe.Pointer(builder)), (*C.char)(unsafe.Pointer(namePtr)), C.size_t(nameLen)))
	}

	ffi_lk_session_builder_add_header_order = func(builder uintptr, namePtr uintptr, nameLen uintptr) int32 {
		return int32(C.lk_session_builder_add_header_order((*C.lk_session_builder_t)(unsafe.Pointer(builder)), (*C.char)(unsafe.Pointer(namePtr)), C.size_t(nameLen)))
	}

	ffi_lk_session_new = func(client uintptr, outSession uintptr, outErr uintptr) int32 {
		return int32(C.lk_session_new((*C.lk_client_t)(unsafe.Pointer(client)), (**C.lk_session_t)(unsafe.Pointer(outSession)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_session_new_with_config = func(client uintptr, proxyPtr uintptr, proxyLen uintptr, maxRedirects uint32, outSession uintptr, outErr uintptr) int32 {
		return int32(C.lk_session_new_with_config(
			(*C.lk_client_t)(unsafe.Pointer(client)),
			(*C.char)(unsafe.Pointer(proxyPtr)),
			C.size_t(proxyLen),
			C.uint32_t(maxRedirects),
			(**C.lk_session_t)(unsafe.Pointer(outSession)),
			(**C.lk_error_t)(unsafe.Pointer(outErr)),
		))
	}

	ffi_lk_session_clone = func(session uintptr) uintptr {
		return uintptr(unsafe.Pointer(C.lk_session_clone((*C.lk_session_t)(unsafe.Pointer(session)))))
	}

	ffi_lk_session_free = func(session uintptr) {
		C.lk_session_free((*C.lk_session_t)(unsafe.Pointer(session)))
	}

	ffi_lk_request_add_header = func(request uintptr, namePtr uintptr, nameLen uintptr, valuePtr uintptr, valueLen uintptr) int32 {
		return int32(C.lk_request_add_header(
			(*C.lk_request_t)(unsafe.Pointer(request)),
			(*C.char)(unsafe.Pointer(namePtr)),
			C.size_t(nameLen),
			(*C.char)(unsafe.Pointer(valuePtr)),
			C.size_t(valueLen),
		))
	}

	ffi_lk_request_add_cookie_order = func(request uintptr, namePtr uintptr, nameLen uintptr) int32 {
		return int32(C.lk_request_add_cookie_order((*C.lk_request_t)(unsafe.Pointer(request)), (*C.char)(unsafe.Pointer(namePtr)), C.size_t(nameLen)))
	}

	ffi_lk_request_add_h3_header_order = func(request uintptr, namePtr uintptr, nameLen uintptr) int32 {
		return int32(C.lk_request_add_h3_header_order((*C.lk_request_t)(unsafe.Pointer(request)), (*C.char)(unsafe.Pointer(namePtr)), C.size_t(nameLen)))
	}

	ffi_lk_request_add_header_order = func(request uintptr, namePtr uintptr, nameLen uintptr) int32 {
		return int32(C.lk_request_add_header_order((*C.lk_request_t)(unsafe.Pointer(request)), (*C.char)(unsafe.Pointer(namePtr)), C.size_t(nameLen)))
	}

	ffi_lk_request_add_query = func(request uintptr, keyPtr uintptr, keyLen uintptr, valuePtr uintptr, valueLen uintptr) int32 {
		return int32(C.lk_request_add_query(
			(*C.lk_request_t)(unsafe.Pointer(request)),
			(*C.char)(unsafe.Pointer(keyPtr)),
			C.size_t(keyLen),
			(*C.char)(unsafe.Pointer(valuePtr)),
			C.size_t(valueLen),
		))
	}

	ffi_lk_request_free = func(request uintptr) {
		C.lk_request_free((*C.lk_request_t)(unsafe.Pointer(request)))
	}

	ffi_lk_request_new = func(session uintptr, methodPtr uintptr, methodLen uintptr, urlPtr uintptr, urlLen uintptr, outRequest uintptr, outErr uintptr) int32 {
		return int32(C.lk_request_new(
			(*C.lk_session_t)(unsafe.Pointer(session)),
			(*C.char)(unsafe.Pointer(methodPtr)),
			C.size_t(methodLen),
			(*C.char)(unsafe.Pointer(urlPtr)),
			C.size_t(urlLen),
			(**C.lk_request_t)(unsafe.Pointer(outRequest)),
			(**C.lk_error_t)(unsafe.Pointer(outErr)),
		))
	}

	ffi_lk_request_send = func(request uintptr, outResponse uintptr, outErr uintptr) int32 {
		return int32(C.lk_request_send((*C.lk_request_t)(unsafe.Pointer(request)), (**C.lk_response_t)(unsafe.Pointer(outResponse)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_request_send_async = func(request uintptr, outOp uintptr, outErr uintptr) int32 {
		return int32(C.lk_request_send_async((*C.lk_request_t)(unsafe.Pointer(request)), (**C.lk_op_t)(unsafe.Pointer(outOp)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_request_send_streaming = func(request uintptr, outStream uintptr, outErr uintptr) int32 {
		return int32(C.lk_request_send_streaming((*C.lk_request_t)(unsafe.Pointer(request)), (**C.lk_streaming_response_t)(unsafe.Pointer(outStream)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_request_send_streaming_async = func(request uintptr, outOp uintptr, outErr uintptr) int32 {
		return int32(C.lk_request_send_streaming_async((*C.lk_request_t)(unsafe.Pointer(request)), (**C.lk_op_t)(unsafe.Pointer(outOp)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_request_set_accept_encoding = func(request uintptr, encodingBits uint8) int32 {
		return int32(C.lk_request_set_accept_encoding((*C.lk_request_t)(unsafe.Pointer(request)), C.uint8_t(encodingBits)))
	}

	ffi_lk_request_set_auto_decompress = func(request uintptr, enabled uintptr) int32 {
		return int32(C.lk_request_set_auto_decompress((*C.lk_request_t)(unsafe.Pointer(request)), cBool(enabled)))
	}

	ffi_lk_request_set_basic_auth = func(request uintptr, usernamePtr uintptr, usernameLen uintptr, passwordPtr uintptr, passwordLen uintptr) int32 {
		return int32(C.lk_request_set_basic_auth(
			(*C.lk_request_t)(unsafe.Pointer(request)),
			(*C.char)(unsafe.Pointer(usernamePtr)),
			C.size_t(usernameLen),
			(*C.char)(unsafe.Pointer(passwordPtr)),
			C.size_t(passwordLen),
		))
	}

	ffi_lk_request_set_bearer_auth = func(request uintptr, tokenPtr uintptr, tokenLen uintptr) int32 {
		return int32(C.lk_request_set_bearer_auth((*C.lk_request_t)(unsafe.Pointer(request)), (*C.char)(unsafe.Pointer(tokenPtr)), C.size_t(tokenLen)))
	}

	ffi_lk_request_set_body_bytes = func(request uintptr, dataPtr uintptr, dataLen uintptr) int32 {
		return int32(C.lk_request_set_body_bytes((*C.lk_request_t)(unsafe.Pointer(request)), (*C.uint8_t)(unsafe.Pointer(dataPtr)), C.size_t(dataLen)))
	}

	ffi_lk_request_set_cookie = func(request uintptr, namePtr uintptr, nameLen uintptr, valuePtr uintptr, valueLen uintptr) int32 {
		return int32(C.lk_request_set_cookie(
			(*C.lk_request_t)(unsafe.Pointer(request)),
			(*C.char)(unsafe.Pointer(namePtr)),
			C.size_t(nameLen),
			(*C.char)(unsafe.Pointer(valuePtr)),
			C.size_t(valueLen),
		))
	}

	ffi_lk_request_set_form = func(request uintptr, pairsPtr uintptr, pairsCount uintptr) int32 {
		return int32(C.lk_request_set_form((*C.lk_request_t)(unsafe.Pointer(request)), (*C.lk_form_pair_t)(unsafe.Pointer(pairsPtr)), C.size_t(pairsCount)))
	}

	ffi_lk_request_set_json_text = func(request uintptr, jsonPtr uintptr, jsonLen uintptr) int32 {
		return int32(C.lk_request_set_json_text((*C.lk_request_t)(unsafe.Pointer(request)), (*C.char)(unsafe.Pointer(jsonPtr)), C.size_t(jsonLen)))
	}

	ffi_lk_request_set_proxy = func(request uintptr, proxyPtr uintptr, proxyLen uintptr) int32 {
		return int32(C.lk_request_set_proxy((*C.lk_request_t)(unsafe.Pointer(request)), (*C.char)(unsafe.Pointer(proxyPtr)), C.size_t(proxyLen)))
	}

	ffi_lk_request_set_text_body = func(request uintptr, textPtr uintptr, textLen uintptr) int32 {
		return int32(C.lk_request_set_text_body((*C.lk_request_t)(unsafe.Pointer(request)), (*C.char)(unsafe.Pointer(textPtr)), C.size_t(textLen)))
	}

	ffi_lk_request_set_timeout = func(request uintptr, timeoutMs uint64) int32 {
		return int32(C.lk_request_set_timeout((*C.lk_request_t)(unsafe.Pointer(request)), C.uint64_t(timeoutMs)))
	}

	ffi_lk_request_set_version = func(request uintptr, version int32) int32 {
		return int32(C.lk_request_set_version((*C.lk_request_t)(unsafe.Pointer(request)), C.lk_http_version_t(version)))
	}

	ffi_lk_response_body = func(response uintptr, outPtr uintptr, outLen uintptr) int32 {
		return int32(C.lk_response_body((*C.lk_response_t)(unsafe.Pointer(response)), (**C.uint8_t)(unsafe.Pointer(outPtr)), (*C.size_t)(unsafe.Pointer(outLen))))
	}

	ffi_lk_response_content_length = func(response uintptr) int64 {
		return int64(C.lk_response_content_length((*C.lk_response_t)(unsafe.Pointer(response))))
	}

	ffi_lk_response_copy_body = func(response uintptr, buf uintptr, bufLen uintptr, outReadLen uintptr) int32 {
		return int32(C.lk_response_copy_body((*C.lk_response_t)(unsafe.Pointer(response)), (*C.uint8_t)(unsafe.Pointer(buf)), C.size_t(bufLen), (*C.size_t)(unsafe.Pointer(outReadLen))))
	}

	ffi_lk_response_free = func(response uintptr) {
		C.lk_response_free((*C.lk_response_t)(unsafe.Pointer(response)))
	}

	ffi_lk_response_get_diagnostics_json = func(response uintptr, outPtr uintptr) int32 {
		return int32(C.lk_response_get_diagnostics_json((*C.lk_response_t)(unsafe.Pointer(response)), (**C.char)(unsafe.Pointer(outPtr))))
	}

	ffi_lk_response_get_header_by_name = func(response uintptr, namePtr uintptr, nameLen uintptr, outPtr uintptr, outLen uintptr, outErr uintptr) int32 {
		return int32(C.lk_response_get_header_by_name(
			(*C.lk_response_t)(unsafe.Pointer(response)),
			(*C.char)(unsafe.Pointer(namePtr)),
			C.size_t(nameLen),
			(**C.uint8_t)(unsafe.Pointer(outPtr)),
			(*C.size_t)(unsafe.Pointer(outLen)),
			(**C.lk_error_t)(unsafe.Pointer(outErr)),
		))
	}

	ffi_lk_response_header_count = func(response uintptr) uintptr {
		return uintptr(C.lk_response_header_count((*C.lk_response_t)(unsafe.Pointer(response))))
	}

	ffi_lk_response_header_name_at = func(response uintptr, index uintptr, outPtr uintptr, outLen uintptr) int32 {
		return int32(C.lk_response_header_name_at((*C.lk_response_t)(unsafe.Pointer(response)), C.size_t(index), (**C.char)(unsafe.Pointer(outPtr)), (*C.size_t)(unsafe.Pointer(outLen))))
	}

	ffi_lk_response_header_value_at = func(response uintptr, index uintptr, outPtr uintptr, outLen uintptr) int32 {
		return int32(C.lk_response_header_value_at((*C.lk_response_t)(unsafe.Pointer(response)), C.size_t(index), (**C.uint8_t)(unsafe.Pointer(outPtr)), (*C.size_t)(unsafe.Pointer(outLen))))
	}

	ffi_lk_response_status = func(response uintptr) uint16 {
		return uint16(C.lk_response_status((*C.lk_response_t)(unsafe.Pointer(response))))
	}

	ffi_lk_response_url = func(response uintptr, outPtr uintptr, outLen uintptr) int32 {
		return int32(C.lk_response_url((*C.lk_response_t)(unsafe.Pointer(response)), (**C.char)(unsafe.Pointer(outPtr)), (*C.size_t)(unsafe.Pointer(outLen))))
	}

	ffi_lk_response_version = func(response uintptr, outVersion uintptr) int32 {
		return int32(C.lk_response_version((*C.lk_response_t)(unsafe.Pointer(response)), (*C.lk_http_version_t)(unsafe.Pointer(outVersion))))
	}

	ffi_lk_streaming_response_free = func(stream uintptr) {
		C.lk_streaming_response_free((*C.lk_streaming_response_t)(unsafe.Pointer(stream)))
	}

	ffi_lk_streaming_response_header_count = func(stream uintptr) uintptr {
		return uintptr(C.lk_streaming_response_header_count((*C.lk_streaming_response_t)(unsafe.Pointer(stream))))
	}

	ffi_lk_streaming_response_header_name_at = func(stream uintptr, index uintptr, outPtr uintptr, outLen uintptr) int32 {
		return int32(C.lk_streaming_response_header_name_at((*C.lk_streaming_response_t)(unsafe.Pointer(stream)), C.size_t(index), (**C.char)(unsafe.Pointer(outPtr)), (*C.size_t)(unsafe.Pointer(outLen))))
	}

	ffi_lk_streaming_response_header_value_at = func(stream uintptr, index uintptr, outPtr uintptr, outLen uintptr) int32 {
		return int32(C.lk_streaming_response_header_value_at((*C.lk_streaming_response_t)(unsafe.Pointer(stream)), C.size_t(index), (**C.uint8_t)(unsafe.Pointer(outPtr)), (*C.size_t)(unsafe.Pointer(outLen))))
	}

	ffi_lk_streaming_response_status = func(stream uintptr) uint16 {
		return uint16(C.lk_streaming_response_status((*C.lk_streaming_response_t)(unsafe.Pointer(stream))))
	}

	ffi_lk_stream_close = func(stream uintptr) int32 {
		return int32(C.lk_stream_close((*C.lk_streaming_response_t)(unsafe.Pointer(stream))))
	}

	ffi_lk_stream_copy_chunk = func(stream uintptr, buf uintptr, bufLen uintptr, outReadLen uintptr) int32 {
		return int32(C.lk_stream_copy_chunk((*C.lk_streaming_response_t)(unsafe.Pointer(stream)), (*C.uint8_t)(unsafe.Pointer(buf)), C.size_t(bufLen), (*C.size_t)(unsafe.Pointer(outReadLen))))
	}

	ffi_lk_stream_read = func(stream uintptr, outChunk uintptr, outErr uintptr) int32 {
		return int32(C.lk_stream_read((*C.lk_streaming_response_t)(unsafe.Pointer(stream)), (*C.lk_chunk_view_t)(unsafe.Pointer(outChunk)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_stream_read_async = func(stream uintptr, outOp uintptr, outErr uintptr) int32 {
		return int32(C.lk_stream_read_async((*C.lk_streaming_response_t)(unsafe.Pointer(stream)), (**C.lk_op_t)(unsafe.Pointer(outOp)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_op_cancel = func(op uintptr) int32 {
		return int32(C.lk_op_cancel((*C.lk_op_t)(unsafe.Pointer(op))))
	}

	ffi_lk_op_free = func(op uintptr) {
		C.lk_op_free((*C.lk_op_t)(unsafe.Pointer(op)))
	}

	ffi_lk_op_poll = func(op uintptr) int32 {
		return int32(C.lk_op_poll((*C.lk_op_t)(unsafe.Pointer(op))))
	}

	ffi_lk_op_take_chunk = func(op uintptr, outChunk uintptr, outErr uintptr) int32 {
		return int32(C.lk_op_take_chunk((*C.lk_op_t)(unsafe.Pointer(op)), (*C.lk_chunk_view_t)(unsafe.Pointer(outChunk)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_op_take_error = func(op uintptr, outErr uintptr) int32 {
		return int32(C.lk_op_take_error((*C.lk_op_t)(unsafe.Pointer(op)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_op_take_response = func(op uintptr, outResponse uintptr, outErr uintptr) int32 {
		return int32(C.lk_op_take_response((*C.lk_op_t)(unsafe.Pointer(op)), (**C.lk_response_t)(unsafe.Pointer(outResponse)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_op_take_streaming_response = func(op uintptr, outStream uintptr, outErr uintptr) int32 {
		return int32(C.lk_op_take_streaming_response((*C.lk_op_t)(unsafe.Pointer(op)), (**C.lk_streaming_response_t)(unsafe.Pointer(outStream)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_op_wait = func(op uintptr, timeoutMs uint64) int32 {
		return int32(C.lk_op_wait((*C.lk_op_t)(unsafe.Pointer(op)), C.uint64_t(timeoutMs)))
	}

	ffi_lk_error_code = func(err uintptr) int32 {
		return int32(C.lk_error_code((*C.lk_error_t)(unsafe.Pointer(err))))
	}

	ffi_lk_error_free = func(err uintptr) {
		C.lk_error_free((*C.lk_error_t)(unsafe.Pointer(err)))
	}

	ffi_lk_error_get_diagnostics_json = func(err uintptr, outPtr uintptr) int32 {
		return int32(C.lk_error_get_diagnostics_json((*C.lk_error_t)(unsafe.Pointer(err)), (**C.char)(unsafe.Pointer(outPtr))))
	}

	ffi_lk_error_http_status = func(err uintptr) int32 {
		return int32(C.lk_error_http_status((*C.lk_error_t)(unsafe.Pointer(err))))
	}

	ffi_lk_error_is_retryable = func(err uintptr) uintptr {
		return boolToUintptr(bool(C.lk_error_is_retryable((*C.lk_error_t)(unsafe.Pointer(err)))))
	}

	ffi_lk_error_message = func(err uintptr, outPtr uintptr, outLen uintptr) int32 {
		return int32(C.lk_error_message((*C.lk_error_t)(unsafe.Pointer(err)), (**C.char)(unsafe.Pointer(outPtr)), (*C.size_t)(unsafe.Pointer(outLen))))
	}

	ffi_lk_error_phase = func(err uintptr) int32 {
		return int32(C.lk_error_phase((*C.lk_error_t)(unsafe.Pointer(err))))
	}

	// Client builder - new
	ffi_lk_client_builder_set_dns = func(builder uintptr, dnsConfig int32) int32 {
		return int32(C.lk_client_builder_set_dns((*C.lk_client_builder_t)(unsafe.Pointer(builder)), C.lk_dns_config_t(dnsConfig)))
	}

	ffi_lk_client_builder_set_dns_custom = func(builder uintptr, addrPtr uintptr, addrLen uintptr) int32 {
		return int32(C.lk_client_builder_set_dns_custom((*C.lk_client_builder_t)(unsafe.Pointer(builder)), (*C.char)(unsafe.Pointer(addrPtr)), C.size_t(addrLen)))
	}

	ffi_lk_client_builder_set_use_native_certs = func(builder uintptr, enabled uintptr) int32 {
		return int32(C.lk_client_builder_set_use_native_certs((*C.lk_client_builder_t)(unsafe.Pointer(builder)), cBool(enabled)))
	}

	// Session cookie management
	ffi_lk_session_set_cookie = func(session uintptr, urlPtr uintptr, urlLen uintptr, namePtr uintptr, nameLen uintptr, valuePtr uintptr, valueLen uintptr) int32 {
		return int32(C.lk_session_set_cookie((*C.lk_session_t)(unsafe.Pointer(session)), (*C.char)(unsafe.Pointer(urlPtr)), C.size_t(urlLen), (*C.char)(unsafe.Pointer(namePtr)), C.size_t(nameLen), (*C.char)(unsafe.Pointer(valuePtr)), C.size_t(valueLen)))
	}

	ffi_lk_session_set_cookie_with_attrs = func(session uintptr, urlPtr uintptr, urlLen uintptr, namePtr uintptr, nameLen uintptr, valuePtr uintptr, valueLen uintptr, pathPtr uintptr, pathLen uintptr, domainPtr uintptr, domainLen uintptr, secure bool, httpOnly bool, outErr uintptr) int32 {
		return int32(C.lk_session_set_cookie_with_attrs(
			(*C.lk_session_t)(unsafe.Pointer(session)),
			(*C.char)(unsafe.Pointer(urlPtr)),
			C.size_t(urlLen),
			(*C.char)(unsafe.Pointer(namePtr)),
			C.size_t(nameLen),
			(*C.char)(unsafe.Pointer(valuePtr)),
			C.size_t(valueLen),
			(*C.char)(unsafe.Pointer(pathPtr)),
			C.size_t(pathLen),
			(*C.char)(unsafe.Pointer(domainPtr)),
			C.size_t(domainLen),
			cBool(boolToUintptr(secure)),
			cBool(boolToUintptr(httpOnly)),
			(**C.lk_error_t)(unsafe.Pointer(outErr)),
		))
	}

	ffi_lk_session_remove_cookie = func(session uintptr, urlPtr uintptr, urlLen uintptr, namePtr uintptr, nameLen uintptr) int32 {
		return int32(C.lk_session_remove_cookie((*C.lk_session_t)(unsafe.Pointer(session)), (*C.char)(unsafe.Pointer(urlPtr)), C.size_t(urlLen), (*C.char)(unsafe.Pointer(namePtr)), C.size_t(nameLen)))
	}

	ffi_lk_session_get_cookie = func(session uintptr, urlPtr uintptr, urlLen uintptr, namePtr uintptr, nameLen uintptr, outValuePtr uintptr, outValueLen uintptr, outErr uintptr) int32 {
		return int32(C.lk_session_get_cookie((*C.lk_session_t)(unsafe.Pointer(session)), (*C.char)(unsafe.Pointer(urlPtr)), C.size_t(urlLen), (*C.char)(unsafe.Pointer(namePtr)), C.size_t(nameLen), (**C.char)(unsafe.Pointer(outValuePtr)), (*C.size_t)(unsafe.Pointer(outValueLen)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_session_get_cookies_json = func(session uintptr, urlPtr uintptr, urlLen uintptr, outJSONPtr uintptr) int32 {
		return int32(C.lk_session_get_cookies_json((*C.lk_session_t)(unsafe.Pointer(session)), (*C.char)(unsafe.Pointer(urlPtr)), C.size_t(urlLen), (**C.char)(unsafe.Pointer(outJSONPtr))))
	}

	ffi_lk_session_clear_cookies = func(session uintptr) int32 {
		return int32(C.lk_session_clear_cookies((*C.lk_session_t)(unsafe.Pointer(session))))
	}

	// Session preconnect
	ffi_lk_session_preconnect = func(session uintptr, urlPtr uintptr, urlLen uintptr, outErr uintptr) int32 {
		return int32(C.lk_session_preconnect((*C.lk_session_t)(unsafe.Pointer(session)), (*C.char)(unsafe.Pointer(urlPtr)), C.size_t(urlLen), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_session_preconnect_async = func(session uintptr, urlPtr uintptr, urlLen uintptr, outOp uintptr, outErr uintptr) int32 {
		return int32(C.lk_session_preconnect_async((*C.lk_session_t)(unsafe.Pointer(session)), (*C.char)(unsafe.Pointer(urlPtr)), C.size_t(urlLen), (**C.lk_op_t)(unsafe.Pointer(outOp)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	// Session connection pool
	ffi_lk_session_connection_pool_stats = func(session uintptr, outH2 uintptr, outH1 uintptr, outTotal uintptr, outMax uintptr, outAtCapacity uintptr) int32 {
		return int32(C.lk_session_connection_pool_stats((*C.lk_session_t)(unsafe.Pointer(session)), (*C.size_t)(unsafe.Pointer(outH2)), (*C.size_t)(unsafe.Pointer(outH1)), (*C.size_t)(unsafe.Pointer(outTotal)), (*C.size_t)(unsafe.Pointer(outMax)), (*C.bool)(unsafe.Pointer(outAtCapacity))))
	}

	ffi_lk_session_connection_pool_clear = func(session uintptr) int32 {
		return int32(C.lk_session_connection_pool_clear((*C.lk_session_t)(unsafe.Pointer(session))))
	}

	// Request - new methods
	ffi_lk_request_set_cookie_override = func(request uintptr, namePtr uintptr, nameLen uintptr, valuePtr uintptr, valueLen uintptr) int32 {
		return int32(C.lk_request_set_cookie_override((*C.lk_request_t)(unsafe.Pointer(request)), (*C.char)(unsafe.Pointer(namePtr)), C.size_t(nameLen), (*C.char)(unsafe.Pointer(valuePtr)), C.size_t(valueLen)))
	}

	ffi_lk_request_set_multipart = func(request uintptr, multipart uintptr) int32 {
		return int32(C.lk_request_set_multipart((*C.lk_request_t)(unsafe.Pointer(request)), (*C.lk_multipart_t)(unsafe.Pointer(multipart))))
	}

	// Multipart
	ffi_lk_multipart_new = func() uintptr {
		return uintptr(unsafe.Pointer(C.lk_multipart_new()))
	}

	ffi_lk_multipart_add_text = func(multipart uintptr, namePtr uintptr, nameLen uintptr, valuePtr uintptr, valueLen uintptr) int32 {
		return int32(C.lk_multipart_add_text((*C.lk_multipart_t)(unsafe.Pointer(multipart)), (*C.char)(unsafe.Pointer(namePtr)), C.size_t(nameLen), (*C.char)(unsafe.Pointer(valuePtr)), C.size_t(valueLen)))
	}

	ffi_lk_multipart_add_file = func(multipart uintptr, namePtr uintptr, nameLen uintptr, filenamePtr uintptr, filenameLen uintptr, contentTypePtr uintptr, contentTypeLen uintptr, dataPtr uintptr, dataLen uintptr) int32 {
		return int32(C.lk_multipart_add_file((*C.lk_multipart_t)(unsafe.Pointer(multipart)), (*C.char)(unsafe.Pointer(namePtr)), C.size_t(nameLen), (*C.char)(unsafe.Pointer(filenamePtr)), C.size_t(filenameLen), (*C.char)(unsafe.Pointer(contentTypePtr)), C.size_t(contentTypeLen), (*C.uint8_t)(unsafe.Pointer(dataPtr)), C.size_t(dataLen)))
	}

	ffi_lk_multipart_free = func(multipart uintptr) {
		C.lk_multipart_free((*C.lk_multipart_t)(unsafe.Pointer(multipart)))
	}

	// Response - new methods
	ffi_lk_response_text = func(response uintptr, outPtr uintptr, outLen uintptr, outErr uintptr) int32 {
		return int32(C.lk_response_text((*C.lk_response_t)(unsafe.Pointer(response)), (**C.char)(unsafe.Pointer(outPtr)), (*C.size_t)(unsafe.Pointer(outLen)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_response_error_for_status = func(response uintptr, outErr uintptr) int32 {
		return int32(C.lk_response_error_for_status((*C.lk_response_t)(unsafe.Pointer(response)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_response_cookie_count = func(response uintptr) uintptr {
		return uintptr(C.lk_response_cookie_count((*C.lk_response_t)(unsafe.Pointer(response))))
	}

	ffi_lk_response_cookie_at = func(response uintptr, index uintptr, outNamePtr uintptr, outNameLen uintptr, outValuePtr uintptr, outValueLen uintptr) int32 {
		return int32(C.lk_response_cookie_at((*C.lk_response_t)(unsafe.Pointer(response)), C.size_t(index), (**C.char)(unsafe.Pointer(outNamePtr)), (*C.size_t)(unsafe.Pointer(outNameLen)), (**C.char)(unsafe.Pointer(outValuePtr)), (*C.size_t)(unsafe.Pointer(outValueLen))))
	}

	ffi_lk_response_was_redirected = func(response uintptr) uintptr {
		return boolToUintptr(bool(C.lk_response_was_redirected((*C.lk_response_t)(unsafe.Pointer(response)))))
	}

	ffi_lk_response_redirect_count = func(response uintptr) uintptr {
		return uintptr(C.lk_response_redirect_count((*C.lk_response_t)(unsafe.Pointer(response))))
	}

	ffi_lk_response_redirect_at = func(response uintptr, index uintptr, outURLPtr uintptr, outURLLen uintptr, outStatus uintptr) int32 {
		return int32(C.lk_response_redirect_at((*C.lk_response_t)(unsafe.Pointer(response)), C.size_t(index), (**C.char)(unsafe.Pointer(outURLPtr)), (*C.size_t)(unsafe.Pointer(outURLLen)), (*C.uint16_t)(unsafe.Pointer(outStatus))))
	}

	// Streaming - new methods
	ffi_lk_streaming_response_get_diagnostics_json = func(stream uintptr, outPtr uintptr) int32 {
		return int32(C.lk_streaming_response_get_diagnostics_json((*C.lk_streaming_response_t)(unsafe.Pointer(stream)), (**C.char)(unsafe.Pointer(outPtr))))
	}

	ffi_lk_streaming_response_get_header_by_name = func(stream uintptr, namePtr uintptr, nameLen uintptr, outPtr uintptr, outLen uintptr, outErr uintptr) int32 {
		return int32(C.lk_streaming_response_get_header_by_name((*C.lk_streaming_response_t)(unsafe.Pointer(stream)), (*C.char)(unsafe.Pointer(namePtr)), C.size_t(nameLen), (**C.uint8_t)(unsafe.Pointer(outPtr)), (*C.size_t)(unsafe.Pointer(outLen)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	// Op - new methods
	ffi_lk_op_take_proxy_guard = func(op uintptr, outGuard uintptr, outErr uintptr) int32 {
		return int32(C.lk_op_take_proxy_guard((*C.lk_op_t)(unsafe.Pointer(op)), (**C.lk_proxy_guard_t)(unsafe.Pointer(outGuard)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_op_take_session_pool_guard = func(op uintptr, outGuard uintptr, outErr uintptr) int32 {
		return int32(C.lk_op_take_session_pool_guard((*C.lk_op_t)(unsafe.Pointer(op)), (**C.lk_session_pool_guard_t)(unsafe.Pointer(outGuard)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	// Logging callback
	ffi_lk_log_init_callback = func(callback uintptr, context uintptr, minLevel int32) int32 {
		return int32(C.lk_log_init_callback((C.lk_log_callback_t)(unsafe.Pointer(callback)), unsafe.Pointer(context), C.int32_t(minLevel)))
	}

	// ProxyPool
	ffi_lk_proxy_pool_builder_new = func() uintptr {
		return uintptr(unsafe.Pointer(C.lk_proxy_pool_builder_new()))
	}

	ffi_lk_proxy_pool_builder_add_proxy = func(builder uintptr, urlPtr uintptr, urlLen uintptr) int32 {
		return int32(C.lk_proxy_pool_builder_add_proxy((*C.lk_proxy_pool_builder_t)(unsafe.Pointer(builder)), (*C.char)(unsafe.Pointer(urlPtr)), C.size_t(urlLen)))
	}

	ffi_lk_proxy_pool_builder_add_proxies = func(builder uintptr, urlPtrs uintptr, urlLens uintptr, count uintptr) int32 {
		return int32(C.lk_proxy_pool_builder_add_proxies((*C.lk_proxy_pool_builder_t)(unsafe.Pointer(builder)), (**C.char)(unsafe.Pointer(urlPtrs)), (*C.size_t)(unsafe.Pointer(urlLens)), C.size_t(count)))
	}

	ffi_lk_proxy_pool_builder_set_rotation = func(builder uintptr, strategy int32) int32 {
		return int32(C.lk_proxy_pool_builder_set_rotation((*C.lk_proxy_pool_builder_t)(unsafe.Pointer(builder)), C.lk_rotation_strategy_t(strategy)))
	}

	ffi_lk_proxy_pool_builder_set_proxy_buffer = func(builder uintptr, capacity uintptr) int32 {
		return int32(C.lk_proxy_pool_builder_set_proxy_buffer((*C.lk_proxy_pool_builder_t)(unsafe.Pointer(builder)), C.size_t(capacity)))
	}

	ffi_lk_proxy_pool_builder_set_health_check = func(builder uintptr, hostPtr uintptr, hostLen uintptr, port uint16, intervalMs uint64, timeoutMs uint64) int32 {
		return int32(C.lk_proxy_pool_builder_set_health_check((*C.lk_proxy_pool_builder_t)(unsafe.Pointer(builder)), (*C.char)(unsafe.Pointer(hostPtr)), C.size_t(hostLen), C.uint16_t(port), C.uint64_t(intervalMs), C.uint64_t(timeoutMs)))
	}

	ffi_lk_proxy_pool_builder_set_bad_proxy_config = func(builder uintptr, failureThreshold uint32, windowMs uint64, cooldownMs uint64, maxCooldowns uint32) int32 {
		return int32(C.lk_proxy_pool_builder_set_bad_proxy_config((*C.lk_proxy_pool_builder_t)(unsafe.Pointer(builder)), C.uint32_t(failureThreshold), C.uint64_t(windowMs), C.uint64_t(cooldownMs), C.uint32_t(maxCooldowns)))
	}

	ffi_lk_proxy_pool_builder_set_max_proxies = func(builder uintptr, n uintptr) int32 {
		return int32(C.lk_proxy_pool_builder_set_max_proxies((*C.lk_proxy_pool_builder_t)(unsafe.Pointer(builder)), C.size_t(n)))
	}

	ffi_lk_proxy_pool_builder_set_provider = func(builder uintptr, provider ffiProxyProvider) int32 {
		return int32(C.lk_proxy_pool_builder_set_provider((*C.lk_proxy_pool_builder_t)(unsafe.Pointer(builder)), *(*C.lk_proxy_provider_t)(unsafe.Pointer(&provider))))
	}

	ffi_lk_proxy_pool_builder_build = func(builder uintptr, outPool uintptr, outErr uintptr) int32 {
		return int32(C.lk_proxy_pool_builder_build((*C.lk_proxy_pool_builder_t)(unsafe.Pointer(builder)), (**C.lk_proxy_pool_t)(unsafe.Pointer(outPool)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_proxy_pool_builder_free = func(builder uintptr) {
		C.lk_proxy_pool_builder_free((*C.lk_proxy_pool_builder_t)(unsafe.Pointer(builder)))
	}

	ffi_lk_proxy_pool_acquire = func(pool uintptr, outGuard uintptr, outErr uintptr) int32 {
		return int32(C.lk_proxy_pool_acquire((*C.lk_proxy_pool_t)(unsafe.Pointer(pool)), (**C.lk_proxy_guard_t)(unsafe.Pointer(outGuard)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_proxy_pool_acquire_async = func(pool uintptr, outOp uintptr, outErr uintptr) int32 {
		return int32(C.lk_proxy_pool_acquire_async((*C.lk_proxy_pool_t)(unsafe.Pointer(pool)), (**C.lk_op_t)(unsafe.Pointer(outOp)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_proxy_pool_acquire_fresh = func(pool uintptr, badGuard uintptr, outGuard uintptr, outErr uintptr) int32 {
		return int32(C.lk_proxy_pool_acquire_fresh((*C.lk_proxy_pool_t)(unsafe.Pointer(pool)), (*C.lk_proxy_guard_t)(unsafe.Pointer(badGuard)), (**C.lk_proxy_guard_t)(unsafe.Pointer(outGuard)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_proxy_pool_mark_bad = func(pool uintptr, identityPtr uintptr, identityLen uintptr) int32 {
		return int32(C.lk_proxy_pool_mark_bad((*C.lk_proxy_pool_t)(unsafe.Pointer(pool)), (*C.char)(unsafe.Pointer(identityPtr)), C.size_t(identityLen)))
	}

	ffi_lk_proxy_pool_max_concurrent = func(pool uintptr) uintptr {
		return uintptr(C.lk_proxy_pool_max_concurrent((*C.lk_proxy_pool_t)(unsafe.Pointer(pool))))
	}

	ffi_lk_proxy_pool_free = func(pool uintptr) {
		C.lk_proxy_pool_free((*C.lk_proxy_pool_t)(unsafe.Pointer(pool)))
	}

	ffi_lk_proxy_guard_url = func(guard uintptr, outPtr uintptr, outLen uintptr) int32 {
		return int32(C.lk_proxy_guard_url((*C.lk_proxy_guard_t)(unsafe.Pointer(guard)), (**C.char)(unsafe.Pointer(outPtr)), (*C.size_t)(unsafe.Pointer(outLen))))
	}

	ffi_lk_proxy_guard_mark_bad = func(guard uintptr) int32 {
		return int32(C.lk_proxy_guard_mark_bad((*C.lk_proxy_guard_t)(unsafe.Pointer(guard))))
	}

	ffi_lk_proxy_guard_free = func(guard uintptr) {
		C.lk_proxy_guard_free((*C.lk_proxy_guard_t)(unsafe.Pointer(guard)))
	}

	// SessionPool
	ffi_lk_session_pool_builder_new = func(client uintptr) uintptr {
		return uintptr(unsafe.Pointer(C.lk_session_pool_builder_new((*C.lk_client_t)(unsafe.Pointer(client)))))
	}

	ffi_lk_session_pool_builder_add_proxy = func(builder uintptr, urlPtr uintptr, urlLen uintptr) int32 {
		return int32(C.lk_session_pool_builder_add_proxy((*C.lk_session_pool_builder_t)(unsafe.Pointer(builder)), (*C.char)(unsafe.Pointer(urlPtr)), C.size_t(urlLen)))
	}

	ffi_lk_session_pool_builder_add_proxies = func(builder uintptr, urlPtrs uintptr, urlLens uintptr, count uintptr) int32 {
		return int32(C.lk_session_pool_builder_add_proxies((*C.lk_session_pool_builder_t)(unsafe.Pointer(builder)), (**C.char)(unsafe.Pointer(urlPtrs)), (*C.size_t)(unsafe.Pointer(urlLens)), C.size_t(count)))
	}

	ffi_lk_session_pool_builder_set_rotation = func(builder uintptr, strategy int32) int32 {
		return int32(C.lk_session_pool_builder_set_rotation((*C.lk_session_pool_builder_t)(unsafe.Pointer(builder)), C.lk_rotation_strategy_t(strategy)))
	}

	ffi_lk_session_pool_builder_set_proxy_buffer = func(builder uintptr, capacity uintptr) int32 {
		return int32(C.lk_session_pool_builder_set_proxy_buffer((*C.lk_session_pool_builder_t)(unsafe.Pointer(builder)), C.size_t(capacity)))
	}

	ffi_lk_session_pool_builder_set_max_sessions = func(builder uintptr, n uintptr) int32 {
		return int32(C.lk_session_pool_builder_set_max_sessions((*C.lk_session_pool_builder_t)(unsafe.Pointer(builder)), C.size_t(n)))
	}

	ffi_lk_session_pool_builder_set_idle_timeout = func(builder uintptr, timeoutMs uint64) int32 {
		return int32(C.lk_session_pool_builder_set_idle_timeout((*C.lk_session_pool_builder_t)(unsafe.Pointer(builder)), C.uint64_t(timeoutMs)))
	}

	ffi_lk_session_pool_builder_set_health_check = func(builder uintptr, hostPtr uintptr, hostLen uintptr, port uint16, intervalMs uint64, timeoutMs uint64) int32 {
		return int32(C.lk_session_pool_builder_set_health_check((*C.lk_session_pool_builder_t)(unsafe.Pointer(builder)), (*C.char)(unsafe.Pointer(hostPtr)), C.size_t(hostLen), C.uint16_t(port), C.uint64_t(intervalMs), C.uint64_t(timeoutMs)))
	}

	ffi_lk_session_pool_builder_set_bad_proxy_config = func(builder uintptr, failureThreshold uint32, windowMs uint64, cooldownMs uint64, maxCooldowns uint32) int32 {
		return int32(C.lk_session_pool_builder_set_bad_proxy_config((*C.lk_session_pool_builder_t)(unsafe.Pointer(builder)), C.uint32_t(failureThreshold), C.uint64_t(windowMs), C.uint64_t(cooldownMs), C.uint32_t(maxCooldowns)))
	}

	ffi_lk_session_pool_builder_set_provider = func(builder uintptr, provider ffiProxyProvider) int32 {
		return int32(C.lk_session_pool_builder_set_provider((*C.lk_session_pool_builder_t)(unsafe.Pointer(builder)), *(*C.lk_proxy_provider_t)(unsafe.Pointer(&provider))))
	}

	ffi_lk_session_pool_builder_build = func(builder uintptr, outPool uintptr, outErr uintptr) int32 {
		return int32(C.lk_session_pool_builder_build((*C.lk_session_pool_builder_t)(unsafe.Pointer(builder)), (**C.lk_session_pool_t)(unsafe.Pointer(outPool)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_session_pool_builder_free = func(builder uintptr) {
		C.lk_session_pool_builder_free((*C.lk_session_pool_builder_t)(unsafe.Pointer(builder)))
	}

	ffi_lk_session_pool_acquire = func(pool uintptr, outGuard uintptr, outErr uintptr) int32 {
		return int32(C.lk_session_pool_acquire((*C.lk_session_pool_t)(unsafe.Pointer(pool)), (**C.lk_session_pool_guard_t)(unsafe.Pointer(outGuard)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_session_pool_acquire_async = func(pool uintptr, outOp uintptr, outErr uintptr) int32 {
		return int32(C.lk_session_pool_acquire_async((*C.lk_session_pool_t)(unsafe.Pointer(pool)), (**C.lk_op_t)(unsafe.Pointer(outOp)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_session_pool_acquire_fresh = func(pool uintptr, badGuard uintptr, outGuard uintptr, outErr uintptr) int32 {
		return int32(C.lk_session_pool_acquire_fresh((*C.lk_session_pool_t)(unsafe.Pointer(pool)), (*C.lk_session_pool_guard_t)(unsafe.Pointer(badGuard)), (**C.lk_session_pool_guard_t)(unsafe.Pointer(outGuard)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_session_pool_mark_bad = func(pool uintptr, guard uintptr) int32 {
		return int32(C.lk_session_pool_mark_bad((*C.lk_session_pool_t)(unsafe.Pointer(pool)), (*C.lk_session_pool_guard_t)(unsafe.Pointer(guard))))
	}

	ffi_lk_session_pool_stats = func(pool uintptr, outIdle uintptr, outMax uintptr) int32 {
		return int32(C.lk_session_pool_stats((*C.lk_session_pool_t)(unsafe.Pointer(pool)), (*C.size_t)(unsafe.Pointer(outIdle)), (*C.size_t)(unsafe.Pointer(outMax))))
	}

	ffi_lk_session_pool_free = func(pool uintptr) {
		C.lk_session_pool_free((*C.lk_session_pool_t)(unsafe.Pointer(pool)))
	}

	ffi_lk_session_pool_guard_request_new = func(guard uintptr, methodPtr uintptr, methodLen uintptr, urlPtr uintptr, urlLen uintptr, outRequest uintptr, outErr uintptr) int32 {
		return int32(C.lk_session_pool_guard_request_new((*C.lk_session_pool_guard_t)(unsafe.Pointer(guard)), (*C.char)(unsafe.Pointer(methodPtr)), C.size_t(methodLen), (*C.char)(unsafe.Pointer(urlPtr)), C.size_t(urlLen), (**C.lk_request_t)(unsafe.Pointer(outRequest)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_session_pool_guard_free = func(guard uintptr) {
		C.lk_session_pool_guard_free((*C.lk_session_pool_guard_t)(unsafe.Pointer(guard)))
	}

	// QUIC / HTTP3 client & session config (upstream feat/quic-h3)
	ffi_lk_client_builder_disable_http3 = func(builder uintptr) int32 {
		return int32(C.lk_client_builder_disable_http3((*C.lk_client_builder_t)(unsafe.Pointer(builder))))
	}

	ffi_lk_client_builder_set_timeout_quic_connect = func(builder uintptr, timeoutMs uint64) int32 {
		return int32(C.lk_client_builder_set_timeout_quic_connect((*C.lk_client_builder_t)(unsafe.Pointer(builder)), C.uint64_t(timeoutMs)))
	}

	ffi_lk_client_builder_set_quic_profile_json = func(builder uintptr, jsonPtr uintptr, jsonLen uintptr) int32 {
		return int32(C.lk_client_builder_set_quic_profile_json((*C.lk_client_builder_t)(unsafe.Pointer(builder)), (*C.char)(unsafe.Pointer(jsonPtr)), C.size_t(jsonLen)))
	}

	ffi_lk_client_builder_set_session_resumption_json = func(builder uintptr, jsonPtr uintptr, jsonLen uintptr) int32 {
		return int32(C.lk_client_builder_set_session_resumption_json((*C.lk_client_builder_t)(unsafe.Pointer(builder)), (*C.char)(unsafe.Pointer(jsonPtr)), C.size_t(jsonLen)))
	}

	ffi_lk_client_builder_set_randomize = func(builder uintptr, mode int32, layers uint32) int32 {
		return int32(C.lk_client_builder_set_randomize((*C.lk_client_builder_t)(unsafe.Pointer(builder)), C.lk_randomize_mode_t(mode), C.uint32_t(layers)))
	}

	ffi_lk_client_builder_set_dns_resolver = func(builder uintptr, resolver uintptr) int32 {
		return int32(C.lk_client_builder_set_dns_resolver((*C.lk_client_builder_t)(unsafe.Pointer(builder)), *(*C.lk_dns_resolver_t)(unsafe.Pointer(resolver))))
	}

	ffi_lk_session_builder_set_http3_with_fallback = func(builder uintptr) int32 {
		return int32(C.lk_session_builder_set_http3_with_fallback((*C.lk_session_builder_t)(unsafe.Pointer(builder))))
	}

	// Preferred / negotiated HTTP version
	ffi_lk_request_set_preferred_http_version = func(request uintptr, version int32) int32 {
		return int32(C.lk_request_set_preferred_http_version((*C.lk_request_t)(unsafe.Pointer(request)), C.lk_preferred_http_version_t(version)))
	}

	ffi_lk_response_negotiated_version = func(response uintptr, outVersion uintptr) int32 {
		return int32(C.lk_response_negotiated_version((*C.lk_response_t)(unsafe.Pointer(response)), (*C.lk_negotiated_http_version_t)(unsafe.Pointer(outVersion))))
	}

	// Response split cookie / redirect accessors
	ffi_lk_response_cookie_name_at = func(response uintptr, index uintptr, outPtr uintptr, outLen uintptr) int32 {
		return int32(C.lk_response_cookie_name_at((*C.lk_response_t)(unsafe.Pointer(response)), C.size_t(index), (**C.char)(unsafe.Pointer(outPtr)), (*C.size_t)(unsafe.Pointer(outLen))))
	}

	ffi_lk_response_cookie_value_at = func(response uintptr, index uintptr, outPtr uintptr, outLen uintptr) int32 {
		return int32(C.lk_response_cookie_value_at((*C.lk_response_t)(unsafe.Pointer(response)), C.size_t(index), (**C.char)(unsafe.Pointer(outPtr)), (*C.size_t)(unsafe.Pointer(outLen))))
	}

	ffi_lk_response_redirect_url_at = func(response uintptr, index uintptr, outPtr uintptr, outLen uintptr) int32 {
		return int32(C.lk_response_redirect_url_at((*C.lk_response_t)(unsafe.Pointer(response)), C.size_t(index), (**C.char)(unsafe.Pointer(outPtr)), (*C.size_t)(unsafe.Pointer(outLen))))
	}

	ffi_lk_response_redirect_status_at = func(response uintptr, index uintptr) uint16 {
		return uint16(C.lk_response_redirect_status_at((*C.lk_response_t)(unsafe.Pointer(response)), C.size_t(index)))
	}

	// SOCKS5 UDP probe
	ffi_lk_socks5_udp_probe = func(client uintptr, proxyPtr uintptr, proxyLen uintptr, config uintptr, outReport uintptr, outErr uintptr) int32 {
		return int32(C.lk_socks5_udp_probe(
			(*C.lk_client_t)(unsafe.Pointer(client)),
			(*C.char)(unsafe.Pointer(proxyPtr)),
			C.size_t(proxyLen),
			(*C.lk_socks5_udp_probe_config_t)(unsafe.Pointer(config)),
			(**C.lk_socks5_udp_probe_report_t)(unsafe.Pointer(outReport)),
			(**C.lk_error_t)(unsafe.Pointer(outErr)),
		))
	}

	ffi_lk_socks5_udp_probe_async = func(client uintptr, proxyPtr uintptr, proxyLen uintptr, config uintptr, outOp uintptr, outErr uintptr) int32 {
		return int32(C.lk_socks5_udp_probe_async(
			(*C.lk_client_t)(unsafe.Pointer(client)),
			(*C.char)(unsafe.Pointer(proxyPtr)),
			C.size_t(proxyLen),
			(*C.lk_socks5_udp_probe_config_t)(unsafe.Pointer(config)),
			(**C.lk_op_t)(unsafe.Pointer(outOp)),
			(**C.lk_error_t)(unsafe.Pointer(outErr)),
		))
	}

	ffi_lk_op_take_socks5_udp_probe_report = func(op uintptr, outReport uintptr, outErr uintptr) int32 {
		return int32(C.lk_op_take_socks5_udp_probe_report((*C.lk_op_t)(unsafe.Pointer(op)), (**C.lk_socks5_udp_probe_report_t)(unsafe.Pointer(outReport)), (**C.lk_error_t)(unsafe.Pointer(outErr))))
	}

	ffi_lk_socks5_udp_probe_report_free = func(report uintptr) {
		C.lk_socks5_udp_probe_report_free((*C.lk_socks5_udp_probe_report_t)(unsafe.Pointer(report)))
	}

	ffi_lk_socks5_udp_probe_report_json = func(report uintptr, outPtr uintptr, outLen uintptr) int32 {
		return int32(C.lk_socks5_udp_probe_report_json((*C.lk_socks5_udp_probe_report_t)(unsafe.Pointer(report)), (**C.char)(unsafe.Pointer(outPtr)), (*C.size_t)(unsafe.Pointer(outLen))))
	}

	ffi_lk_socks5_udp_probe_report_error = func(report uintptr, outPtr uintptr, outLen uintptr) int32 {
		return int32(C.lk_socks5_udp_probe_report_error((*C.lk_socks5_udp_probe_report_t)(unsafe.Pointer(report)), (**C.char)(unsafe.Pointer(outPtr)), (*C.size_t)(unsafe.Pointer(outLen))))
	}

	ffi_lk_socks5_udp_probe_report_proxy = func(report uintptr, outPtr uintptr, outLen uintptr) int32 {
		return int32(C.lk_socks5_udp_probe_report_proxy((*C.lk_socks5_udp_probe_report_t)(unsafe.Pointer(report)), (**C.char)(unsafe.Pointer(outPtr)), (*C.size_t)(unsafe.Pointer(outLen))))
	}

	ffi_lk_socks5_udp_probe_report_relay_addr = func(report uintptr, outPtr uintptr, outLen uintptr) int32 {
		return int32(C.lk_socks5_udp_probe_report_relay_addr((*C.lk_socks5_udp_probe_report_t)(unsafe.Pointer(report)), (**C.char)(unsafe.Pointer(outPtr)), (*C.size_t)(unsafe.Pointer(outLen))))
	}

	ffi_lk_socks5_udp_probe_report_elapsed_ms = func(report uintptr) uint64 {
		return uint64(C.lk_socks5_udp_probe_report_elapsed_ms((*C.lk_socks5_udp_probe_report_t)(unsafe.Pointer(report))))
	}

	ffi_lk_socks5_udp_probe_report_phase = func(report uintptr) int32 {
		return int32(C.lk_socks5_udp_probe_report_phase((*C.lk_socks5_udp_probe_report_t)(unsafe.Pointer(report))))
	}

	ffi_lk_socks5_udp_probe_report_support = func(report uintptr) int32 {
		return int32(C.lk_socks5_udp_probe_report_support((*C.lk_socks5_udp_probe_report_t)(unsafe.Pointer(report))))
	}

	// Proxy configuration handles
	ffi_lk_proxy_config_parse = func(urlPtr uintptr, urlLen uintptr, outConfig uintptr) int32 {
		return int32(C.lk_proxy_config_parse((*C.char)(unsafe.Pointer(urlPtr)), C.size_t(urlLen), (**C.lk_proxy_config_t)(unsafe.Pointer(outConfig))))
	}

	ffi_lk_proxy_config_set_user_pass = func(config uintptr, usernamePtr uintptr, usernameLen uintptr, passwordPtr uintptr, passwordLen uintptr) int32 {
		return int32(C.lk_proxy_config_set_user_pass(
			(*C.lk_proxy_config_t)(unsafe.Pointer(config)),
			(*C.char)(unsafe.Pointer(usernamePtr)),
			C.size_t(usernameLen),
			(*C.char)(unsafe.Pointer(passwordPtr)),
			C.size_t(passwordLen),
		))
	}

	ffi_lk_proxy_config_set_http_auth = func(config uintptr, schemePtr uintptr, schemeLen uintptr, credentialsPtr uintptr, credentialsLen uintptr) int32 {
		return int32(C.lk_proxy_config_set_http_auth(
			(*C.lk_proxy_config_t)(unsafe.Pointer(config)),
			(*C.char)(unsafe.Pointer(schemePtr)),
			C.size_t(schemeLen),
			(*C.char)(unsafe.Pointer(credentialsPtr)),
			C.size_t(credentialsLen),
		))
	}

	ffi_lk_proxy_config_set_auth_header = func(config uintptr, header int32) int32 {
		return int32(C.lk_proxy_config_set_auth_header((*C.lk_proxy_config_t)(unsafe.Pointer(config)), C.lk_proxy_auth_header_t(header)))
	}

	ffi_lk_proxy_config_url = func(config uintptr, outPtr uintptr, outLen uintptr) int32 {
		return int32(C.lk_proxy_config_url((*C.lk_proxy_config_t)(unsafe.Pointer(config)), (**C.char)(unsafe.Pointer(outPtr)), (*C.size_t)(unsafe.Pointer(outLen))))
	}

	ffi_lk_proxy_config_identity = func(config uintptr, outPtr uintptr, outLen uintptr) int32 {
		return int32(C.lk_proxy_config_identity((*C.lk_proxy_config_t)(unsafe.Pointer(config)), (**C.char)(unsafe.Pointer(outPtr)), (*C.size_t)(unsafe.Pointer(outLen))))
	}

	ffi_lk_proxy_config_free = func(config uintptr) {
		C.lk_proxy_config_free((*C.lk_proxy_config_t)(unsafe.Pointer(config)))
	}

	ffi_lk_session_builder_set_proxy_config = func(builder uintptr, config uintptr) int32 {
		return int32(C.lk_session_builder_set_proxy_config((*C.lk_session_builder_t)(unsafe.Pointer(builder)), (*C.lk_proxy_config_t)(unsafe.Pointer(config))))
	}

	ffi_lk_proxy_pool_builder_add_proxy_config = func(builder uintptr, config uintptr) int32 {
		return int32(C.lk_proxy_pool_builder_add_proxy_config((*C.lk_proxy_pool_builder_t)(unsafe.Pointer(builder)), (*C.lk_proxy_config_t)(unsafe.Pointer(config))))
	}

	// MASQUE outer-connection configuration
	ffi_lk_masque_config_new = func(outConfig uintptr) int32 {
		return int32(C.lk_masque_config_new((**C.lk_masque_config_t)(unsafe.Pointer(outConfig))))
	}

	ffi_lk_masque_config_add_ca_cert_der = func(config uintptr, derPtr uintptr, derLen uintptr) int32 {
		return int32(C.lk_masque_config_add_ca_cert_der((*C.lk_masque_config_t)(unsafe.Pointer(config)), (*C.uint8_t)(unsafe.Pointer(derPtr)), C.size_t(derLen)))
	}

	ffi_lk_masque_config_add_ca_certs_pem = func(config uintptr, pemPtr uintptr, pemLen uintptr, outAdded uintptr) int32 {
		return int32(C.lk_masque_config_add_ca_certs_pem(
			(*C.lk_masque_config_t)(unsafe.Pointer(config)),
			(*C.char)(unsafe.Pointer(pemPtr)),
			C.size_t(pemLen),
			(*C.size_t)(unsafe.Pointer(outAdded)),
		))
	}

	ffi_lk_masque_config_set_use_native_certs = func(config uintptr, enabled uintptr) int32 {
		return int32(C.lk_masque_config_set_use_native_certs((*C.lk_masque_config_t)(unsafe.Pointer(config)), cBool(enabled)))
	}

	ffi_lk_masque_config_set_server_name = func(config uintptr, namePtr uintptr, nameLen uintptr) int32 {
		return int32(C.lk_masque_config_set_server_name((*C.lk_masque_config_t)(unsafe.Pointer(config)), (*C.char)(unsafe.Pointer(namePtr)), C.size_t(nameLen)))
	}

	ffi_lk_masque_config_set_tunnel_idle_timeout = func(config uintptr, timeoutMs uint64) int32 {
		return int32(C.lk_masque_config_set_tunnel_idle_timeout((*C.lk_masque_config_t)(unsafe.Pointer(config)), C.uint64_t(timeoutMs)))
	}

	ffi_lk_masque_config_set_max_idle_tunnels = func(config uintptr, max uintptr) int32 {
		return int32(C.lk_masque_config_set_max_idle_tunnels((*C.lk_masque_config_t)(unsafe.Pointer(config)), C.size_t(max)))
	}

	ffi_lk_masque_config_set_verify = func(config uintptr, enabled uintptr) int32 {
		return int32(C.lk_masque_config_set_verify((*C.lk_masque_config_t)(unsafe.Pointer(config)), cBool(enabled)))
	}

	ffi_lk_masque_config_free = func(config uintptr) {
		C.lk_masque_config_free((*C.lk_masque_config_t)(unsafe.Pointer(config)))
	}

	ffi_lk_client_builder_set_masque_config = func(builder uintptr, config uintptr) int32 {
		return int32(C.lk_client_builder_set_masque_config((*C.lk_client_builder_t)(unsafe.Pointer(builder)), (*C.lk_masque_config_t)(unsafe.Pointer(config))))
	}

	ffi_lk_proxy_pool_builder_set_masque_tunnel_probe = func(builder uintptr, config uintptr) int32 {
		return int32(C.lk_proxy_pool_builder_set_masque_tunnel_probe((*C.lk_proxy_pool_builder_t)(unsafe.Pointer(builder)), (*C.lk_masque_config_t)(unsafe.Pointer(config))))
	}

	ffi_lk_session_pool_builder_set_masque_tunnel_probe = func(builder uintptr, config uintptr) int32 {
		return int32(C.lk_session_pool_builder_set_masque_tunnel_probe((*C.lk_session_pool_builder_t)(unsafe.Pointer(builder)), (*C.lk_masque_config_t)(unsafe.Pointer(config))))
	}

	// Session cookie readers with attributes
	ffi_lk_session_get_cookies_with_attrs_json = func(session uintptr, urlPtr uintptr, urlLen uintptr, outJSONPtr uintptr, outJSONLen uintptr, outErr uintptr) int32 {
		return int32(C.lk_session_get_cookies_with_attrs_json(
			(*C.lk_session_t)(unsafe.Pointer(session)),
			(*C.char)(unsafe.Pointer(urlPtr)),
			C.size_t(urlLen),
			(**C.char)(unsafe.Pointer(outJSONPtr)),
			(*C.size_t)(unsafe.Pointer(outJSONLen)),
			(**C.lk_error_t)(unsafe.Pointer(outErr)),
		))
	}

	ffi_lk_session_get_all_cookies_json = func(session uintptr, outJSONPtr uintptr, outJSONLen uintptr, outErr uintptr) int32 {
		return int32(C.lk_session_get_all_cookies_json(
			(*C.lk_session_t)(unsafe.Pointer(session)),
			(**C.char)(unsafe.Pointer(outJSONPtr)),
			(*C.size_t)(unsafe.Pointer(outJSONLen)),
			(**C.lk_error_t)(unsafe.Pointer(outErr)),
		))
	}

	// Per-request H2 priority weight
	ffi_lk_request_set_h2_priority_weight = func(request uintptr, weight uint32) int32 {
		return int32(C.lk_request_set_h2_priority_weight((*C.lk_request_t)(unsafe.Pointer(request)), C.uint32_t(weight)))
	}
}
