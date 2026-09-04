# Changelog

All important changes to this project will be documented in this file.

## lkrequest-go 0.2.1

Ships the macOS (Apple Silicon) library that previous releases declared but
never included, and updates the bundled `lkrequest-ffi` native engine to 0.2.2,
which adds the Chrome 152 preset and a large batch of protocol-correctness
fixes across TLS, HTTP/2, DNS, redirects, cookies and proxying. There are no Go
API changes and no new native symbols — the FFI surface is unchanged from the
one 0.2.0 shipped — so every existing call shape keeps working. Two changes
alter the bytes on the wire: the `multipart/form-data` boundary format and the
HTTP/2 DATA frame size used with Chrome presets.

### 🚀 Features

- **macOS / Apple Silicon Support** — `darwin/arm64` now ships with an embedded library, so `go get` builds and runs on Apple Silicon without any manual setup. Earlier releases carried the `darwin/arm64` embed declaration without the matching library, which made the package fail to compile on macOS with `pattern lib/darwin_arm64/liblkrequest_ffi.dylib: no matching files found`. The bundled macOS engine is built with `synthetic-fp` and `quic-h3`, like the Linux and Windows libraries, so fingerprint randomization and HTTP/3 are available there too.
- **Chrome 152 Preset** — `SetPreset("chrome_152")` / `NewClient("chrome_152")`, captured from Chrome 152.0.7977.65 (Windows, August 2026). TCP TLS prepends an independently randomized GREASE code point to `signature_algorithms` ahead of the ML-DSA entries, and adds Chromium/BoringSSL's trust-anchor identifiers extension (`0xca34`) with its order reshuffled per ClientHello. HTTP/2 keeps Chrome 151's SETTINGS, connection window, pseudo-header order and priority shape. QUIC keeps the Chrome 151 signature and transport baseline minus the `google_initial_rtt` transport parameter (`0x3127`), which Chrome no longer sends; H3 SETTINGS retain the latest verified Chrome baseline because the capture did not establish a complete H3 exchange.
- **Chrome-Accurate HTTP/2 DATA Framing** — Chrome presets now cap each outbound DATA payload at 16375 bytes, so the 9-byte frame header plus payload land in a single 16 KiB socket write the way Chromium does. Request bodies larger than that are split across more DATA frames than before; this is a fingerprint-fidelity improvement and needs no code change.

### 🐞 Fixes

- **Multipart Boundary No Longer Names the Library** — the boundary was built as `----lkrequest{hex}`, putting the crate name in the `Content-Type` of every upload, where a single server-side regex identified it whatever the TLS and HTTP/2 fingerprints said. It now matches what a real Chrome emits: `----WebKitFormBoundary` followed by exactly 16 alphanumerics, drawn by rejection sampling so the character distribution is uniform. **This changes the bytes on the wire** — tests that pin the boundary or the exact request body need updating.
- **Multipart Field Names and Filenames Are Escaped** — `Multipart.AddText` / `AddFile` interpolated caller-supplied names into `Content-Disposition` verbatim, so a name containing CR or LF could terminate the header line and append part headers, start an extra part, or forge the boundary. `"` now becomes `%22`, CR and LF become `%0D` and `%0A`, and a CR or LF in a part's content type is rejected outright since that header has no quoted form to escape into.
- **Sessions Are No Longer Capped at 64 Connections by Default** — the implicit ceiling made unrelated high-concurrency workloads fail with `session connection limit reached`. The default is now unbounded; `ClientBuilder.SetMaxConnectionsPerSession` and `SessionBuilder.SetMaxConnections` still impose a hard budget when set. This revises the strict-enforcement note from 0.2.0: enforcement is unchanged, only the default ceiling is gone.
- **`304 Not Modified` Is No Longer Treated as a Redirect** — every 3xx was classified as a redirect, so a conditional request answered with `304` (and likewise `300` / `305` / `306`) reached the `Location` lookup and failed with `redirect response missing Location header`. Only `301` / `302` / `303` / `307` / `308` are followed now; the rest are returned to the caller as-is.
- **Redirects Rewrite the Method Only Where the Spec Says To** — `301`, `302` and `303` unconditionally became `GET`, so a `HEAD` request came back as a `GET`, which Chrome does not do. `303` still rewrites everything except `HEAD`.
- **Multi-Layer `Content-Encoding` Is Decoded** — only the outermost coding was handled, so `gzip, br` was returned still compressed under a header claiming otherwise. Codings are applied in reverse order now, on both `Send()` and `SendStreaming()`.
- **`Content-Encoding: deflate` Accepts the zlib Form** — only a bare raw-deflate stream was decodable, so servers sending the zlib-wrapped form the spec actually defines failed outright. Both the buffered and streaming paths sniff the header and pick the matching decoder.
- **Status-Code Retries Are Gated on Replay Safety** — the error-retry path checked replay safety, the status path checked nothing, so a `POST` answered with `502` was sent twice. A non-idempotent method is now replayed only on a status that says it was not applied (`429`, `503`); `502`, `504` and the Cloudflare `520`–`530` range are ambiguous about whether the origin already processed the request and are returned to the caller instead. Safe methods are unaffected.
- **HTTP/2 Graceful Shutdown** — a `GOAWAY` carrying `NO_ERROR` (which Google and Cloudflare send routinely for connection turnover) set no connection error, while the draining check keyed off "has error", so the driver kept opening streams the peer had already promised to refuse. Draining is keyed off GOAWAY receipt itself now, and queued requests fail over to a fresh connection.
- **HTTP/2 Responses Are Validated Rather Than Quietly Repaired** — a `content-length` disagreeing with the DATA actually delivered was accepted (a response-length desync), unparseable headers were dropped, and a missing or malformed `:status` became `500`. Connection-specific fields are now rejected, pseudo-header rules enforced, DATA held to the same standard as HEADERS, and a malformed response resets the stream instead of leaving the peer sending DATA on a stream already torn down.
- **HTTP/2 Connection State Machine** — frames could be interleaved into a CONTINUATION sequence, HEADERS-path stream-state checks were weaker than the DATA path's, and an `INITIAL_WINDOW_SIZE` overflow mapped to the wrong error code. Fixing the DATA path also closed a connection-window leak where a frame on a stream that could not receive consumed window that was never replenished.
- **TLS Post-Handshake Message Reassembly** — each record was treated as exactly one message, so a `KeyUpdate` sharing a record with a `NewSessionTicket` was silently dropped: read keys were never rotated and every subsequent record failed to decrypt. Coalesced tickets past the first were lost the same way. Decrypted bytes are buffered and only complete messages dispatched now.
- **The Handshake Fails on Server Choices the Client Never Offered** — a TLS 1.2 `named_curve`, a `ServerKeyExchange` signature algorithm, or a TLS 1.3 `CertificateVerify` scheme was checked only against what the implementation supports, never against what the ClientHello advertised. Duplicate-extension detection also ran only on the HelloRetryRequest branch, and unsolicited TLS 1.2 extensions had no "did we offer this?" check at all.
- **TLS Record Layer Hardened and Session-Ticket Trust Bounded** — plaintext handshake and alert records were still accepted after handshake keys were installed, an unknown record content type was silently skipped, TLS 1.2 resumption neither recorded nor checked Extended Master Secret state, and ticket lifetimes were unbounded (now capped at seven days).
- **The Peer's `record_size_limit` Is Honoured When Sending** — the extension was advertised but never applied, so a peer that asked for smaller records got 16384-byte ones anyway.
- **A TLS 1.2 `HelloRequest` Is Refused With a Fatal Alert** — it was dropped silently, which no browser does, leaving a behavioural probe for any server willing to send one. It now sends a fatal `no_renegotiation` and aborts, matching BoringSSL. **Behaviour change:** such a connection now ends in an error where it previously carried on.
- **A Connection Ending Without `close_notify` Is Distinguishable** — the read loop now stops at `close_notify` rather than polling on, because anything arriving after it did not come from the peer, and those bytes were being appended to the response body.
- **ECH Rejection Aborts Instead of Returning an Unprotected Connection** — when a server did not accept Encrypted ClientHello, the library logged it and carried on, handing back a connection whose true server name had never been protected, so the request ECH existed to hide went out in the clear. Both the TCP and QUIC paths now abort before any traffic keys exist, and the session redials once with the `retry_configs` the server supplied, as a browser does. Retrying without ECH is deliberately not a fallback. GREASE ECH is unaffected.
- **Alternative Services No Longer Supply the TLS Identity** — an Alt-Svc entry or HTTPS RR names where to send packets, but the endpoint must still prove it may serve the origin. The advertised host was used as the SNI and certificate name, so the alternative authenticated as itself, and conforming deployments failed outright because an HTTPS RR target is typically a CDN's internal name serving the origin's certificate. The dial target still follows the advertisement.
- **Alternative-Service Parsing** — the bracketed IPv6 authority form was stripped at parse time while URL host lookups keep it, so a bare `2001:db8::1` reached the dialer unusable; `persist=1` is now parsed; `h3=":0"` no longer produces an alt-authority on a reserved port; and draft advertisements such as `h3-29` are ignored, matching Chrome 140+.
- **DNS HTTPS/SVCB Records Are Read as Records** — fields from same-priority records were merged together, producing endpoint descriptions no server ever published. `AliasMode` is followed now (with loop detection and a hop cap), and `mandatory` and `no-default-alpn` are honoured instead of silently dropped, so a client no longer connects with a protocol the endpoint declared it does not speak.
- **HTTPS-RR Discovery Issues One Query per Request Instead of Two** — protocol discovery and the handshake's automatic ECH configuration each ran their own lookup. Beyond the extra round trip, the two answers could disagree, leaving the ECH config out of step with the endpoint the HTTP/3 decision was made against.
- **Cookie Name Prefixes Are Enforced** — `__Secure-` and `__Host-` carry security meaning only if their requirements are checked, and a `Secure` cookie received over plain HTTP was stored regardless of its name.
- **HSTS Is Learned Only From a Secure Transport** — a plaintext response could inject a `Strict-Transport-Security` policy into a session, and IP-literal hosts are never pinned now.
- **SOCKS5 and CONNECT Proxy Compatibility** — the username/password sub-negotiation reply's version byte was never validated, and only an exact `200` completed a CONNECT tunnel where every 2xx is a success.
- **QUIC and HTTP/3 Validation** — a QUIC ticket's `max_early_data_size` was accepted whenever non-zero rather than at the one value the spec allows, custom H3 profiles could emit reserved settings identifiers, and the H3 path was missing two of the three ECH configuration fallbacks the TCP path already had.

## lkrequest-go 0.2.0

Updates the bundled `lkrequest-ffi` native engine with upstream stability,
interoperability, and security fixes, and adds a Go-side convenience layer over
the existing FFI surface. The native changes need the matching FFI libraries
for this release; the Go API additions introduce no new native symbols. All
existing call shapes keep working.

### 🚀 Features

- **HTTP Verb Helpers** — `Session` and `SessionPoolGuard` gain `Get` / `Post` / `Put` / `Delete` / `Head` / `Patch` / `Options`, plus a generic `Request(method, url)` for methods without a helper. They return `*Request` rather than `(*Request, error)`, so a construction failure travels with the request and is reported by `Send()` instead of needing its own error check: `session.Get(url).AddHeader(...).Send()`. `NewRequest(session, method, url)` is unchanged for callers that want the error up front.
- **JSON Body Marshalling** — `Request.SetJSON(v any)` encodes a Go value with `encoding/json`, alongside `SetJSONBody` which still takes a pre-encoded document. `Clone()` replays the encoded form rather than re-marshalling the original value.
- **Aggregate Response Views** — `Response.Cookies()` and `Response.Redirects()` return slices, matching how `Headers()` already relates to `HeaderAt()`. The indexed accessors (`CookieAt`, `RedirectAt`, ...) are unchanged.
- **Structured Cookie Jar Access** — `Session.GetCookies(url)` decodes what `GetCookiesJSON(url)` reports raw.
- **New Types** — `Cookie` and `Redirect`. `Cookie` is deliberately not `net/http.Cookie`: the FFI layer reports name and value only, so the remaining attribute fields would read as "not set" rather than "not available".
- **Chrome 151 Preset** — `SetPreset("chrome_151")` / `NewClient("chrome_151")`, captured from Chrome 151.0.7922.72 (Windows, July 2026). TCP TLS/H2 stable fields match Chrome 150; the QUIC layer removes the three ML-DSA signature algorithms while keeping the classic list plus `rsa_pkcs1_sha1`, with QUIC transport parameters and H3 SETTINGS carried over from Chrome 150.
- **Dedicated Chrome 150 QUIC / HTTP/3 Preset** — Chrome 150 now uses dedicated QUIC-TLS and H3 profiles instead of reusing Chrome 146. The preset covers its captured signature algorithms, QUIC transport parameters, QPACK limits, navigation `PRIORITY_UPDATE`, connection ID lengths, and Initial datagram sizing.

### 🐞 Fixes

- **HTTP/2 Interim Responses** — The native HTTP/2 driver now consumes `1xx` informational responses (e.g. `103 Early Hints`, which Cloudflare sends ahead of its final response) and returns the subsequent final response. `Response.StatusCode()` no longer reports `103` as the final status for such origins.
- **HTTP/2 Stream-Limit Backpressure** — Requests exceeding the peer's `SETTINGS_MAX_CONCURRENT_STREAMS` (commonly 100–128) now wait asynchronously for a free stream slot instead of failing locally with a `MAX_CONCURRENT_STREAMS exceeded` protocol error. Queued requests remain bounded by the total timeout (`SetTimeoutTotal`).
- **Multi-Address DNS Fallback** — Connections now try every resolved address (interleaved by family, RFC 8305-style) instead of only the first one, fixing dual-stack hostnames on hosts without a working IPv6 route. Applies to direct TCP and to `socks5://` local resolution, which reconnects and re-authenticates per candidate on retryable replies; `socks5h://` behaviour is unchanged. `SetTimeoutTCPConnect` still bounds all attempts together.
- **DNS Lookup Coalescing** — Concurrent system-DNS lookups for the same host are merged into a single resolver call instead of issuing duplicate queries.
- **TLS Optional Client Certificates** — Servers requesting an optional client certificate no longer fail the handshake when no certificate is configured.
- **HTTP/1 Redirect Reconnect** — A redirect canceled on a reused pooled connection now reconnects and completes instead of failing the request.
- **Session Connection Quota** — `SessionBuilder.SetMaxConnections` is enforced strictly: background QUIC probes also take a connection permit, and exceeding the cap surfaces `session connection limit reached` instead of silently overshooting the limit.
- **Linux glibc Baseline** — The Linux native library is built against glibc ≤ 2.36 again. Interim builds silently required `GLIBC_2.38` and failed to `dlopen` on Debian 12 / Ubuntu 22.04; upstream CI now pins the build image and asserts the baseline.
- **TLS 1.3 ECH + HelloRetryRequest Parity** — Real ECH retries now reuse the ECH configuration, HPKE sender context, inner random, GREASE values, and extension permutation. HRR confirmation, selected groups, cookie-only retries, dummy CCS, invalid extensions, and local fatal alerts now follow the expected browser and RFC flow.
- **ECH + PSK + HRR Binders** — Pre-shared keys remain confined to `ClientHelloInner`, with binders calculated from the correct CH1 and CH2 transcripts. Resumed real-ECH handshakes no longer abort through the previous unsupported path.
- **QUIC/H3 HRR State Transitions** — QUIC now revokes 0-RTT stream and packet state immediately after HRR, maps local TLS alerts to QUIC `CRYPTO_ERROR` codes, and completes forced P-384 retries through the H3 handshake flow.
- **TLS Handshake Record Reassembly** — TLS 1.2 and TLS 1.3 handshake messages fragmented across multiple records are now buffered until complete, fixing failures such as `truncated handshake message in record` on large certificate flights. ClientHello bytes and observable JA3 / JA4 fingerprints are unchanged.
- **ECH and DNS-Leak Hardening** — Corrects the real-ECH transcript, prevents HTTPS/SVCB discovery from bypassing remote DNS on `socks5h` and CONNECT routes, and restricts ALPS use to applicable protocol paths.
- **ALPS Fingerprint Correctness** — Decouples ALPS advertisement from the client-settings payload, matching Chrome behavior when ALPS is advertised with an empty payload.
- **HTTP/2 Frame Bounds** — Validates inbound frame sizes against the client's advertised `MAX_FRAME_SIZE` instead of trusting the peer's setting.
- **Certificate Parsing Safety** — Malformed certificate DER lengths now return a TLS error instead of triggering an out-of-bounds panic.
- **Retry Replay Safety** — Automatic retries no longer resend non-idempotent requests after ambiguous post-send failures; retries remain allowed when the request is known not to have been processed.
- **Redirect Semantics** — The total timeout now spans the complete redirect chain, `303 See Other` is followed as `GET`, and sensitive/body-specific headers are stripped according to redirect origin and method changes.
- **SOCKS5 UDP Compatibility** — QUIC-over-SOCKS5 accepts relay replies from the advertised relay IP even when the source port differs from `BND.PORT`, improving compatibility with relays using separate egress sockets or load balancers.
- **Synthetic Fingerprint Negotiability** — Libraries built with `synthetic-fp` enforce a signature-algorithm floor so recombined fingerprints remain capable of completing a TLS handshake.


## lkrequest-go 0.1.0

### 🚀 Features

- **TLS Fingerprint Control** — Byte-level ClientHello matching real browsers, via named presets
- **HTTP/2 Fingerprint Control** — SETTINGS frame order, pseudo-header order, WINDOW_UPDATE, PRIORITY frames
- **HTTP/3 + QUIC** — Feature-gated (`FeatureSupported("quic-h3")`): HTTP/3 over QUIC with fingerprint-aware Transport Parameters, Alt-Svc auto-discovery, and H2→H3 upgrade; per-request protocol preference (`SetPreferredHTTPVersion`) and negotiated version (`Response.NegotiatedVersion`)
- **Built-in Browser Presets** — Chrome 131/144/145/146/147/148/149/150, Firefox 133/147, Safari 18/26
- **Fingerprint Randomization** — Per-connection TLS extension-order jitter (drifts JA3, stays a real browser); synthetic cross-layer identities with a `synthetic-fp` library (`ClientBuilder.SetRandomize`)
- **Custom TCP Fingerprint** — JA4T-style TCP fingerprinting (`SetTCPFingerprintJA4T`)
- **Header / Cookie Order** — Send order configurable at client / session / request levels (`AddHeaderOrder` / `AddH3HeaderOrder` / `AddCookieOrder`)
- **Dual Binding Engine** — `purego` (default, no C compiler, embedded library) and `cgo` (`lkcgo` build tag, static link)
- **Session Management** — Cookie jar, connection pooling, HTTP/2 multiplexing, redirect and retry policy
- **Retry Policies** — Fixed interval / exponential backoff (with jitter)
- **Proxy Support** — SOCKS5 and HTTP CONNECT; per-session and per-request overrides
- **Proxy Pool & Session Pool** — Rotation (round-robin / random), health checks, and bad-proxy marking
- **Custom DNS Resolver** — Built-in DoH (Google / Cloudflare / Quad9) or a pluggable resolver with HTTPS/SVCB lookup
- **SOCKS5 UDP Relay Probing** — Detect UDP ASSOCIATE support for QUIC-over-SOCKS5 (`Client.Socks5UDPProbe`)
- **Connection Prewarming** — `Preconnect` / `PreconnectAsync` to pre-establish DNS + TCP + TLS
- **Streaming Responses** — `StreamingResponse` implements `io.ReadCloser`
- **Multipart / Form-Data** — Text and file uploads
- **Authentication** — Bearer token and basic auth
- **Auto-Decompression** — gzip / br / deflate / zstd, controllable via `AcceptEncoding`
- **Certificate Management** — Custom CA, verify toggle, native certs, ECH, and TLS keylog
- **Context Support** — Cancelable (`SendWithContext`) and async (`SendAsync`, channel-based) sends
- **Request Diagnostics** — Per-phase timings (DNS / TCP / TLS / TTFB) via `DiagnosticsJSON`
- **Typed Errors** — Error code, phase, retryable flag, and diagnostics on `LkError`
