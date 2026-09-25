# Changelog

All important changes to this project will be documented in this file.

## lkrequest-go 0.2.3

Moves the module to the renamed GitHub organization. The API and the bundled
`lkrequest-ffi` native engine are unchanged from 0.2.2; the import path is the
only thing to update.

### ⚠️ Breaking Changes

- **New Module Path** — the module is now `github.com/EZXLabs/lkrequest-go`. Replace `github.com/EZ-XLabs/lkrequest-go/lkrequest` with `github.com/EZXLabs/lkrequest-go/lkrequest` in your imports and `go get github.com/EZXLabs/lkrequest-go/lkrequest@v0.2.3`. Versions up to 0.2.2 declare the old path and can only be fetched under it; requesting them under the new one fails with `module declares its path as: github.com/EZ-XLabs/lkrequest-go`. From 0.2.3 on, only the new path works.

## lkrequest-go 0.2.2

Updates the bundled `lkrequest-ffi` native engine, adds the Chrome 153,
Chrome 154 and Firefox 156 presets, and binds the new proxy-configuration and
MASQUE entry points. Every addition is new API — the native surface grew by 21
symbols and changed none — so every existing call shape keeps working.

Most of this release is invisible until you look at the wire. The `Cookie`
header was being sent in an arbitrary order that differed between sessions, and
the QUIC wire image of the existing Chrome presets has changed — both are
fingerprint fixes that apply to code you have already written. The largest
performance fix is an HTTP/3 fallback path that silently rebuilt the connection
for every request. Two entries change behaviour you may be relying on and are
worth reading before upgrading: the default port for a portless HTTP proxy URL,
and the proxy identity encoding.

### 🚀 Features

- **Chrome 153 Preset** — `SetPreset("chrome_153")` / `NewClient("chrome_153")`, captured from Chrome 153.0.8010.37 (Windows, September 2026). TCP TLS keeps Chrome 152's signature-algorithm GREASE and trims the shuffled `0xca34` trust-anchor identifier vector from 32 values to 28. HTTP/2 keeps Chrome 152's SETTINGS, connection window, pseudo-header order and initial HEADERS priority. QUIC keeps the classic nine signature algorithms, shuffled ClientHello extensions and the same 28 identifiers.
- **Chromium QUIC Wire Generation** — the Chrome presets now follow Chromium's own generation rules for the randomized parts of the HTTP/3 and QUIC wire image: a random reserved SETTINGS id and value, a 0–3 byte control-stream GREASE frame and no request-stream GREASE; reserved transport parameters drawn from the full 62-bit GREASE id space with 0–15 byte payloads and a randomized GREASE version position; and client Initials that split, pad and shuffle their CRYPTO data with interleaved PING/PADDING, including across loss, Retry and 0-RTT. **This applies to the Chrome presets you are already using, not only to Chrome 153.** Custom profiles and previously serialized ones are unaffected — both underlying fields default to off.
- **Chrome 154 Preset** — `SetPreset("chrome_154")`, captured from Chrome 154.0.8037.58 (Windows, September 2026). Every layer keeps Chrome 153's shape: TCP TLS with its 28 trust-anchor identifiers, HTTP/2, QUIC TLS, QUIC transport parameters and Chromium Initial packetization. HTTP/3 application settings keep Chrome 153's baseline, as upstream's capture path received no QUIC response to compare against.
- **Firefox 156 Preset** — `SetPreset("firefox_156")`, captured from Firefox 156.0.1 (Windows, September 2026). Against Firefox 147 it drops the two ECDSA CBC cipher suites and both FFDHE groups and moves ECH GREASE to AES-128-GCM; HTTP/2 is unchanged. It has no QUIC profile: upstream keeps HTTP/3 off until it can reproduce Firefox's transport and Initial-packet fields exactly.
- **Proxy Credentials Beyond Username/Password** — `ParseProxyConfig` returns a `ProxyConfig` that can carry a static HTTP credential — `SetHTTPAuth("Preshared", psk)`, `PrivateToken`, `Bearer` and the like — and choose its header with `SetAuthHeader`: `Proxy-Authorization` by default, or `Authorization` for masque-go and RFC 9484 style deployments. `SetUserPass` sets a username and password without URL-encoding them. Hand it to `SessionBuilder.SetProxyConfig` or `ProxyPoolBuilder.AddProxyConfig`; both copy it. A header credential on a SOCKS5 proxy, which has nowhere to put one, is an error rather than a credential silently never sent.
- **MASQUE Proxies (Experimental)** — a `masque://` proxy carries the QUIC connection to the origin inside an RFC 9298 CONNECT-UDP tunnel. `NewMasqueConfig` configures the hop to the proxy — a private CA via `AddCACertsPEM` / `AddCACertDER`, `SetUseNativeCerts`, `SetServerName`, tunnel reclamation and, for development only, `SetVerify` — independently of the client's own trust settings. `ClientBuilder.SetMasqueConfig` applies it to requests, and `SetMasqueTunnelProbe` on both pools health-checks MASQUE proxies with a real handshake instead of skipping them. A MASQUE route carries HTTP/3 only and never falls back to a direct connection. **This needs a library built with the `masque` feature, and the bundled libraries are not:** with them `NewMasqueConfig` returns an error saying so and `masque://` URLs are rejected.

### 🐞 Fixes

- **`Cookie` Header Order Now Matches Chrome** — the session jar iterated a hash map, so the order cookies were sent in was arbitrary and differed from one session to the next, which is itself a tell. `AddCookieOrder` only pinned the names it listed; everything else, including the relative order of same-name cookies at different paths, stayed random. The jar now records creation order and sorts the way Chrome's `CookieSorter` does — longer paths first, then older cookies first — verified upstream against a Chrome 153 capture fed the same `Set-Cookie` sequence. `AddCookieOrder` still takes precedence for the names it lists. `Session.GetCookie` also now reliably returns the longest-path match, which it previously documented but did not do.
- **HTTP/3 Fallback No Longer Loses the Connection Pool** — with HTTP/3 enabled, an origin whose QUIC was unreachable re-established TCP and TLS for *every* request. The fallback did pool its HTTP/2 connection, but the pool lookup for the HTTP/3-with-fallback mode only ever searched for an HTTP/3 connection and never found it. Nothing failed and nothing was logged — the only symptom was latency, roughly 3× against affected origins, which includes any host advertising Alt-Svc `h3` from a network where UDP is blocked. Healthy TCP connections and preconnects are now reused, failed QUIC reservations are released before falling back, and HTTP/3 advertisements from DNS are promoted into protocol selection.
- **Portless HTTP Proxy URLs Now Default to Port 80** — `SetProxy("http://proxy.example.com")` previously resolved to port **8080**, a common proxy port but not the scheme's default. It now resolves to **80**, as the URL says. Spell the port out (`http://proxy.example.com:8080`) if 8080 was what you relied on. `https://` proxies are unaffected — 443 either way.
- **Proxy Identity Encoding Changed** — the key used for proxy-pool cooldown and failure tracking is now a JSON array of ordered hops, each carrying protocol, host, port and a credential fragment — `user:<name>` for a username (never the password), an opaque fingerprint for a static HTTP credential — replacing the previous `user@host:port` and `a>b>c` chain form. The old encoding could not tell `socks5` from `socks5h`, so two chains resolving hostnames at different hops shared a single cooldown entry. Existing cooldown state does not carry across the change.
- **`ProxyPool.MarkBad` Is Usable from Go** — it takes the proxy's identity, which Go had no way to obtain; the one proxy string Go did expose, the URL, matches nothing, so `MarkBad(url)` succeeded and did nothing. `ProxyConfig.Identity()` now returns the key — for a proxy added by URL, `ParseProxyConfig(url)` and ask it. The mark is recorded asynchronously and takes effect shortly after `MarkBad` returns.
- **HTTP/3 Responses Using the QPACK Dynamic Table** — the HTTP/3 client decoded response headers statelessly, although the Chrome presets advertise a QPACK dynamic table. A server that took up the offer and encoded headers against it failed the request with `QPACK_DECOMPRESSION_FAILED`. Responses are now decoded against the dynamic table, and the connection honours the QPACK capacity the profile advertises.
- **Replay Safety Across a Protocol Switch** — a request that changed protocol mid-flight could be retried even though the earlier attempt may already have reached the server. Retry eligibility is now tracked through the switch, closing the gap left by the replay-safety work in 0.2.0.
- **Streaming Responses Honour HSTS** — `SendStreaming()` skipped the session's HSTS handling: a known HSTS host was not upgraded to `https://` on that path, and the response did not update HSTS state. Both now match `Send()`.
- **`TCP_NODELAY` Is On by Default for Direct Connections** — interactive HTTP benefits from disabling Nagle's algorithm, so direct TCP connections now set it unless a fingerprint turns it off explicitly. `SetTCPFingerprintJA4T` does not encode this field, so there is currently no way to turn it back off from Go.
- **Faster HTTP/2 Request Processing** — the native HTTP/2 driver's write path was reworked and the HPACK encoder replaced with an in-house implementation. No API change. The per-header encoding policy that the browser presets depend on (indexed vs literal, Huffman on or off) drives the new encoder unchanged, and upstream's fingerprint goldens and packet-capture baselines were not regenerated, so the wire shape is unchanged.

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
