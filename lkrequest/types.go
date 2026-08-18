package lkrequest

import (
	"fmt"
	"strings"
)

type ErrorCode int32

const (
	ErrUnknown               ErrorCode = 1
	ErrInvalidArgument       ErrorCode = 2
	ErrInternalPanic         ErrorCode = 3
	ErrStreamClosed          ErrorCode = 4
	ErrInvalidHandle         ErrorCode = 5
	ErrBusy                  ErrorCode = 6
	ErrResourceLimitExceeded ErrorCode = 7
	ErrInvalidConfig         ErrorCode = 8
	ErrNotFound              ErrorCode = 9
	ErrDecompressionFailed   ErrorCode = 10
	ErrTLS                   ErrorCode = 11
	ErrHTTP                  ErrorCode = 12
	ErrH2                    ErrorCode = 13
	ErrIO                    ErrorCode = 14
	ErrProxy                 ErrorCode = 15
	ErrConnection            ErrorCode = 16
	ErrTimeout               ErrorCode = 17
	ErrTooManyRedirects      ErrorCode = 18
	ErrPool                  ErrorCode = 19
	ErrURLParse              ErrorCode = 20
	ErrStatus                ErrorCode = 21
	ErrQUIC                  ErrorCode = 22
	ErrH3                    ErrorCode = 23
)

func (c ErrorCode) String() string {
	switch c {
	case ErrUnknown:
		return "ErrUnknown"
	case ErrInvalidArgument:
		return "ErrInvalidArgument"
	case ErrInternalPanic:
		return "ErrInternalPanic"
	case ErrStreamClosed:
		return "ErrStreamClosed"
	case ErrInvalidHandle:
		return "ErrInvalidHandle"
	case ErrBusy:
		return "ErrBusy"
	case ErrResourceLimitExceeded:
		return "ErrResourceLimitExceeded"
	case ErrInvalidConfig:
		return "ErrInvalidConfig"
	case ErrNotFound:
		return "ErrNotFound"
	case ErrDecompressionFailed:
		return "ErrDecompressionFailed"
	case ErrTLS:
		return "ErrTLS"
	case ErrHTTP:
		return "ErrHTTP"
	case ErrH2:
		return "ErrH2"
	case ErrIO:
		return "ErrIO"
	case ErrProxy:
		return "ErrProxy"
	case ErrConnection:
		return "ErrConnection"
	case ErrTimeout:
		return "ErrTimeout"
	case ErrTooManyRedirects:
		return "ErrTooManyRedirects"
	case ErrPool:
		return "ErrPool"
	case ErrURLParse:
		return "ErrURLParse"
	case ErrStatus:
		return "ErrStatus"
	case ErrQUIC:
		return "ErrQUIC"
	case ErrH3:
		return "ErrH3"
	default:
		return fmt.Sprintf("ErrorCode(%d)", int32(c))
	}
}

type Phase int32

const (
	PhaseNone          Phase = 0
	PhaseDNSResolution Phase = 1
	PhaseTCPConnect    Phase = 2
	PhaseProxyTunnel   Phase = 3
	PhaseTLSHandshake  Phase = 4
	PhaseH2Negotiation Phase = 5
	PhaseH2CUpgrade    Phase = 6
	PhaseHTTPRequest   Phase = 7
	PhaseQUICHandshake Phase = 8
	PhaseH3Negotiation Phase = 9
	PhaseQUICFallback  Phase = 10
)

func (p Phase) String() string {
	switch p {
	case PhaseNone:
		return "PhaseNone"
	case PhaseDNSResolution:
		return "PhaseDNSResolution"
	case PhaseTCPConnect:
		return "PhaseTCPConnect"
	case PhaseProxyTunnel:
		return "PhaseProxyTunnel"
	case PhaseTLSHandshake:
		return "PhaseTLSHandshake"
	case PhaseH2Negotiation:
		return "PhaseH2Negotiation"
	case PhaseH2CUpgrade:
		return "PhaseH2CUpgrade"
	case PhaseHTTPRequest:
		return "PhaseHTTPRequest"
	case PhaseQUICHandshake:
		return "PhaseQUICHandshake"
	case PhaseH3Negotiation:
		return "PhaseH3Negotiation"
	case PhaseQUICFallback:
		return "PhaseQUICFallback"
	default:
		return fmt.Sprintf("Phase(%d)", int32(p))
	}
}

type HttpVersion int32

const (
	HttpVersionUnknown HttpVersion = 0
	HttpVersion10      HttpVersion = 10
	HttpVersion11      HttpVersion = 11
	HttpVersion2       HttpVersion = 20
	HttpVersion3       HttpVersion = 30
)

func (v HttpVersion) String() string {
	switch v {
	case HttpVersionUnknown:
		return "HttpVersionUnknown"
	case HttpVersion10:
		return "HttpVersion10"
	case HttpVersion11:
		return "HttpVersion11"
	case HttpVersion2:
		return "HttpVersion2"
	case HttpVersion3:
		return "HttpVersion3"
	default:
		return fmt.Sprintf("HttpVersion(%d)", int32(v))
	}
}

type OpState int32

const (
	OpInProgress   OpState = 0
	OpCompletedOK  OpState = 1
	OpCompletedErr OpState = 2
	OpCancelled    OpState = 3
	OpConsumed     OpState = 4
)

func (s OpState) String() string {
	switch s {
	case OpInProgress:
		return "OpInProgress"
	case OpCompletedOK:
		return "OpCompletedOK"
	case OpCompletedErr:
		return "OpCompletedErr"
	case OpCancelled:
		return "OpCancelled"
	case OpConsumed:
		return "OpConsumed"
	default:
		return fmt.Sprintf("OpState(%d)", int32(s))
	}
}

type AcceptEncoding uint8

const (
	AcceptEncodingGzip    AcceptEncoding = 0x01
	AcceptEncodingBr      AcceptEncoding = 0x02
	AcceptEncodingDeflate AcceptEncoding = 0x04
	AcceptEncodingZstd    AcceptEncoding = 0x08
)

func (e AcceptEncoding) String() string {
	if e == 0 {
		return "AcceptEncoding(0)"
	}

	var parts []string
	if e&AcceptEncodingGzip != 0 {
		parts = append(parts, "AcceptEncodingGzip")
	}
	if e&AcceptEncodingBr != 0 {
		parts = append(parts, "AcceptEncodingBr")
	}
	if e&AcceptEncodingDeflate != 0 {
		parts = append(parts, "AcceptEncodingDeflate")
	}
	if e&AcceptEncodingZstd != 0 {
		parts = append(parts, "AcceptEncodingZstd")
	}

	known := AcceptEncodingGzip | AcceptEncodingBr | AcceptEncodingDeflate | AcceptEncodingZstd
	if extra := e &^ known; extra != 0 {
		parts = append(parts, fmt.Sprintf("AcceptEncoding(0x%02x)", uint8(extra)))
	}

	return strings.Join(parts, "|")
}

type DnsConfig int32

const (
	DnsSystem          DnsConfig = 0
	DnsGoogle          DnsConfig = 1
	DnsGoogleHTTPS     DnsConfig = 2
	DnsCloudflare      DnsConfig = 3
	DnsCloudflareHTTPS DnsConfig = 4
	DnsQuad9           DnsConfig = 5
	DnsQuad9HTTPS      DnsConfig = 6
)

func (d DnsConfig) String() string {
	switch d {
	case DnsSystem:
		return "DnsSystem"
	case DnsGoogle:
		return "DnsGoogle"
	case DnsGoogleHTTPS:
		return "DnsGoogleHTTPS"
	case DnsCloudflare:
		return "DnsCloudflare"
	case DnsCloudflareHTTPS:
		return "DnsCloudflareHTTPS"
	case DnsQuad9:
		return "DnsQuad9"
	case DnsQuad9HTTPS:
		return "DnsQuad9HTTPS"
	default:
		return fmt.Sprintf("DnsConfig(%d)", int32(d))
	}
}

type RotationStrategy int32

const (
	RotationRoundRobin RotationStrategy = 0
	RotationRandom     RotationStrategy = 1
)

func (r RotationStrategy) String() string {
	switch r {
	case RotationRoundRobin:
		return "RotationRoundRobin"
	case RotationRandom:
		return "RotationRandom"
	default:
		return fmt.Sprintf("RotationStrategy(%d)", int32(r))
	}
}

// RandomizeMode selects the fingerprint randomization tier passed to
// ClientBuilder.SetRandomize. Off and ExtensionOrder are always available;
// Recombine and Full require a library built with the synthetic-fp feature,
// otherwise SetRandomize returns an error.
type RandomizeMode int32

const (
	// RandomizeOff (Tier 0) emits the preset's fingerprint as-is.
	RandomizeOff RandomizeMode = 0
	// RandomizeExtensionOrder (Tier 1) forces per-connection TLS
	// extension-order permutation, drifting JA3 while staying a real browser.
	RandomizeExtensionOrder RandomizeMode = 1
	// RandomizeRecombine (Tier 3a) synthesizes a per-session identity by
	// recombining the real-preset corpus across the selected layers.
	RandomizeRecombine RandomizeMode = 2
	// RandomizeFull (Tier 3b) is RandomizeRecombine plus out-of-corpus H2/QUIC
	// values on the advertise-only layers.
	RandomizeFull RandomizeMode = 3
)

func (m RandomizeMode) String() string {
	switch m {
	case RandomizeOff:
		return "RandomizeOff"
	case RandomizeExtensionOrder:
		return "RandomizeExtensionOrder"
	case RandomizeRecombine:
		return "RandomizeRecombine"
	case RandomizeFull:
		return "RandomizeFull"
	default:
		return fmt.Sprintf("RandomizeMode(%d)", int32(m))
	}
}

// FingerprintLayers is a bitmask selecting which fingerprint layers a synthetic
// randomization mode (RandomizeRecombine/RandomizeFull) varies. OR the bits
// together; the zero value (FingerprintLayersAll) means "all layers", the safe
// default. Ignored for RandomizeOff and RandomizeExtensionOrder.
type FingerprintLayers uint32

const (
	// FingerprintLayersAll (the zero value) synthesizes every layer — the safe
	// "no layer left constant" default.
	FingerprintLayersAll FingerprintLayers = 0
	// LayerTLS selects the TLS ClientHello (JA3/JA4); also drives QUIC-native TLS.
	LayerTLS FingerprintLayers = 1
	// LayerH2 selects the HTTP/2 fingerprint (SETTINGS + pseudo-header order).
	LayerH2 FingerprintLayers = 2
	// LayerQUIC selects the QUIC transport params + HTTP/3 fingerprint (needs quic-h3).
	LayerQUIC FingerprintLayers = 4
)

// CookieAttrs controls optional attributes when inserting a cookie into a session jar.
// Empty Path or Domain values are omitted.
type CookieAttrs struct {
	Path     string
	Domain   string
	Secure   bool
	HTTPOnly bool
}

// Cookie is a name/value pair, either from a response Set-Cookie header or
// from a session cookie jar.
//
// This is deliberately not net/http.Cookie: the FFI layer only exposes the
// name and the value, so every other attribute field would be permanently
// zero and read as "not set" rather than "not reported by the library". The
// JSON tags match the payload of lk_session_get_cookies_json.
type Cookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Redirect is one hop of a response redirect chain.
type Redirect struct {
	URL    string
	Status int
}

// PreferredHTTPVersion selects the per-request protocol preference applied by
// Request.SetPreferredHTTPVersion. It mirrors lk_preferred_http_version_t.
type PreferredHTTPVersion int32

const (
	PreferredHTTPVersionAuto              PreferredHTTPVersion = 0
	PreferredHTTPVersionHTTP1Only         PreferredHTTPVersion = 10
	PreferredHTTPVersionHTTP2Only         PreferredHTTPVersion = 20
	PreferredHTTPVersionHTTP3Only         PreferredHTTPVersion = 30
	PreferredHTTPVersionHTTP3WithFallback PreferredHTTPVersion = 31
)

func (v PreferredHTTPVersion) String() string {
	switch v {
	case PreferredHTTPVersionAuto:
		return "PreferredHTTPVersionAuto"
	case PreferredHTTPVersionHTTP1Only:
		return "PreferredHTTPVersionHTTP1Only"
	case PreferredHTTPVersionHTTP2Only:
		return "PreferredHTTPVersionHTTP2Only"
	case PreferredHTTPVersionHTTP3Only:
		return "PreferredHTTPVersionHTTP3Only"
	case PreferredHTTPVersionHTTP3WithFallback:
		return "PreferredHTTPVersionHTTP3WithFallback"
	default:
		return fmt.Sprintf("PreferredHTTPVersion(%d)", int32(v))
	}
}

// Socks5UDPProbeMode selects how Client.Socks5UDPProbe exercises a SOCKS5 proxy.
// It mirrors lk_socks5_udp_probe_mode_t.
type Socks5UDPProbeMode int32

const (
	// Socks5UDPProbeAssociateOnly performs only the UDP ASSOCIATE handshake.
	Socks5UDPProbeAssociateOnly Socks5UDPProbeMode = 0
	// Socks5UDPProbeDNSRoundTrip additionally sends a DNS query through the relay.
	Socks5UDPProbeDNSRoundTrip Socks5UDPProbeMode = 1
)

func (m Socks5UDPProbeMode) String() string {
	switch m {
	case Socks5UDPProbeAssociateOnly:
		return "Socks5UDPProbeAssociateOnly"
	case Socks5UDPProbeDNSRoundTrip:
		return "Socks5UDPProbeDNSRoundTrip"
	default:
		return fmt.Sprintf("Socks5UDPProbeMode(%d)", int32(m))
	}
}

// Socks5UDPProbePhase reports how far a SOCKS5 UDP probe progressed.
// It mirrors lk_socks5_udp_probe_phase_t.
type Socks5UDPProbePhase int32

const (
	Socks5UDPProbePhaseNotSocks5    Socks5UDPProbePhase = 0
	Socks5UDPProbePhaseUDPAssociate Socks5UDPProbePhase = 1
	Socks5UDPProbePhaseUDPRoundTrip Socks5UDPProbePhase = 2
)

func (p Socks5UDPProbePhase) String() string {
	switch p {
	case Socks5UDPProbePhaseNotSocks5:
		return "Socks5UDPProbePhaseNotSocks5"
	case Socks5UDPProbePhaseUDPAssociate:
		return "Socks5UDPProbePhaseUDPAssociate"
	case Socks5UDPProbePhaseUDPRoundTrip:
		return "Socks5UDPProbePhaseUDPRoundTrip"
	default:
		return fmt.Sprintf("Socks5UDPProbePhase(%d)", int32(p))
	}
}

// Socks5UDPProbeSupport summarizes the UDP relay support detected by a probe.
// It mirrors lk_socks5_udp_probe_support_t.
type Socks5UDPProbeSupport int32

const (
	Socks5UDPSupportNotSocks5   Socks5UDPProbeSupport = 0
	Socks5UDPSupportAssociateOK Socks5UDPProbeSupport = 1
	Socks5UDPSupportRelayOK     Socks5UDPProbeSupport = 2
	Socks5UDPSupportUnsupported Socks5UDPProbeSupport = 3
	Socks5UDPSupportFailed      Socks5UDPProbeSupport = 4
)

func (s Socks5UDPProbeSupport) String() string {
	switch s {
	case Socks5UDPSupportNotSocks5:
		return "Socks5UDPSupportNotSocks5"
	case Socks5UDPSupportAssociateOK:
		return "Socks5UDPSupportAssociateOK"
	case Socks5UDPSupportRelayOK:
		return "Socks5UDPSupportRelayOK"
	case Socks5UDPSupportUnsupported:
		return "Socks5UDPSupportUnsupported"
	case Socks5UDPSupportFailed:
		return "Socks5UDPSupportFailed"
	default:
		return fmt.Sprintf("Socks5UDPProbeSupport(%d)", int32(s))
	}
}
