# lkrequest-go

Go bindings for [`lkrequest`](https://github.com/EZXLabs/lkrequest-rs) — a Rust HTTP client with
byte-level **TLS / HTTP2 / HTTP3(QUIC) fingerprint control**, for browser emulation,
anti-detection research, and web scraping.

[English](README.md) | [简体中文](README-zh.md)

The package wraps the `lkrequest-ffi` C ABI behind a thin, idiomatic Go API:

- `Client` — immutable, shareable client configuration (fingerprint, TLS, timeouts)
- `Session` — connection pool, proxy, redirect, retry, and cookie state
- `Request` — single-owner, consume-on-send request builder
- `Response` / `StreamingResponse` — buffered and streaming (`io.ReadCloser`) reads

## Features

- **Browser fingerprint presets** — Chrome / Firefox / Safari TLS+HTTP2 fingerprints by name, or loaded from a profile bundle you can export, edit and load back
- **Fingerprint randomization** — per-connection TLS extension-order jitter (drifts JA3, stays a real browser); synthetic cross-layer identities with the `synthetic-fp` library
- **HTTP/2 & HTTP/3 (QUIC)** — Alt-Svc discovery, per-request protocol preference, H3 header order, per-request HTTP/2 priority weight
- **Dual engine** — default `purego` (no C compiler, embedded library) or `lkcgo` (CGo, linking the bundled shared library)
- **Streaming responses** — `SendStreaming()` returns an `io.ReadCloser`
- **Cookie management** — jar, attribute-aware `SetCookieWithAttrs` / `GetCookiesWithAttrs` / `GetAllCookies`, per-request override
- **Multipart upload** — text + file parts, with Chrome-accurate `----WebKitFormBoundary` boundaries
- **Retries & attempt limits** — fixed interval / exponential backoff (with jitter), opt-in replay on a closed connection and H2→H1 fallback, caps on connection starts and extra submissions, and a JSON summary of a request's attempts
- **Timeouts** — per-phase, total and response read-idle timeouts, each of which can be turned off
- **Proxy** — per-session / per-request / proxy pool / session pool, with rotation, health checks, and bad-proxy marking; static header credentials (`Preshared`, `PrivateToken`, `Bearer`, …) via `ProxyConfig`
- **MASQUE proxies (experimental)** — QUIC tunneled through an RFC 9298 CONNECT-UDP proxy, with a private CA for the proxy hop; needs a `masque`-enabled library
- **Connection prewarming** — `Preconnect` / `PreconnectAsync`
- **Automatic decompression** — gzip / br / deflate / zstd, controllable via `AcceptEncoding`
- **Certificate management** — custom CA, verify toggle, native certs, ECH, TLS keylog, Trust Anchor Identifiers negotiation, TLS session cache control
- **Header / cookie ordering** — configurable at client / session / request levels; cookies you do not order explicitly follow Chrome's own order; HTTP/1.1 header names in Title-Case or lowercase
- **Custom DNS** — built-in DoH resolvers (Google / Cloudflare / Quad9) or a custom resolver callback
- **SOCKS5 UDP relay probing** — detect UDP ASSOCIATE support for QUIC-over-SOCKS5
- **Request diagnostics** — per-phase timings (DNS / TCP / TLS / TTFB) via `DiagnosticsJSON()`
- **Context support** — cancelable (`SendWithContext`) and async (`SendAsync`) sends
- **Typed errors** — error code, phase, retryable flag, and diagnostics on `LkError`

## Installation

```bash
go get github.com/EZXLabs/lkrequest-go/lkrequest@latest
```

```go
import "github.com/EZXLabs/lkrequest-go/lkrequest"
```

## Quick Start

Top-level helpers use a process-wide default client:

```go
resp, err := lkrequest.Get("https://httpbin.org/get")
if err != nil {
	log.Fatal(err)
}
defer resp.Close()

fmt.Println(resp.StatusCode())
fmt.Println(resp.String())
```

The full `Client → Session → Request → Response` flow (all handles are explicitly closed):

```go
client, err := lkrequest.NewClient("chrome_150") // or NewDefaultClient()
if err != nil {
	log.Fatal(err)
}
defer client.Close()

session, err := lkrequest.NewSession(client)
if err != nil {
	log.Fatal(err)
}
defer session.Close()

resp, err := session.Get("https://httpbin.org/get").Send()
if err != nil {
	log.Fatal(err)
}
defer resp.Close()

fmt.Println(resp.StatusCode(), resp.Header("content-type"))
```

## Usage

### Building & sending requests

`Session` has one helper per HTTP verb. Each returns a `*Request` to chain
setters onto, so a construction failure travels with the request and is
reported by `Send()` rather than needing its own error check.

```go
resp, err := session.Get("https://httpbin.org/get").Send()

// Get / Post / Put / Delete / Head / Patch / Options,
// plus Request(method, url) for anything else.
resp, err = session.Request("PROPFIND", "https://example.com/dav").Send()
```

`NewRequest(session, method, url)` is the same construction with the error
returned up front. `SessionPoolGuard` carries the same set of helpers.

`Request` is **consume-on-send** — chain setters, then send. Reuse via `Clone()`.

```go
resp, err := session.Post("https://httpbin.org/post").
	AddHeader("accept", "application/json").
	AddQuery("page", "1").
	SetBearerAuth("my-token").                     // or SetBasicAuth(user, pass)
	SetJSON(map[string]string{"hello": "world"}).  // marshals with encoding/json
	Send()                                         // SetJSONBody takes a pre-encoded document
```

Cancelable and async sends:

```go
resp, err := req.SendWithContext(ctx)          // honors context cancellation
respCh, errCh := req.SendAsync(ctx)            // runs in a goroutine, returns channels
```

### Custom client

`ClientBuilder` is a chained builder; setter errors surface at `Build()`.

```go
client, err := lkrequest.NewClientBuilder().
	SetPreset("chrome_144").
	SetVerify(true).
	SetUseNativeCerts(true).
	AddDefaultHeader("Accept-Language", "en-US").
	SetTimeoutTotal(30000).             // milliseconds
	SetMaxConnectionsPerSession(16).
	SetFallbackH2ToH1(true).
	Build()
```

HTTP/1.1 requests spell header names in Title-Case (`Content-Type`) by default.
`SetHTTP1TitleCaseHeaders(false)` sends them in lowercase (`content-type`), default headers
and generated ones such as `content-length` too; only `Host` stays capitalized. Either way
the spelling given to `AddHeader` is not kept. HTTP/2 and HTTP/3 always send lowercase names.

```go
client, err := lkrequest.NewClientBuilder().
	SetHTTP1TitleCaseHeaders(false). // HTTP/1.1 only
	Build()
```

### Timeouts, replays and the session cache

Every deadline can be turned off. The total timeout, 120 seconds by default, covers a request from
connecting to the end of its body, streaming bodies included; `SetTimeoutDNS`,
`SetTimeoutTCPConnect`, `SetTimeoutTLSHandshake`, `SetTimeoutTTFB` and `SetTimeoutQUICConnect`
bound its phases, and `SetTimeoutReadIdle` (off by default) how long each read of a body may wait for
data. Each has a `DisableTimeout…` counterpart, and calling the setter again turns it back on; `0`
times out at once rather than disabling.

```go
client, err := lkrequest.NewClientBuilder().
	DisableTimeoutTotal().     // a long-lived stream…
	SetTimeoutReadIdle(45000). // …that may not stall for 45 s
	Build()
```

A request is not sent again when its pooled connection closes before an answer, and HTTP/2 does not
fall back to HTTP/1.1, unless the client enables `SetRetryOnConnectionClose(true)` or
`SetFallbackH2ToH1(true)`. Session retry policies (`SetRetryFixed`, `SetRetryExponential`) are
configured apart from these.

TLS and QUIC sessions are cached for resumption. `SetTLSSessionCacheEnabled(false)` makes every new
connection a full handshake, `SetTLSSessionCacheTTL(ms)` limits how long a session may be resumed
(`0` caches none), and `ResetTLSSessionCacheTTL()` restores the default lifetimes.

### Session configuration & retry

```go
session, err := lkrequest.NewSessionBuilder(client).
	SetProxy("socks5://user:pass@host:1080").
	SetMaxRedirects(5).
	SetHTTP2Only().
	SetDefaultAcceptEncoding(lkrequest.AcceptEncodingGzip | lkrequest.AcceptEncodingBr).
	SetRetryExponential(3, 50, 500, true). // maxRetries, baseMs, maxMs, jitter
	Build()

// Fixed interval instead: SetRetryFixed(2, 100)  // maxRetries, intervalMs
```

### Attempt limits and summaries

Two optional caps bound what a single request may cost. `SetConnectAttemptLimit` caps the TCP and
QUIC connections it may start: failed starts and the losers of a race count, reusing a pooled
connection does not, and a request over the cap fails with `ErrResourceLimitExceeded`.
`SetExtraHTTPAttemptLimit` caps how many times it may be sent again while recovering from a failure;
the first submission and redirects do not count. Set on a `ClientBuilder` they apply to each request
of the client, not as a shared quota; a `SessionBuilder` or a `Request` may only tighten what it
inherits, and a looser value is reported by `Build()` or `Send()`. Neither enables a retry or sets a
deadline.

```go
client, err := lkrequest.NewClientBuilder().
	SetRetryOnConnectionClose(true).
	SetExtraHTTPAttemptLimit(2).
	Build()
// …
resp, err := session.Get(url).
	SetConnectAttemptLimit(1).
	EnableAttemptSummary(true).
	Send()
fmt.Println(resp.AttemptSummaryJSON())
// {"connect_initiations":1,"http_submissions":1,"extra_http_submissions":0,"outcome":"succeeded",…}
```

`ConnectAttemptLimit()` and `ExtraHTTPAttemptLimit()`, on the builders, `Client`, `Session` and
`Request`, report an `AttemptLimit`: the value set at that level, the one in force and the `Scope` it
comes from. `AttemptSummaryJSON()` on a `Response`, `StreamingResponse` or `LkError` returns the
summary `EnableAttemptSummary(true)` asked for — submissions, connection starts, recoveries, the
limits in force and, after a failure, the `denial_reason` — and `""` for a request without one.

### Cookies

```go
session.SetCookie("https://example.com", "token", "abc123")
session.SetCookieWithAttrs("https://example.com", "sid", "xyz", lkrequest.CookieAttrs{
	Path: "/api", Domain: "example.com", Secure: true, HTTPOnly: true,
})
val, _ := session.GetCookie("https://example.com", "token")
jar, _ := session.GetCookies("https://example.com") // []Cookie; GetCookiesJSON returns the raw payload

// With attributes: domain, path, Secure, HttpOnly, SameSite and expiry (nil for a session cookie)
scoped, _ := session.GetCookiesWithAttrs("https://example.com/api") // []CookieInfo, longest path first
all, _ := session.GetAllCookies()                                   // every domain, oldest first — the view to persist

req.SetCookieOverride("token", "fresh-value") // per-request override of the jar
```

### Multipart upload

```go
mp := lkrequest.NewMultipart().
	AddText("title", "report").
	AddFile("document", "report.pdf", "application/pdf", pdfBytes)
req.SetMultipart(mp) // transfers ownership; the request can no longer be cloned
```

### Streaming responses

```go
stream, err := req.SendStreaming()
if err != nil {
	log.Fatal(err)
}
defer stream.Close()

body, err := io.ReadAll(stream) // StreamingResponse implements io.ReadCloser
```

### Proxy pool & session pool

```go
// Proxy pool — rotates URLs, marks bad proxies
pool, err := lkrequest.NewProxyPoolBuilder().
	AddProxies([]string{"socks5://p1:1080", "http://p2:8080"}).
	SetRotation(lkrequest.RotationRoundRobin).
	Build()
guard, _ := pool.Acquire()
defer guard.Close()
proxyURL := guard.URL()

// Session pool — each session bound to a different proxy
sp, err := lkrequest.NewSessionPoolBuilder(client).
	AddProxy("http://p1:8080").
	SetMaxSessions(8).
	Build()
g, _ := sp.Acquire()
defer g.Close()
poolReq, _ := g.NewRequest("GET", "https://example.com")
```

A pool with proxies never falls back to a direct connection. While every proxy
is cooling down, `Acquire` waits for one to recover; once every proxy is removed
for good, it waits forever. Bound the wait with `AcquireAsync(ctx)`, which gives
up when `ctx` ends. Only a pool built without any proxy hands out direct leases,
whose `URL()` is `""`. After `SessionPool.MarkBad(g)`, closing `g` discards its
session instead of returning it to the idle pool.

### Proxy credentials beyond username/password

A proxy URL only carries a username and password. `ProxyConfig` also takes a
static HTTP credential and the header it travels in:

```go
proxy, err := lkrequest.ParseProxyConfig("https://proxy.example:443")
if err != nil {
	log.Fatal(err)
}
defer proxy.Close() // builders copy the config

proxy.SetHTTPAuth("Preshared", "<psk>"). // sent as "Preshared <psk>"
	SetAuthHeader(lkrequest.ProxyAuthHeaderAuthorization) // default: Proxy-Authorization

session, err := lkrequest.NewSessionBuilder(client).SetProxyConfig(proxy).Build()
// or pool it: lkrequest.NewProxyPoolBuilder().AddProxyConfig(proxy)
```

`proxy.Identity()` is the key `ProxyPool.MarkBad` expects — a proxy URL is not.
The mark is in effect when `MarkBad` returns.

### MASQUE (experimental)

A `masque://` proxy carries the QUIC connection to the origin inside a
CONNECT-UDP tunnel. `MasqueConfig` configures the hop to the proxy; its trust
settings are independent of the client's `SetVerify` and CA certificates.

```go
masque, err := lkrequest.NewMasqueConfig() // errors if the library lacks MASQUE
if err != nil {
	log.Fatal(err)
}
defer masque.Close()
masque.AddCACertsPEM(caPEM). // a private CA for the proxy hop
	SetUseNativeCerts(false)

client, _ := lkrequest.NewClientBuilder().SetMasqueConfig(masque).Build()
proxy, _ := lkrequest.ParseProxyConfig("masque://proxy.example") // port defaults to 443
session, _ := lkrequest.NewSessionBuilder(client).SetHTTP3Only().SetProxyConfig(proxy).Build()
```

- A MASQUE route carries HTTP/3 only. HTTP/1.1 and HTTP/2 requests through it
  fail instead of connecting directly, even with `SetFallbackProxyToDirect(true)`.
- Pools skip `masque://` proxies in health checks unless given
  `SetMasqueTunnelProbe(masque)` together with `SetHealthCheck`.
- The embedded libraries are built without the `masque` feature, so with them
  `NewMasqueConfig` returns an error and `masque://` URLs are rejected.

### Browser fingerprints & randomization

```go
presets, _ := lkrequest.ListPresetsJSON()            // enumerate available presets
detail, _ := lkrequest.GetPresetDetailJSON("chrome_150")

// Tier 1 — per-connection JA3 jitter, always available, still a real browser
client, _ := lkrequest.NewClientBuilder().
	SetRandomize(lkrequest.RandomizeExtensionOrder, lkrequest.FingerprintLayersAll).
	Build()
```

`RandomizeRecombine` / `RandomizeFull` synthesize non-browser fingerprints (for blocklist targets)
and require a library built with the `synthetic-fp` feature; otherwise `SetRandomize` returns an error.

A request can override the weight its HTTP/2 HEADERS frame carries in the legacy
priority field, for a request the browser weighs differently from the preset's
default. It leaves the `priority` header alone; HTTP/1.1 and HTTP/3 ignore it.

```go
resp, err := session.Get("https://example.com/").
	SetH2PriorityWeight(21). // the wire value, 0-255 (weight 22)
	Send()
```

### Custom profiles and profile bundles

A fingerprint can come from data instead of a preset name. `ExportPresetJSON` writes a built-in
preset as a complete profile bundle: a JSON document (`"format": "lkrequest-profile-bundle"`,
`"schema_version": 2`) holding every layer a preset does, HTTP/2 runtime behaviour and protocol
policy included, which `SetPresetJSON` or `SetPresetFile` loads back, edited or not.

```go
bundle, err := lkrequest.ExportPresetJSON("chrome_157") // edit, store, ship
client, err := lkrequest.NewClientBuilder().
	SetPresetJSON(bundle). // or SetPresetFile("chrome-157.json")
	Build()
```

Single layers can be replaced on top of a preset or a bundle with native profile JSON, in the form
`FingerprintInfoJSON()` reports for the layer: `SetTLSProfileJSON`, `SetH2ProfileJSON` (data fields
only, so the preset's HTTP/2 runtime behaviour gives way to the baseline), `SetQUICTLSProfileJSON`,
and `ClearQUICFingerprint` to make QUIC use the TCP TLS profile. They take precedence over the preset
or bundle whichever is called first, and an empty string hands the layer back.

- Inputs are limited to 1 MiB and validated, and `Build()` reports what the library rejects. A bundle
  error is an `LkError` (`ErrInvalidConfig`, `ErrResourceLimitExceeded` or `ErrIO`) whose
  `DiagnosticsJSON()` carries a `preset_error`: a stable `code`, the JSON Pointer `path` and a `message`.
- `SetPreset` after `SetPresetJSON` replaces the bundle, and `SetPresetJSON("")` removes it. A bundle
  cannot change proxy, CA or verification settings, and loading reads nothing but the file you name.
- `FingerprintInfoJSON()` reports the QUIC TLS profile as `quic_tls`, `null` when QUIC uses the TCP one.
- Loading a bundle reproduces its fields; that alone does not make a fingerprint match a browser on
  the wire.

### Trust Anchor Identifiers

Chrome presets advertise trust anchor identifiers, IDs of the roots the client accepts, in the
TLS `trust_anchors` extension (draft-ietf-tls-trust-anchor-ids). By default a client sends the
list its preset carries, as the browser does, and does nothing more with it. `SetTrustAnchorPolicy`
can instead send the IDs of the roots the client trusts, and let a request reconnect once when its
handshake fails after the server named the roots it can serve. None of this changes which
certificates are trusted.

```go
client, err := lkrequest.NewClientBuilder().
	SetPreset("chrome_157").
	SetTrustAnchorPolicy(lkrequest.TrustAnchorPolicy{
		Advertisement: lkrequest.TrustAnchorAdvertiseTrusted,
		Recovery:      lkrequest.TrustAnchorRecoveryOnce,
	}).
	Build()

// A private root: trust it as usual, then bind the ID its deployment assigned.
identity, err := lkrequest.TrustAnchorIdentity(rootDER)
client, err = lkrequest.NewClientBuilder().
	AddCACertMemory(rootDER).
	AddTrustAnchorBinding(identity, "81fd5901").
	Build()
```

- `Build()` reports, as `ErrInvalidConfig`, a policy it cannot honour. Recovery needs certificate
  verification and cannot combine with `SetFallbackProxyToDirect`. Any policy but the zero value needs
  a preset whose TLS profiles carry the extension (Chrome's do; Firefox's and Safari's do not), and
  rules out synthetic TLS randomization. A binding must name a root the client trusts.
- `DiagnosticsJSON()` of a response or an `LkError` carries a `trust_anchor` object for the TLS
  connection the request opened: the requested and available IDs, `verified_anchor` (the
  `TrustAnchorIdentity` of the root that verified the chain), `authenticated`, `recovered`, and the
  failed first attempt as `previous_attempt`. It is `null` for a request on a pooled connection.
- A recovery that fails too ends the request with an error that is not retryable. So does any
  handshake failure after the server listed its trust anchors, with or without recovery.

### QUIC / HTTP3

```go
if lkrequest.FeatureSupported("quic-h3") {
	session, _ := lkrequest.NewSessionBuilder(client).
		SetHTTP3WithFallback(). // prefer H3, fall back to H2
		Build()
}
```

### Custom DNS

```go
client, _ := lkrequest.NewClientBuilder().
	SetDNS(lkrequest.DnsCloudflareHTTPS). // built-in DoH; or SetDNSCustom("1.1.1.1:53")
	Build()
```

### Logging

```go
lkrequest.InitLog("info", "/tmp/lkrequest.log") // level string, log file

lkrequest.InitLogCallback(func(level lkrequest.LogLevel, target, msg string) {
	log.Printf("[%v] %s: %s", level, target, msg)
}, lkrequest.LogLevelInfo)
```

### Error handling

```go
resp, err := req.Send()
if err != nil {
	var lkErr *lkrequest.LkError
	if errors.As(err, &lkErr) {
		fmt.Println(lkErr.Code(), lkErr.Phase(), lkErr.IsRetryable())
	}
	log.Fatal(err)
}
defer resp.Close()

if err := resp.ErrorForStatus(); err != nil {
	// 4xx / 5xx
}
```

## Dual Engine

Two interchangeable binding engines fill the same `ffi_lk_*` function pointers:

| Engine | Build tag | How it loads the library | Needs a C compiler |
|---|---|---|---|
| `purego` (default) | `!lkcgo` | extracts the embedded shared library to a temp dir and `dlopen`s it | No |
| `cgo` | `lkcgo` | `#include "lkrequest.h"` and links the shared library in `lib/<platform>/` | Yes |

```bash
go build ./lkrequest/...              # purego (default)
go build -tags lkcgo ./lkrequest/...  # cgo
```

A cgo binary loads that library when it starts. On darwin it finds it through an rpath to
`lib/darwin_arm64` of the module it was built from; on linux, put `lib/linux_amd64` on
`LD_LIBRARY_PATH` (or install the library); on windows, place `lkrequest_ffi.dll` next to the
executable or on `PATH`.

## Platform Support

| Platform | Status |
|---|---|
| linux / amd64 | ✅ shipped (embedded library) |
| windows / amd64 | ✅ shipped |
| darwin / arm64 | ✅ shipped (since 0.2.1) |

## Requirements

- **Go 1.21+**.

## API Overview

| Group | Symbols |
|---|---|
| Top-level helpers | `Get`, `PostJSON`, `NewClient`, `NewDefaultClient`, `NewSession`, `NewRequest`, `NewMultipart`, `ParseProxyConfig`, `NewMasqueConfig`, `TrustAnchorIdentity`, `ExportPresetJSON` |
| HTTP verbs | `Session.Get` / `Post` / `Put` / `Delete` / `Head` / `Patch` / `Options` / `Request` — same set on `SessionPoolGuard` |
| Builders | `NewClientBuilder`, `NewSessionBuilder`, `NewProxyPoolBuilder`, `NewSessionPoolBuilder` |
| Core types | `Client`, `Session`, `Request`, `Response`, `StreamingResponse`, `ProxyPool`, `SessionPool`, `ProxyConfig`, `MasqueConfig`, `Multipart`, `Cookie`, `CookieInfo`, `Redirect`, `TrustAnchorPolicy`, `AttemptLimit` |
| Introspection | `ABIVersion`, `LibraryVersion`, `FeatureSupported`, `ListPresetsJSON`, `GetPresetDetailJSON` |
| Logging | `InitLog`, `InitLogCallback` |
| Enums | `RandomizeMode`, `FingerprintLayers`, `AcceptEncoding`, `RotationStrategy`, `HttpVersion`, `PreferredHTTPVersion`, `DnsConfig`, `LogLevel`, `ProxyAuthHeader`, `TrustAnchorAdvertisement`, `TrustAnchorRecovery`, `AttemptLimitScope` |

Full API docs: `go doc github.com/EZXLabs/lkrequest-go/lkrequest`.

## Notes

- `Client` and `Session` are safe to share across goroutines; `Request` is consume-on-send (reuse via `Clone()`).
- `StreamingResponse` implements `io.ReadCloser`.
- A `ClientBuilder` built without `SetPreset`, like `NewDefaultClient()`, uses the library's default preset — currently `chrome_155` — header order included. Call `SetPreset` for another fingerprint, or `AddHeaderOrder` / `AddCookieOrder` to pin a specific order.
- QUIC/H3, custom DNS resolver, SOCKS5 UDP probe, preferred/negotiated HTTP version, `ProxyConfig` / `MasqueConfig`, cookie attribute readers, the per-request H2 priority weight, HTTP/1.1 header casing, Trust Anchor Identifiers, custom profiles and profile bundles, the read-idle timeout and the `DisableTimeout…` setters, TLS session cache settings, attempt limits and summaries, and synthetic `SetRandomize` tiers require a library that exports the corresponding symbols. Against an older embedded library these return an `lk: ... not supported by the loaded lkrequest library` error (builders defer it to `Build()`); the package still loads normally.
- `FeatureSupported(name)` tells whether the loaded library has a feature, by names such as `preset-bundle-v2`, `read-idle-timeout`, `optional-total-timeout`, `optional-phase-timeouts`, `tls-session-cache`, `connect-attempt-limit`, `extra-http-attempt-limit` and `attempt-summary`.
