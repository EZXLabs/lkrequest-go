# Changelog

All important changes to this project will be documented in this file.

## lkrequest-go 0.2.0

Updates the bundled `lkrequest-ffi` native engine with upstream stability,
interoperability, and security fixes. This release adds no new Go API; existing
clients benefit automatically when using the matching 0.1.1 FFI libraries.

### 🚀 Features

- **Dedicated Chrome 150 QUIC / HTTP/3 Preset** — Chrome 150 now uses dedicated QUIC-TLS and H3 profiles instead of reusing Chrome 146. The preset covers its captured signature algorithms, QUIC transport parameters, QPACK limits, navigation `PRIORITY_UPDATE`, connection ID lengths, and Initial datagram sizing.

### 🐞 Fixes

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
