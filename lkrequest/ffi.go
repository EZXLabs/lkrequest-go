package lkrequest

import "unsafe"

// Version / capability
var (
	ffi_lk_abi_version       func() uint32
	ffi_lk_library_version   func() unsafe.Pointer
	ffi_lk_feature_supported func(name uintptr) uintptr
	ffi_lk_metrics_snapshot  func(out uintptr) int32
)

// Logging
var ffi_lk_log_init func(levelPtr uintptr, filePathPtr uintptr) int32

// Preset
var (
	ffi_lk_preset_list_json       func(outJSONPtr uintptr) int32
	ffi_lk_preset_get_detail_json func(namePtr uintptr, nameLen uintptr, outJSONPtr uintptr, outErr uintptr) int32
)

// Client builder
var (
	ffi_lk_client_builder_add_ca_cert_file                func(builder uintptr, path uintptr) int32
	ffi_lk_client_builder_add_ca_cert_memory              func(builder uintptr, dataPtr uintptr, dataLen uintptr) int32
	ffi_lk_client_builder_add_cookie_order                func(builder uintptr, namePtr uintptr, nameLen uintptr) int32
	ffi_lk_client_builder_add_default_header              func(builder uintptr, namePtr uintptr, nameLen uintptr, valuePtr uintptr, valueLen uintptr) int32
	ffi_lk_client_builder_add_h3_header_order             func(builder uintptr, namePtr uintptr, nameLen uintptr) int32
	ffi_lk_client_builder_add_header_order                func(builder uintptr, namePtr uintptr, nameLen uintptr) int32
	ffi_lk_client_builder_build                           func(builder uintptr, outClient uintptr, outErr uintptr) int32
	ffi_lk_client_builder_free                            func(builder uintptr)
	ffi_lk_client_builder_new                             func() uintptr
	ffi_lk_client_builder_set_ech_config                  func(builder uintptr, dataPtr uintptr, dataLen uintptr) int32
	ffi_lk_client_builder_set_fallback_h2_to_h1           func(builder uintptr, enabled uintptr) int32
	ffi_lk_client_builder_set_fallback_proxy_to_direct    func(builder uintptr, enabled uintptr) int32
	ffi_lk_client_builder_set_h2_preset                   func(builder uintptr, presetName uintptr) int32
	ffi_lk_client_builder_set_keylog_file                 func(builder uintptr, pathPtr uintptr, pathLen uintptr) int32
	ffi_lk_client_builder_set_max_connections_per_session func(builder uintptr, maxConnections uintptr) int32
	ffi_lk_client_builder_set_max_header_count            func(builder uintptr, count uintptr) int32
	ffi_lk_client_builder_set_max_header_size             func(builder uintptr, size uintptr) int32
	ffi_lk_client_builder_set_max_headers_total_size      func(builder uintptr, size uintptr) int32
	ffi_lk_client_builder_set_max_outstanding_ops         func(builder uintptr, maxOutstandingOps uintptr) int32
	ffi_lk_client_builder_set_max_response_body_size      func(builder uintptr, size uintptr) int32
	ffi_lk_client_builder_set_min_transfer_rate           func(builder uintptr, bytesPerSec uintptr, windowMs uint64) int32
	ffi_lk_client_builder_set_preset                      func(builder uintptr, presetName uintptr) int32
	ffi_lk_client_builder_set_retry_on_connection_close   func(builder uintptr, enabled uintptr) int32
	ffi_lk_client_builder_set_tcp_fingerprint_ja4t        func(builder uintptr, ja4tPtr uintptr, ja4tLen uintptr) int32
	ffi_lk_client_builder_set_timeout_dns                 func(builder uintptr, timeoutMs uint64) int32
	ffi_lk_client_builder_set_timeout_tcp_connect         func(builder uintptr, timeoutMs uint64) int32
	ffi_lk_client_builder_set_timeout_tls_handshake       func(builder uintptr, timeoutMs uint64) int32
	ffi_lk_client_builder_set_timeout_total               func(builder uintptr, timeoutMs uint64) int32
	ffi_lk_client_builder_set_timeout_ttfb                func(builder uintptr, timeoutMs uint64) int32
	ffi_lk_client_builder_set_verify                      func(builder uintptr, enabled uintptr) int32
	ffi_lk_client_builder_set_dns                         func(builder uintptr, dnsConfig int32) int32
	ffi_lk_client_builder_set_dns_custom                  func(builder uintptr, addrPtr uintptr, addrLen uintptr) int32
	ffi_lk_client_builder_set_use_native_certs            func(builder uintptr, enabled uintptr) int32
)

// Client
var (
	ffi_lk_client_new                   func(presetName uintptr, outClient uintptr, outErr uintptr) int32
	ffi_lk_client_new_default           func(outClient uintptr, outErr uintptr) int32
	ffi_lk_client_clone                 func(client uintptr) uintptr
	ffi_lk_client_fingerprint_info_json func(client uintptr, outJSONPtr uintptr, outJSONLen uintptr, outErr uintptr) int32
	ffi_lk_client_free                  func(client uintptr)
)

// Session builder
var (
	ffi_lk_session_builder_build                       func(builder uintptr, outSession uintptr, outErr uintptr) int32
	ffi_lk_session_builder_free                        func(builder uintptr)
	ffi_lk_session_builder_new                         func(client uintptr) uintptr
	ffi_lk_session_builder_set_default_accept_encoding func(builder uintptr, encodingBits uint8) int32
	ffi_lk_session_builder_disable_redirects           func(builder uintptr) int32
	ffi_lk_session_builder_set_ech_config              func(builder uintptr, dataPtr uintptr, dataLen uintptr) int32
	ffi_lk_session_builder_set_http1_only              func(builder uintptr) int32
	ffi_lk_session_builder_set_http2_only              func(builder uintptr) int32
	ffi_lk_session_builder_set_http3_only              func(builder uintptr) int32
	ffi_lk_session_builder_set_idle_timeout            func(builder uintptr, idleTimeoutMs uint64) int32
	ffi_lk_session_builder_set_max_connections         func(builder uintptr, maxConnections uintptr) int32
	ffi_lk_session_builder_set_max_redirects           func(builder uintptr, maxRedirects uint32) int32
	ffi_lk_session_builder_set_proxy                   func(builder uintptr, proxyPtr uintptr, proxyLen uintptr) int32
	ffi_lk_session_builder_set_retry_exponential       func(builder uintptr, maxRetries uint32, baseDelayMs uint64, maxDelayMs uint64, jitter uintptr) int32
	ffi_lk_session_builder_set_retry_fixed             func(builder uintptr, maxRetries uint32, intervalMs uint64) int32
	ffi_lk_session_builder_add_cookie_order            func(builder uintptr, namePtr uintptr, nameLen uintptr) int32
	ffi_lk_session_builder_add_h3_header_order         func(builder uintptr, namePtr uintptr, nameLen uintptr) int32
	ffi_lk_session_builder_add_header_order            func(builder uintptr, namePtr uintptr, nameLen uintptr) int32
)

// Session
var (
	ffi_lk_session_new             func(client uintptr, outSession uintptr, outErr uintptr) int32
	ffi_lk_session_new_with_config func(client uintptr, proxyPtr uintptr, proxyLen uintptr, maxRedirects uint32, outSession uintptr, outErr uintptr) int32
	ffi_lk_session_clone           func(session uintptr) uintptr
	ffi_lk_session_free            func(session uintptr)
)

// Session cookie management
var (
	ffi_lk_session_set_cookie            func(session uintptr, urlPtr uintptr, urlLen uintptr, namePtr uintptr, nameLen uintptr, valuePtr uintptr, valueLen uintptr) int32
	ffi_lk_session_set_cookie_with_attrs func(session uintptr, urlPtr uintptr, urlLen uintptr, namePtr uintptr, nameLen uintptr, valuePtr uintptr, valueLen uintptr, pathPtr uintptr, pathLen uintptr, domainPtr uintptr, domainLen uintptr, secure bool, httpOnly bool, outErr uintptr) int32
	ffi_lk_session_remove_cookie         func(session uintptr, urlPtr uintptr, urlLen uintptr, namePtr uintptr, nameLen uintptr) int32
	ffi_lk_session_get_cookie            func(session uintptr, urlPtr uintptr, urlLen uintptr, namePtr uintptr, nameLen uintptr, outValuePtr uintptr, outValueLen uintptr, outErr uintptr) int32
	ffi_lk_session_get_cookies_json      func(session uintptr, urlPtr uintptr, urlLen uintptr, outJSONPtr uintptr) int32
	ffi_lk_session_clear_cookies         func(session uintptr) int32
)

// Session preconnect
var (
	ffi_lk_session_preconnect       func(session uintptr, urlPtr uintptr, urlLen uintptr, outErr uintptr) int32
	ffi_lk_session_preconnect_async func(session uintptr, urlPtr uintptr, urlLen uintptr, outOp uintptr, outErr uintptr) int32
)

// Session connection pool
var (
	ffi_lk_session_connection_pool_stats func(session uintptr, outH2 uintptr, outH1 uintptr, outTotal uintptr, outMax uintptr, outAtCapacity uintptr) int32
	ffi_lk_session_connection_pool_clear func(session uintptr) int32
)

// Request
var (
	ffi_lk_request_add_header           func(request uintptr, namePtr uintptr, nameLen uintptr, valuePtr uintptr, valueLen uintptr) int32
	ffi_lk_request_add_cookie_order     func(request uintptr, namePtr uintptr, nameLen uintptr) int32
	ffi_lk_request_add_h3_header_order  func(request uintptr, namePtr uintptr, nameLen uintptr) int32
	ffi_lk_request_add_header_order     func(request uintptr, namePtr uintptr, nameLen uintptr) int32
	ffi_lk_request_add_query            func(request uintptr, keyPtr uintptr, keyLen uintptr, valuePtr uintptr, valueLen uintptr) int32
	ffi_lk_request_free                 func(request uintptr)
	ffi_lk_request_new                  func(session uintptr, methodPtr uintptr, methodLen uintptr, urlPtr uintptr, urlLen uintptr, outRequest uintptr, outErr uintptr) int32
	ffi_lk_request_send                 func(request uintptr, outResponse uintptr, outErr uintptr) int32
	ffi_lk_request_send_async           func(request uintptr, outOp uintptr, outErr uintptr) int32
	ffi_lk_request_send_streaming       func(request uintptr, outStream uintptr, outErr uintptr) int32
	ffi_lk_request_send_streaming_async func(request uintptr, outOp uintptr, outErr uintptr) int32
	ffi_lk_request_set_accept_encoding  func(request uintptr, encodingBits uint8) int32
	ffi_lk_request_set_auto_decompress  func(request uintptr, enabled uintptr) int32
	ffi_lk_request_set_basic_auth       func(request uintptr, usernamePtr uintptr, usernameLen uintptr, passwordPtr uintptr, passwordLen uintptr) int32
	ffi_lk_request_set_bearer_auth      func(request uintptr, tokenPtr uintptr, tokenLen uintptr) int32
	ffi_lk_request_set_body_bytes       func(request uintptr, dataPtr uintptr, dataLen uintptr) int32
	ffi_lk_request_set_cookie           func(request uintptr, namePtr uintptr, nameLen uintptr, valuePtr uintptr, valueLen uintptr) int32
	ffi_lk_request_set_form             func(request uintptr, pairsPtr uintptr, pairsCount uintptr) int32
	ffi_lk_request_set_json_text        func(request uintptr, jsonPtr uintptr, jsonLen uintptr) int32
	ffi_lk_request_set_proxy            func(request uintptr, proxyPtr uintptr, proxyLen uintptr) int32
	ffi_lk_request_set_text_body        func(request uintptr, textPtr uintptr, textLen uintptr) int32
	ffi_lk_request_set_timeout          func(request uintptr, timeoutMs uint64) int32
	ffi_lk_request_set_version          func(request uintptr, version int32) int32
)

// Request - new methods
var (
	ffi_lk_request_set_cookie_override func(request uintptr, namePtr uintptr, nameLen uintptr, valuePtr uintptr, valueLen uintptr) int32
	ffi_lk_request_set_multipart       func(request uintptr, multipart uintptr) int32
)

// Multipart
var (
	ffi_lk_multipart_new      func() uintptr
	ffi_lk_multipart_add_text func(multipart uintptr, namePtr uintptr, nameLen uintptr, valuePtr uintptr, valueLen uintptr) int32
	ffi_lk_multipart_add_file func(multipart uintptr, namePtr uintptr, nameLen uintptr, filenamePtr uintptr, filenameLen uintptr, contentTypePtr uintptr, contentTypeLen uintptr, dataPtr uintptr, dataLen uintptr) int32
	ffi_lk_multipart_free     func(multipart uintptr)
)

// Response
var (
	ffi_lk_response_body                 func(response uintptr, outPtr uintptr, outLen uintptr) int32
	ffi_lk_response_content_length       func(response uintptr) int64
	ffi_lk_response_copy_body            func(response uintptr, buf uintptr, bufLen uintptr, outReadLen uintptr) int32
	ffi_lk_response_free                 func(response uintptr)
	ffi_lk_response_get_diagnostics_json func(response uintptr, outPtr uintptr) int32
	ffi_lk_response_get_header_by_name   func(response uintptr, namePtr uintptr, nameLen uintptr, outPtr uintptr, outLen uintptr, outErr uintptr) int32
	ffi_lk_response_header_count         func(response uintptr) uintptr
	ffi_lk_response_header_name_at       func(response uintptr, index uintptr, outPtr uintptr, outLen uintptr) int32
	ffi_lk_response_header_value_at      func(response uintptr, index uintptr, outPtr uintptr, outLen uintptr) int32
	ffi_lk_response_status               func(response uintptr) uint16
	ffi_lk_response_url                  func(response uintptr, outPtr uintptr, outLen uintptr) int32
	ffi_lk_response_version              func(response uintptr, outVersion uintptr) int32
	ffi_lk_response_text                 func(response uintptr, outPtr uintptr, outLen uintptr, outErr uintptr) int32
	ffi_lk_response_error_for_status     func(response uintptr, outErr uintptr) int32
	ffi_lk_response_cookie_count         func(response uintptr) uintptr
	ffi_lk_response_cookie_at            func(response uintptr, index uintptr, outNamePtr uintptr, outNameLen uintptr, outValuePtr uintptr, outValueLen uintptr) int32
	ffi_lk_response_was_redirected       func(response uintptr) uintptr
	ffi_lk_response_redirect_count       func(response uintptr) uintptr
	ffi_lk_response_redirect_at          func(response uintptr, index uintptr, outURLPtr uintptr, outURLLen uintptr, outStatus uintptr) int32
)

// Streaming
var (
	ffi_lk_streaming_response_free                 func(stream uintptr)
	ffi_lk_streaming_response_header_count         func(stream uintptr) uintptr
	ffi_lk_streaming_response_header_name_at       func(stream uintptr, index uintptr, outPtr uintptr, outLen uintptr) int32
	ffi_lk_streaming_response_header_value_at      func(stream uintptr, index uintptr, outPtr uintptr, outLen uintptr) int32
	ffi_lk_streaming_response_status               func(stream uintptr) uint16
	ffi_lk_stream_close                            func(stream uintptr) int32
	ffi_lk_stream_copy_chunk                       func(stream uintptr, buf uintptr, bufLen uintptr, outReadLen uintptr) int32
	ffi_lk_stream_read                             func(stream uintptr, outChunk uintptr, outErr uintptr) int32
	ffi_lk_stream_read_async                       func(stream uintptr, outOp uintptr, outErr uintptr) int32
	ffi_lk_streaming_response_get_diagnostics_json func(stream uintptr, outPtr uintptr) int32
	ffi_lk_streaming_response_get_header_by_name   func(stream uintptr, namePtr uintptr, nameLen uintptr, outPtr uintptr, outLen uintptr, outErr uintptr) int32
)

// Op
var (
	ffi_lk_op_cancel                  func(op uintptr) int32
	ffi_lk_op_free                    func(op uintptr)
	ffi_lk_op_poll                    func(op uintptr) int32
	ffi_lk_op_take_chunk              func(op uintptr, outChunk uintptr, outErr uintptr) int32
	ffi_lk_op_take_error              func(op uintptr, outErr uintptr) int32
	ffi_lk_op_take_response           func(op uintptr, outResponse uintptr, outErr uintptr) int32
	ffi_lk_op_take_streaming_response func(op uintptr, outStream uintptr, outErr uintptr) int32
	ffi_lk_op_wait                    func(op uintptr, timeoutMs uint64) int32
	ffi_lk_op_take_proxy_guard        func(op uintptr, outGuard uintptr, outErr uintptr) int32
	ffi_lk_op_take_session_pool_guard func(op uintptr, outGuard uintptr, outErr uintptr) int32
)

// Error
var (
	ffi_lk_error_code                 func(err uintptr) int32
	ffi_lk_error_free                 func(err uintptr)
	ffi_lk_error_get_diagnostics_json func(err uintptr, outPtr uintptr) int32
	ffi_lk_error_http_status          func(err uintptr) int32
	ffi_lk_error_is_retryable         func(err uintptr) uintptr
	ffi_lk_error_message              func(err uintptr, outPtr uintptr, outLen uintptr) int32
	ffi_lk_error_phase                func(err uintptr) int32
)

// Logging callback
var ffi_lk_log_init_callback func(callback uintptr, context uintptr, minLevel int32) int32

// ProxyPool
var (
	ffi_lk_proxy_pool_builder_new                  func() uintptr
	ffi_lk_proxy_pool_builder_add_proxy            func(builder uintptr, urlPtr uintptr, urlLen uintptr) int32
	ffi_lk_proxy_pool_builder_add_proxies          func(builder uintptr, urlPtrs uintptr, urlLens uintptr, count uintptr) int32
	ffi_lk_proxy_pool_builder_set_rotation         func(builder uintptr, strategy int32) int32
	ffi_lk_proxy_pool_builder_set_proxy_buffer     func(builder uintptr, capacity uintptr) int32
	ffi_lk_proxy_pool_builder_set_health_check     func(builder uintptr, hostPtr uintptr, hostLen uintptr, port uint16, intervalMs uint64, timeoutMs uint64) int32
	ffi_lk_proxy_pool_builder_set_bad_proxy_config func(builder uintptr, failureThreshold uint32, windowMs uint64, cooldownMs uint64, maxCooldowns uint32) int32
	ffi_lk_proxy_pool_builder_set_max_proxies      func(builder uintptr, n uintptr) int32
	ffi_lk_proxy_pool_builder_build                func(builder uintptr, outPool uintptr, outErr uintptr) int32
	ffi_lk_proxy_pool_builder_free                 func(builder uintptr)
	ffi_lk_proxy_pool_acquire                      func(pool uintptr, outGuard uintptr, outErr uintptr) int32
	ffi_lk_proxy_pool_acquire_async                func(pool uintptr, outOp uintptr, outErr uintptr) int32
	ffi_lk_proxy_pool_acquire_fresh                func(pool uintptr, badGuard uintptr, outGuard uintptr, outErr uintptr) int32
	ffi_lk_proxy_pool_mark_bad                     func(pool uintptr, identityPtr uintptr, identityLen uintptr) int32
	ffi_lk_proxy_pool_max_concurrent               func(pool uintptr) uintptr
	ffi_lk_proxy_pool_free                         func(pool uintptr)
	ffi_lk_proxy_guard_url                         func(guard uintptr, outPtr uintptr, outLen uintptr) int32
	ffi_lk_proxy_guard_mark_bad                    func(guard uintptr) int32
	ffi_lk_proxy_guard_free                        func(guard uintptr)
)

// SessionPool
var (
	ffi_lk_session_pool_builder_new                  func(client uintptr) uintptr
	ffi_lk_session_pool_builder_add_proxy            func(builder uintptr, urlPtr uintptr, urlLen uintptr) int32
	ffi_lk_session_pool_builder_add_proxies          func(builder uintptr, urlPtrs uintptr, urlLens uintptr, count uintptr) int32
	ffi_lk_session_pool_builder_set_rotation         func(builder uintptr, strategy int32) int32
	ffi_lk_session_pool_builder_set_proxy_buffer     func(builder uintptr, capacity uintptr) int32
	ffi_lk_session_pool_builder_set_max_sessions     func(builder uintptr, n uintptr) int32
	ffi_lk_session_pool_builder_set_idle_timeout     func(builder uintptr, timeoutMs uint64) int32
	ffi_lk_session_pool_builder_set_health_check     func(builder uintptr, hostPtr uintptr, hostLen uintptr, port uint16, intervalMs uint64, timeoutMs uint64) int32
	ffi_lk_session_pool_builder_set_bad_proxy_config func(builder uintptr, failureThreshold uint32, windowMs uint64, cooldownMs uint64, maxCooldowns uint32) int32
	ffi_lk_session_pool_builder_build                func(builder uintptr, outPool uintptr, outErr uintptr) int32
	ffi_lk_session_pool_builder_free                 func(builder uintptr)
	ffi_lk_session_pool_acquire                      func(pool uintptr, outGuard uintptr, outErr uintptr) int32
	ffi_lk_session_pool_acquire_async                func(pool uintptr, outOp uintptr, outErr uintptr) int32
	ffi_lk_session_pool_acquire_fresh                func(pool uintptr, badGuard uintptr, outGuard uintptr, outErr uintptr) int32
	ffi_lk_session_pool_mark_bad                     func(pool uintptr, guard uintptr) int32
	ffi_lk_session_pool_stats                        func(pool uintptr, outIdle uintptr, outMax uintptr) int32
	ffi_lk_session_pool_free                         func(pool uintptr)
	ffi_lk_session_pool_guard_request_new            func(guard uintptr, methodPtr uintptr, methodLen uintptr, urlPtr uintptr, urlLen uintptr, outRequest uintptr, outErr uintptr) int32
	ffi_lk_session_pool_guard_free                   func(guard uintptr)
)

// QUIC / HTTP3 client & session config (upstream feat/quic-h3)
var (
	ffi_lk_client_builder_disable_http3               func(builder uintptr) int32
	ffi_lk_client_builder_set_timeout_quic_connect    func(builder uintptr, timeoutMs uint64) int32
	ffi_lk_client_builder_set_quic_profile_json       func(builder uintptr, jsonPtr uintptr, jsonLen uintptr) int32
	ffi_lk_client_builder_set_session_resumption_json func(builder uintptr, jsonPtr uintptr, jsonLen uintptr) int32
	ffi_lk_session_builder_set_http3_with_fallback    func(builder uintptr) int32
)

// Fingerprint randomization (upstream feat/ffi-randomize). Synthetic modes
// require a library built with the `synthetic-fp` feature.
var (
	ffi_lk_client_builder_set_randomize func(builder uintptr, mode int32, layers uint32) int32
)

// Preferred / negotiated HTTP version
var (
	ffi_lk_request_set_preferred_http_version func(request uintptr, version int32) int32
	ffi_lk_response_negotiated_version        func(response uintptr, outVersion uintptr) int32
)

// Response split cookie / redirect accessors
var (
	ffi_lk_response_cookie_name_at     func(response uintptr, index uintptr, outPtr uintptr, outLen uintptr) int32
	ffi_lk_response_cookie_value_at    func(response uintptr, index uintptr, outPtr uintptr, outLen uintptr) int32
	ffi_lk_response_redirect_url_at    func(response uintptr, index uintptr, outPtr uintptr, outLen uintptr) int32
	ffi_lk_response_redirect_status_at func(response uintptr, index uintptr) uint16
)

// SOCKS5 UDP probe
var (
	ffi_lk_socks5_udp_probe                   func(client uintptr, proxyPtr uintptr, proxyLen uintptr, config uintptr, outReport uintptr, outErr uintptr) int32
	ffi_lk_socks5_udp_probe_async             func(client uintptr, proxyPtr uintptr, proxyLen uintptr, config uintptr, outOp uintptr, outErr uintptr) int32
	ffi_lk_op_take_socks5_udp_probe_report    func(op uintptr, outReport uintptr, outErr uintptr) int32
	ffi_lk_socks5_udp_probe_report_free       func(report uintptr)
	ffi_lk_socks5_udp_probe_report_json       func(report uintptr, outPtr uintptr, outLen uintptr) int32
	ffi_lk_socks5_udp_probe_report_error      func(report uintptr, outPtr uintptr, outLen uintptr) int32
	ffi_lk_socks5_udp_probe_report_proxy      func(report uintptr, outPtr uintptr, outLen uintptr) int32
	ffi_lk_socks5_udp_probe_report_relay_addr func(report uintptr, outPtr uintptr, outLen uintptr) int32
	ffi_lk_socks5_udp_probe_report_elapsed_ms func(report uintptr) uint64
	ffi_lk_socks5_udp_probe_report_phase      func(report uintptr) int32
	ffi_lk_socks5_udp_probe_report_support    func(report uintptr) int32
)

// Proxy configuration handles: credentials a proxy URL cannot express.
var (
	ffi_lk_proxy_config_parse                  func(urlPtr uintptr, urlLen uintptr, outConfig uintptr) int32
	ffi_lk_proxy_config_set_user_pass          func(config uintptr, usernamePtr uintptr, usernameLen uintptr, passwordPtr uintptr, passwordLen uintptr) int32
	ffi_lk_proxy_config_set_http_auth          func(config uintptr, schemePtr uintptr, schemeLen uintptr, credentialsPtr uintptr, credentialsLen uintptr) int32
	ffi_lk_proxy_config_set_auth_header        func(config uintptr, header int32) int32
	ffi_lk_proxy_config_url                    func(config uintptr, outPtr uintptr, outLen uintptr) int32
	ffi_lk_proxy_config_identity               func(config uintptr, outPtr uintptr, outLen uintptr) int32
	ffi_lk_proxy_config_free                   func(config uintptr)
	ffi_lk_session_builder_set_proxy_config    func(builder uintptr, config uintptr) int32
	ffi_lk_proxy_pool_builder_add_proxy_config func(builder uintptr, config uintptr) int32
)

// MASQUE outer-connection configuration. Every symbol exists in every build,
// but refuses unless the library was built with the `masque` feature.
var (
	ffi_lk_masque_config_new                            func(outConfig uintptr) int32
	ffi_lk_masque_config_add_ca_cert_der                func(config uintptr, derPtr uintptr, derLen uintptr) int32
	ffi_lk_masque_config_add_ca_certs_pem               func(config uintptr, pemPtr uintptr, pemLen uintptr, outAdded uintptr) int32
	ffi_lk_masque_config_set_use_native_certs           func(config uintptr, enabled uintptr) int32
	ffi_lk_masque_config_set_server_name                func(config uintptr, namePtr uintptr, nameLen uintptr) int32
	ffi_lk_masque_config_set_tunnel_idle_timeout        func(config uintptr, timeoutMs uint64) int32
	ffi_lk_masque_config_set_max_idle_tunnels           func(config uintptr, max uintptr) int32
	ffi_lk_masque_config_set_verify                     func(config uintptr, enabled uintptr) int32
	ffi_lk_masque_config_free                           func(config uintptr)
	ffi_lk_client_builder_set_masque_config             func(builder uintptr, config uintptr) int32
	ffi_lk_proxy_pool_builder_set_masque_tunnel_probe   func(builder uintptr, config uintptr) int32
	ffi_lk_session_pool_builder_set_masque_tunnel_probe func(builder uintptr, config uintptr) int32
)

// Session cookie readers that keep each cookie's attributes.
var (
	ffi_lk_session_get_cookies_with_attrs_json func(session uintptr, urlPtr uintptr, urlLen uintptr, outJSONPtr uintptr, outJSONLen uintptr, outErr uintptr) int32
	ffi_lk_session_get_all_cookies_json        func(session uintptr, outJSONPtr uintptr, outJSONLen uintptr, outErr uintptr) int32
)

// Per-request legacy H2 HEADERS priority weight.
var (
	ffi_lk_request_set_h2_priority_weight func(request uintptr, weight uint32) int32
)
