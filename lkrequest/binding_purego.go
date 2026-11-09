//go:build !lkcgo

package lkrequest

import (
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"

	"github.com/ebitengine/purego"
)

func init() {
	path := extractEmbeddedLib()

	handle, err := openLibrary(path)
	if err != nil {
		panic(fmt.Errorf("lk: failed to load embedded library: %w", err))
	}

	purego.RegisterLibFunc(&ffi_lk_abi_version, handle, "lk_abi_version")
	purego.RegisterLibFunc(&ffi_lk_library_version, handle, "lk_library_version")
	purego.RegisterLibFunc(&ffi_lk_feature_supported, handle, "lk_feature_supported")
	purego.RegisterLibFunc(&ffi_lk_metrics_snapshot, handle, "lk_metrics_snapshot")
	purego.RegisterLibFunc(&ffi_lk_log_init, handle, "lk_log_init")
	purego.RegisterLibFunc(&ffi_lk_preset_list_json, handle, "lk_preset_list_json")
	purego.RegisterLibFunc(&ffi_lk_preset_get_detail_json, handle, "lk_preset_get_detail_json")
	purego.RegisterLibFunc(&ffi_lk_client_builder_add_ca_cert_file, handle, "lk_client_builder_add_ca_cert_file")
	purego.RegisterLibFunc(&ffi_lk_client_builder_add_ca_cert_memory, handle, "lk_client_builder_add_ca_cert_memory")
	purego.RegisterLibFunc(&ffi_lk_client_builder_add_cookie_order, handle, "lk_client_builder_add_cookie_order")
	purego.RegisterLibFunc(&ffi_lk_client_builder_add_default_header, handle, "lk_client_builder_add_default_header")
	purego.RegisterLibFunc(&ffi_lk_client_builder_add_h3_header_order, handle, "lk_client_builder_add_h3_header_order")
	purego.RegisterLibFunc(&ffi_lk_client_builder_add_header_order, handle, "lk_client_builder_add_header_order")
	purego.RegisterLibFunc(&ffi_lk_client_builder_build, handle, "lk_client_builder_build")
	purego.RegisterLibFunc(&ffi_lk_client_builder_free, handle, "lk_client_builder_free")
	purego.RegisterLibFunc(&ffi_lk_client_builder_new, handle, "lk_client_builder_new")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_ech_config, handle, "lk_client_builder_set_ech_config")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_fallback_h2_to_h1, handle, "lk_client_builder_set_fallback_h2_to_h1")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_fallback_proxy_to_direct, handle, "lk_client_builder_set_fallback_proxy_to_direct")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_h2_preset, handle, "lk_client_builder_set_h2_preset")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_keylog_file, handle, "lk_client_builder_set_keylog_file")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_max_connections_per_session, handle, "lk_client_builder_set_max_connections_per_session")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_max_header_count, handle, "lk_client_builder_set_max_header_count")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_max_header_size, handle, "lk_client_builder_set_max_header_size")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_max_headers_total_size, handle, "lk_client_builder_set_max_headers_total_size")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_max_outstanding_ops, handle, "lk_client_builder_set_max_outstanding_ops")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_max_response_body_size, handle, "lk_client_builder_set_max_response_body_size")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_min_transfer_rate, handle, "lk_client_builder_set_min_transfer_rate")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_preset, handle, "lk_client_builder_set_preset")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_retry_on_connection_close, handle, "lk_client_builder_set_retry_on_connection_close")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_tcp_fingerprint_ja4t, handle, "lk_client_builder_set_tcp_fingerprint_ja4t")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_timeout_dns, handle, "lk_client_builder_set_timeout_dns")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_timeout_tcp_connect, handle, "lk_client_builder_set_timeout_tcp_connect")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_timeout_tls_handshake, handle, "lk_client_builder_set_timeout_tls_handshake")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_timeout_total, handle, "lk_client_builder_set_timeout_total")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_timeout_ttfb, handle, "lk_client_builder_set_timeout_ttfb")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_verify, handle, "lk_client_builder_set_verify")
	purego.RegisterLibFunc(&ffi_lk_client_new, handle, "lk_client_new")
	purego.RegisterLibFunc(&ffi_lk_client_new_default, handle, "lk_client_new_default")
	purego.RegisterLibFunc(&ffi_lk_client_clone, handle, "lk_client_clone")
	purego.RegisterLibFunc(&ffi_lk_client_fingerprint_info_json, handle, "lk_client_fingerprint_info_json")
	purego.RegisterLibFunc(&ffi_lk_client_free, handle, "lk_client_free")
	purego.RegisterLibFunc(&ffi_lk_session_builder_build, handle, "lk_session_builder_build")
	purego.RegisterLibFunc(&ffi_lk_session_builder_free, handle, "lk_session_builder_free")
	purego.RegisterLibFunc(&ffi_lk_session_builder_new, handle, "lk_session_builder_new")
	purego.RegisterLibFunc(&ffi_lk_session_builder_set_default_accept_encoding, handle, "lk_session_builder_set_default_accept_encoding")
	purego.RegisterLibFunc(&ffi_lk_session_builder_disable_redirects, handle, "lk_session_builder_disable_redirects")
	purego.RegisterLibFunc(&ffi_lk_session_builder_set_ech_config, handle, "lk_session_builder_set_ech_config")
	purego.RegisterLibFunc(&ffi_lk_session_builder_set_http1_only, handle, "lk_session_builder_set_http1_only")
	purego.RegisterLibFunc(&ffi_lk_session_builder_set_http2_only, handle, "lk_session_builder_set_http2_only")
	purego.RegisterLibFunc(&ffi_lk_session_builder_set_http3_only, handle, "lk_session_builder_set_http3_only")
	purego.RegisterLibFunc(&ffi_lk_session_builder_set_idle_timeout, handle, "lk_session_builder_set_idle_timeout")
	purego.RegisterLibFunc(&ffi_lk_session_builder_set_max_connections, handle, "lk_session_builder_set_max_connections")
	purego.RegisterLibFunc(&ffi_lk_session_builder_set_max_redirects, handle, "lk_session_builder_set_max_redirects")
	purego.RegisterLibFunc(&ffi_lk_session_builder_set_proxy, handle, "lk_session_builder_set_proxy")
	purego.RegisterLibFunc(&ffi_lk_session_builder_set_retry_exponential, handle, "lk_session_builder_set_retry_exponential")
	purego.RegisterLibFunc(&ffi_lk_session_builder_set_retry_fixed, handle, "lk_session_builder_set_retry_fixed")
	purego.RegisterLibFunc(&ffi_lk_session_builder_add_cookie_order, handle, "lk_session_builder_add_cookie_order")
	purego.RegisterLibFunc(&ffi_lk_session_builder_add_h3_header_order, handle, "lk_session_builder_add_h3_header_order")
	purego.RegisterLibFunc(&ffi_lk_session_builder_add_header_order, handle, "lk_session_builder_add_header_order")
	purego.RegisterLibFunc(&ffi_lk_session_new, handle, "lk_session_new")
	purego.RegisterLibFunc(&ffi_lk_session_new_with_config, handle, "lk_session_new_with_config")
	purego.RegisterLibFunc(&ffi_lk_session_clone, handle, "lk_session_clone")
	purego.RegisterLibFunc(&ffi_lk_session_free, handle, "lk_session_free")
	purego.RegisterLibFunc(&ffi_lk_request_add_header, handle, "lk_request_add_header")
	purego.RegisterLibFunc(&ffi_lk_request_add_cookie_order, handle, "lk_request_add_cookie_order")
	purego.RegisterLibFunc(&ffi_lk_request_add_h3_header_order, handle, "lk_request_add_h3_header_order")
	purego.RegisterLibFunc(&ffi_lk_request_add_header_order, handle, "lk_request_add_header_order")
	purego.RegisterLibFunc(&ffi_lk_request_add_query, handle, "lk_request_add_query")
	purego.RegisterLibFunc(&ffi_lk_request_free, handle, "lk_request_free")
	purego.RegisterLibFunc(&ffi_lk_request_new, handle, "lk_request_new")
	purego.RegisterLibFunc(&ffi_lk_request_send, handle, "lk_request_send")
	purego.RegisterLibFunc(&ffi_lk_request_send_async, handle, "lk_request_send_async")
	purego.RegisterLibFunc(&ffi_lk_request_send_streaming, handle, "lk_request_send_streaming")
	purego.RegisterLibFunc(&ffi_lk_request_send_streaming_async, handle, "lk_request_send_streaming_async")
	purego.RegisterLibFunc(&ffi_lk_request_set_accept_encoding, handle, "lk_request_set_accept_encoding")
	purego.RegisterLibFunc(&ffi_lk_request_set_auto_decompress, handle, "lk_request_set_auto_decompress")
	purego.RegisterLibFunc(&ffi_lk_request_set_basic_auth, handle, "lk_request_set_basic_auth")
	purego.RegisterLibFunc(&ffi_lk_request_set_bearer_auth, handle, "lk_request_set_bearer_auth")
	purego.RegisterLibFunc(&ffi_lk_request_set_body_bytes, handle, "lk_request_set_body_bytes")
	purego.RegisterLibFunc(&ffi_lk_request_set_cookie, handle, "lk_request_set_cookie")
	purego.RegisterLibFunc(&ffi_lk_request_set_form, handle, "lk_request_set_form")
	purego.RegisterLibFunc(&ffi_lk_request_set_json_text, handle, "lk_request_set_json_text")
	purego.RegisterLibFunc(&ffi_lk_request_set_proxy, handle, "lk_request_set_proxy")
	purego.RegisterLibFunc(&ffi_lk_request_set_text_body, handle, "lk_request_set_text_body")
	purego.RegisterLibFunc(&ffi_lk_request_set_timeout, handle, "lk_request_set_timeout")
	purego.RegisterLibFunc(&ffi_lk_request_set_version, handle, "lk_request_set_version")
	purego.RegisterLibFunc(&ffi_lk_response_body, handle, "lk_response_body")
	purego.RegisterLibFunc(&ffi_lk_response_content_length, handle, "lk_response_content_length")
	purego.RegisterLibFunc(&ffi_lk_response_copy_body, handle, "lk_response_copy_body")
	purego.RegisterLibFunc(&ffi_lk_response_free, handle, "lk_response_free")
	purego.RegisterLibFunc(&ffi_lk_response_get_diagnostics_json, handle, "lk_response_get_diagnostics_json")
	purego.RegisterLibFunc(&ffi_lk_response_get_header_by_name, handle, "lk_response_get_header_by_name")
	purego.RegisterLibFunc(&ffi_lk_response_header_count, handle, "lk_response_header_count")
	purego.RegisterLibFunc(&ffi_lk_response_header_name_at, handle, "lk_response_header_name_at")
	purego.RegisterLibFunc(&ffi_lk_response_header_value_at, handle, "lk_response_header_value_at")
	purego.RegisterLibFunc(&ffi_lk_response_status, handle, "lk_response_status")
	purego.RegisterLibFunc(&ffi_lk_response_url, handle, "lk_response_url")
	purego.RegisterLibFunc(&ffi_lk_response_version, handle, "lk_response_version")
	purego.RegisterLibFunc(&ffi_lk_streaming_response_free, handle, "lk_streaming_response_free")
	purego.RegisterLibFunc(&ffi_lk_streaming_response_header_count, handle, "lk_streaming_response_header_count")
	purego.RegisterLibFunc(&ffi_lk_streaming_response_header_name_at, handle, "lk_streaming_response_header_name_at")
	purego.RegisterLibFunc(&ffi_lk_streaming_response_header_value_at, handle, "lk_streaming_response_header_value_at")
	purego.RegisterLibFunc(&ffi_lk_streaming_response_status, handle, "lk_streaming_response_status")
	purego.RegisterLibFunc(&ffi_lk_stream_close, handle, "lk_stream_close")
	purego.RegisterLibFunc(&ffi_lk_stream_copy_chunk, handle, "lk_stream_copy_chunk")
	purego.RegisterLibFunc(&ffi_lk_stream_read, handle, "lk_stream_read")
	purego.RegisterLibFunc(&ffi_lk_stream_read_async, handle, "lk_stream_read_async")
	purego.RegisterLibFunc(&ffi_lk_op_cancel, handle, "lk_op_cancel")
	purego.RegisterLibFunc(&ffi_lk_op_free, handle, "lk_op_free")
	purego.RegisterLibFunc(&ffi_lk_op_poll, handle, "lk_op_poll")
	purego.RegisterLibFunc(&ffi_lk_op_take_chunk, handle, "lk_op_take_chunk")
	purego.RegisterLibFunc(&ffi_lk_op_take_error, handle, "lk_op_take_error")
	purego.RegisterLibFunc(&ffi_lk_op_take_response, handle, "lk_op_take_response")
	purego.RegisterLibFunc(&ffi_lk_op_take_streaming_response, handle, "lk_op_take_streaming_response")
	purego.RegisterLibFunc(&ffi_lk_op_wait, handle, "lk_op_wait")
	purego.RegisterLibFunc(&ffi_lk_error_code, handle, "lk_error_code")
	purego.RegisterLibFunc(&ffi_lk_error_free, handle, "lk_error_free")
	purego.RegisterLibFunc(&ffi_lk_error_get_diagnostics_json, handle, "lk_error_get_diagnostics_json")
	purego.RegisterLibFunc(&ffi_lk_error_http_status, handle, "lk_error_http_status")
	purego.RegisterLibFunc(&ffi_lk_error_is_retryable, handle, "lk_error_is_retryable")
	purego.RegisterLibFunc(&ffi_lk_error_message, handle, "lk_error_message")
	purego.RegisterLibFunc(&ffi_lk_error_phase, handle, "lk_error_phase")
	// Client builder - new
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_dns, handle, "lk_client_builder_set_dns")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_dns_custom, handle, "lk_client_builder_set_dns_custom")
	purego.RegisterLibFunc(&ffi_lk_client_builder_set_use_native_certs, handle, "lk_client_builder_set_use_native_certs")
	// Session cookie management
	purego.RegisterLibFunc(&ffi_lk_session_set_cookie, handle, "lk_session_set_cookie")
	purego.RegisterLibFunc(&ffi_lk_session_set_cookie_with_attrs, handle, "lk_session_set_cookie_with_attrs")
	purego.RegisterLibFunc(&ffi_lk_session_remove_cookie, handle, "lk_session_remove_cookie")
	purego.RegisterLibFunc(&ffi_lk_session_get_cookie, handle, "lk_session_get_cookie")
	purego.RegisterLibFunc(&ffi_lk_session_get_cookies_json, handle, "lk_session_get_cookies_json")
	purego.RegisterLibFunc(&ffi_lk_session_clear_cookies, handle, "lk_session_clear_cookies")
	// Session preconnect
	purego.RegisterLibFunc(&ffi_lk_session_preconnect, handle, "lk_session_preconnect")
	purego.RegisterLibFunc(&ffi_lk_session_preconnect_async, handle, "lk_session_preconnect_async")
	// Session connection pool
	purego.RegisterLibFunc(&ffi_lk_session_connection_pool_stats, handle, "lk_session_connection_pool_stats")
	purego.RegisterLibFunc(&ffi_lk_session_connection_pool_clear, handle, "lk_session_connection_pool_clear")
	// Request - new methods
	purego.RegisterLibFunc(&ffi_lk_request_set_cookie_override, handle, "lk_request_set_cookie_override")
	purego.RegisterLibFunc(&ffi_lk_request_set_multipart, handle, "lk_request_set_multipart")
	// Multipart
	purego.RegisterLibFunc(&ffi_lk_multipart_new, handle, "lk_multipart_new")
	purego.RegisterLibFunc(&ffi_lk_multipart_add_text, handle, "lk_multipart_add_text")
	purego.RegisterLibFunc(&ffi_lk_multipart_add_file, handle, "lk_multipart_add_file")
	purego.RegisterLibFunc(&ffi_lk_multipart_free, handle, "lk_multipart_free")
	// Response - new methods
	purego.RegisterLibFunc(&ffi_lk_response_text, handle, "lk_response_text")
	purego.RegisterLibFunc(&ffi_lk_response_error_for_status, handle, "lk_response_error_for_status")
	purego.RegisterLibFunc(&ffi_lk_response_cookie_count, handle, "lk_response_cookie_count")
	purego.RegisterLibFunc(&ffi_lk_response_cookie_at, handle, "lk_response_cookie_at")
	purego.RegisterLibFunc(&ffi_lk_response_was_redirected, handle, "lk_response_was_redirected")
	purego.RegisterLibFunc(&ffi_lk_response_redirect_count, handle, "lk_response_redirect_count")
	purego.RegisterLibFunc(&ffi_lk_response_redirect_at, handle, "lk_response_redirect_at")
	// Streaming - new methods
	purego.RegisterLibFunc(&ffi_lk_streaming_response_get_diagnostics_json, handle, "lk_streaming_response_get_diagnostics_json")
	purego.RegisterLibFunc(&ffi_lk_streaming_response_get_header_by_name, handle, "lk_streaming_response_get_header_by_name")
	// Op - new methods
	purego.RegisterLibFunc(&ffi_lk_op_take_proxy_guard, handle, "lk_op_take_proxy_guard")
	purego.RegisterLibFunc(&ffi_lk_op_take_session_pool_guard, handle, "lk_op_take_session_pool_guard")
	// Logging callback
	purego.RegisterLibFunc(&ffi_lk_log_init_callback, handle, "lk_log_init_callback")
	// ProxyPool
	purego.RegisterLibFunc(&ffi_lk_proxy_pool_builder_new, handle, "lk_proxy_pool_builder_new")
	purego.RegisterLibFunc(&ffi_lk_proxy_pool_builder_add_proxy, handle, "lk_proxy_pool_builder_add_proxy")
	purego.RegisterLibFunc(&ffi_lk_proxy_pool_builder_add_proxies, handle, "lk_proxy_pool_builder_add_proxies")
	purego.RegisterLibFunc(&ffi_lk_proxy_pool_builder_set_rotation, handle, "lk_proxy_pool_builder_set_rotation")
	purego.RegisterLibFunc(&ffi_lk_proxy_pool_builder_set_proxy_buffer, handle, "lk_proxy_pool_builder_set_proxy_buffer")
	purego.RegisterLibFunc(&ffi_lk_proxy_pool_builder_set_health_check, handle, "lk_proxy_pool_builder_set_health_check")
	purego.RegisterLibFunc(&ffi_lk_proxy_pool_builder_set_bad_proxy_config, handle, "lk_proxy_pool_builder_set_bad_proxy_config")
	purego.RegisterLibFunc(&ffi_lk_proxy_pool_builder_set_max_proxies, handle, "lk_proxy_pool_builder_set_max_proxies")
	// set_provider passes structs by value, whose ABI is platform-specific.
	// purego does not support struct arguments on Windows, so registration for
	// both proxy and session providers is split into the platform files.
	registerProviderFuncs(handle)
	purego.RegisterLibFunc(&ffi_lk_proxy_pool_builder_build, handle, "lk_proxy_pool_builder_build")
	purego.RegisterLibFunc(&ffi_lk_proxy_pool_builder_free, handle, "lk_proxy_pool_builder_free")
	purego.RegisterLibFunc(&ffi_lk_proxy_pool_acquire, handle, "lk_proxy_pool_acquire")
	purego.RegisterLibFunc(&ffi_lk_proxy_pool_acquire_async, handle, "lk_proxy_pool_acquire_async")
	purego.RegisterLibFunc(&ffi_lk_proxy_pool_acquire_fresh, handle, "lk_proxy_pool_acquire_fresh")
	purego.RegisterLibFunc(&ffi_lk_proxy_pool_mark_bad, handle, "lk_proxy_pool_mark_bad")
	purego.RegisterLibFunc(&ffi_lk_proxy_pool_max_concurrent, handle, "lk_proxy_pool_max_concurrent")
	purego.RegisterLibFunc(&ffi_lk_proxy_pool_free, handle, "lk_proxy_pool_free")
	purego.RegisterLibFunc(&ffi_lk_proxy_guard_url, handle, "lk_proxy_guard_url")
	purego.RegisterLibFunc(&ffi_lk_proxy_guard_mark_bad, handle, "lk_proxy_guard_mark_bad")
	purego.RegisterLibFunc(&ffi_lk_proxy_guard_free, handle, "lk_proxy_guard_free")
	// SessionPool
	purego.RegisterLibFunc(&ffi_lk_session_pool_builder_new, handle, "lk_session_pool_builder_new")
	purego.RegisterLibFunc(&ffi_lk_session_pool_builder_add_proxy, handle, "lk_session_pool_builder_add_proxy")
	purego.RegisterLibFunc(&ffi_lk_session_pool_builder_add_proxies, handle, "lk_session_pool_builder_add_proxies")
	purego.RegisterLibFunc(&ffi_lk_session_pool_builder_set_rotation, handle, "lk_session_pool_builder_set_rotation")
	purego.RegisterLibFunc(&ffi_lk_session_pool_builder_set_proxy_buffer, handle, "lk_session_pool_builder_set_proxy_buffer")
	purego.RegisterLibFunc(&ffi_lk_session_pool_builder_set_max_sessions, handle, "lk_session_pool_builder_set_max_sessions")
	purego.RegisterLibFunc(&ffi_lk_session_pool_builder_set_idle_timeout, handle, "lk_session_pool_builder_set_idle_timeout")
	purego.RegisterLibFunc(&ffi_lk_session_pool_builder_set_health_check, handle, "lk_session_pool_builder_set_health_check")
	purego.RegisterLibFunc(&ffi_lk_session_pool_builder_set_bad_proxy_config, handle, "lk_session_pool_builder_set_bad_proxy_config")
	purego.RegisterLibFunc(&ffi_lk_session_pool_builder_build, handle, "lk_session_pool_builder_build")
	purego.RegisterLibFunc(&ffi_lk_session_pool_builder_free, handle, "lk_session_pool_builder_free")
	purego.RegisterLibFunc(&ffi_lk_session_pool_acquire, handle, "lk_session_pool_acquire")
	purego.RegisterLibFunc(&ffi_lk_session_pool_acquire_async, handle, "lk_session_pool_acquire_async")
	purego.RegisterLibFunc(&ffi_lk_session_pool_acquire_fresh, handle, "lk_session_pool_acquire_fresh")
	purego.RegisterLibFunc(&ffi_lk_session_pool_mark_bad, handle, "lk_session_pool_mark_bad")
	purego.RegisterLibFunc(&ffi_lk_session_pool_stats, handle, "lk_session_pool_stats")
	purego.RegisterLibFunc(&ffi_lk_session_pool_free, handle, "lk_session_pool_free")
	purego.RegisterLibFunc(&ffi_lk_session_pool_guard_request_new, handle, "lk_session_pool_guard_request_new")
	purego.RegisterLibFunc(&ffi_lk_session_pool_guard_free, handle, "lk_session_pool_guard_free")

	// Symbols added by the upstream feat/quic-h3 work. They are registered
	// optionally so the package still loads against older embedded libraries
	// that predate these exports; callers guard on a nil pointer and return a
	// "not supported by the loaded library" error.
	registerOptional(&ffi_lk_client_builder_disable_http3, handle, "lk_client_builder_disable_http3")
	registerOptional(&ffi_lk_client_builder_set_timeout_quic_connect, handle, "lk_client_builder_set_timeout_quic_connect")
	registerOptional(&ffi_lk_client_builder_set_quic_profile_json, handle, "lk_client_builder_set_quic_profile_json")
	registerOptional(&ffi_lk_client_builder_set_session_resumption_json, handle, "lk_client_builder_set_session_resumption_json")
	registerOptional(&ffi_lk_session_builder_set_http3_with_fallback, handle, "lk_session_builder_set_http3_with_fallback")
	registerOptional(&ffi_lk_client_builder_set_randomize, handle, "lk_client_builder_set_randomize")
	registerOptional(&ffi_lk_request_set_preferred_http_version, handle, "lk_request_set_preferred_http_version")
	registerOptional(&ffi_lk_response_negotiated_version, handle, "lk_response_negotiated_version")
	registerOptional(&ffi_lk_response_cookie_name_at, handle, "lk_response_cookie_name_at")
	registerOptional(&ffi_lk_response_cookie_value_at, handle, "lk_response_cookie_value_at")
	registerOptional(&ffi_lk_response_redirect_url_at, handle, "lk_response_redirect_url_at")
	registerOptional(&ffi_lk_response_redirect_status_at, handle, "lk_response_redirect_status_at")
	registerOptional(&ffi_lk_socks5_udp_probe, handle, "lk_socks5_udp_probe")
	registerOptional(&ffi_lk_socks5_udp_probe_async, handle, "lk_socks5_udp_probe_async")
	registerOptional(&ffi_lk_op_take_socks5_udp_probe_report, handle, "lk_op_take_socks5_udp_probe_report")
	registerOptional(&ffi_lk_socks5_udp_probe_report_free, handle, "lk_socks5_udp_probe_report_free")
	registerOptional(&ffi_lk_socks5_udp_probe_report_json, handle, "lk_socks5_udp_probe_report_json")
	registerOptional(&ffi_lk_socks5_udp_probe_report_error, handle, "lk_socks5_udp_probe_report_error")
	registerOptional(&ffi_lk_socks5_udp_probe_report_proxy, handle, "lk_socks5_udp_probe_report_proxy")
	registerOptional(&ffi_lk_socks5_udp_probe_report_relay_addr, handle, "lk_socks5_udp_probe_report_relay_addr")
	registerOptional(&ffi_lk_socks5_udp_probe_report_elapsed_ms, handle, "lk_socks5_udp_probe_report_elapsed_ms")
	registerOptional(&ffi_lk_socks5_udp_probe_report_phase, handle, "lk_socks5_udp_probe_report_phase")
	registerOptional(&ffi_lk_socks5_udp_probe_report_support, handle, "lk_socks5_udp_probe_report_support")
	// Proxy configuration handles
	registerOptional(&ffi_lk_proxy_config_parse, handle, "lk_proxy_config_parse")
	registerOptional(&ffi_lk_proxy_config_set_user_pass, handle, "lk_proxy_config_set_user_pass")
	registerOptional(&ffi_lk_proxy_config_set_http_auth, handle, "lk_proxy_config_set_http_auth")
	registerOptional(&ffi_lk_proxy_config_set_auth_header, handle, "lk_proxy_config_set_auth_header")
	registerOptional(&ffi_lk_proxy_config_url, handle, "lk_proxy_config_url")
	registerOptional(&ffi_lk_proxy_config_identity, handle, "lk_proxy_config_identity")
	registerOptional(&ffi_lk_proxy_config_free, handle, "lk_proxy_config_free")
	registerOptional(&ffi_lk_session_builder_set_proxy_config, handle, "lk_session_builder_set_proxy_config")
	registerOptional(&ffi_lk_proxy_pool_builder_add_proxy_config, handle, "lk_proxy_pool_builder_add_proxy_config")
	// MASQUE outer-connection configuration
	registerOptional(&ffi_lk_masque_config_new, handle, "lk_masque_config_new")
	registerOptional(&ffi_lk_masque_config_add_ca_cert_der, handle, "lk_masque_config_add_ca_cert_der")
	registerOptional(&ffi_lk_masque_config_add_ca_certs_pem, handle, "lk_masque_config_add_ca_certs_pem")
	registerOptional(&ffi_lk_masque_config_set_use_native_certs, handle, "lk_masque_config_set_use_native_certs")
	registerOptional(&ffi_lk_masque_config_set_server_name, handle, "lk_masque_config_set_server_name")
	registerOptional(&ffi_lk_masque_config_set_tunnel_idle_timeout, handle, "lk_masque_config_set_tunnel_idle_timeout")
	registerOptional(&ffi_lk_masque_config_set_max_idle_tunnels, handle, "lk_masque_config_set_max_idle_tunnels")
	registerOptional(&ffi_lk_masque_config_set_verify, handle, "lk_masque_config_set_verify")
	registerOptional(&ffi_lk_masque_config_free, handle, "lk_masque_config_free")
	registerOptional(&ffi_lk_client_builder_set_masque_config, handle, "lk_client_builder_set_masque_config")
	registerOptional(&ffi_lk_proxy_pool_builder_set_masque_tunnel_probe, handle, "lk_proxy_pool_builder_set_masque_tunnel_probe")
	registerOptional(&ffi_lk_session_pool_builder_set_masque_tunnel_probe, handle, "lk_session_pool_builder_set_masque_tunnel_probe")
	// Session cookie readers with attributes
	registerOptional(&ffi_lk_session_get_cookies_with_attrs_json, handle, "lk_session_get_cookies_with_attrs_json")
	registerOptional(&ffi_lk_session_get_all_cookies_json, handle, "lk_session_get_all_cookies_json")
	// Per-request H2 priority weight
	registerOptional(&ffi_lk_request_set_h2_priority_weight, handle, "lk_request_set_h2_priority_weight")
	// HTTP/1.1 header-name casing
	registerOptional(&ffi_lk_client_builder_set_http1_title_case_headers, handle, "lk_client_builder_set_http1_title_case_headers")
	// Trust Anchor Identifiers (TAI) negotiation
	registerOptional(&ffi_lk_client_builder_set_trust_anchor_policy_json, handle, "lk_client_builder_set_trust_anchor_policy_json")
	registerOptional(&ffi_lk_client_builder_add_trust_anchor_binding_json, handle, "lk_client_builder_add_trust_anchor_binding_json")
	registerOptional(&ffi_lk_trust_anchor_identity_from_der, handle, "lk_trust_anchor_identity_from_der")
}

// registerOptional registers an FFI symbol that may be absent from older
// embedded libraries. A missing symbol leaves the function pointer nil instead
// of panicking, so the package still loads. Callers must guard on the nil
// pointer and surface a clear "not supported by the loaded library" error.
func registerOptional(fptr any, handle uintptr, name string) {
	defer func() { _ = recover() }()
	purego.RegisterLibFunc(fptr, handle, name)
}

func extractEmbeddedLib() string {
	if len(embeddedLib) == 0 {
		panic("lk: embedded library is empty")
	}

	checksum := crc32.ChecksumIEEE(embeddedLib)
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("lkrequest-%08x", checksum))
	path := filepath.Join(dir, embeddedLibName)

	if info, err := os.Stat(path); err == nil && info.Size() == int64(len(embeddedLib)) {
		return path
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		panic(fmt.Errorf("lk: failed to create lib dir: %w", err))
	}

	// Write to a unique temporary file and rename it into place rather than
	// writing the final path directly.
	//
	// Writing directly truncates an existing file, so a second process cold
	// starting at the same time (parallel package tests, multi-replica deploys
	// sharing /tmp) could dlopen a half-written library, or have the file
	// truncated underneath it after mmap, which is a fatal SIGBUS rather than
	// a recoverable error. A rename within the same directory is atomic, so
	// every process observes either no file or the complete one.
	tmp, err := os.CreateTemp(dir, embeddedLibName+".*.tmp")
	if err != nil {
		panic(fmt.Errorf("lk: failed to create temp lib file: %w", err))
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once the rename below succeeded

	if _, err := tmp.Write(embeddedLib); err != nil {
		tmp.Close()
		panic(fmt.Errorf("lk: failed to extract embedded library: %w", err))
	}
	if err := tmp.Close(); err != nil {
		panic(fmt.Errorf("lk: failed to extract embedded library: %w", err))
	}

	if err := os.Rename(tmpPath, path); err != nil {
		// Another process may have installed it first; on Windows renaming
		// onto an existing file fails outright. Accept a complete file, but
		// never fall back to one of unexpected size: a partial write (ENOSPC,
		// for instance) would otherwise be handed to dlopen.
		if info, statErr := os.Stat(path); statErr == nil && info.Size() == int64(len(embeddedLib)) {
			return path
		}
		panic(fmt.Errorf("lk: failed to install embedded library: %w", err))
	}

	return path
}
