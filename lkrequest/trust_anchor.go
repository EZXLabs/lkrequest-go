package lkrequest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"runtime"
)

// TrustAnchorAdvertisement selects the trust anchor identifiers a client
// advertises in the TLS trust_anchors extension (0xca34) of Trust Anchor
// Identifiers (TAI, draft-ietf-tls-trust-anchor-ids).
type TrustAnchorAdvertisement int32

const (
	// TrustAnchorAdvertiseProfile sends the identifiers the TLS fingerprint
	// lists, in its order, as the preset does by itself. The default.
	TrustAnchorAdvertiseProfile TrustAnchorAdvertisement = 0
	// TrustAnchorAdvertiseTrusted sends instead the sorted identifiers bound
	// to roots the client trusts: the bundled ones of the built-in roots and
	// those added with AddTrustAnchorBinding.
	TrustAnchorAdvertiseTrusted TrustAnchorAdvertisement = 1
)

// TrustAnchorRecovery selects whether a request may reconnect when its TLS
// handshake fails after the server listed the trust anchors it can serve.
type TrustAnchorRecovery int32

const (
	// TrustAnchorRecoveryDisabled never reconnects. The default.
	TrustAnchorRecoveryDisabled TrustAnchorRecovery = 0
	// TrustAnchorRecoveryOnce lets a request make one new connection when a
	// handshake fails on an unknown issuer, or on the server closing or
	// alerting, after the server listed the identifiers it can serve. The new
	// connection asks for the first of them this client trusts. If it fails
	// too, the request fails with an error that is not retryable.
	TrustAnchorRecoveryOnce TrustAnchorRecovery = 1
)

// TrustAnchorPolicy configures Trust Anchor Identifiers negotiation. The zero
// value is the default: the profile's identifiers and no recovery.
type TrustAnchorPolicy struct {
	Advertisement TrustAnchorAdvertisement
	Recovery      TrustAnchorRecovery
}

// trustAnchorIdentityLen is the length of an anchor identity: a SHA-256
// digest in hexadecimal.
const trustAnchorIdentityLen = 64

// SetTrustAnchorPolicy configures Trust Anchor Identifiers negotiation on TLS
// 1.3 connections, over TCP and, with a QUIC/H3-capable library, QUIC. It
// never changes which certificates are trusted. Build reports a policy it
// cannot honour: recovery needs certificate verification and cannot combine
// with SetFallbackProxyToDirect, and any policy but the zero value needs every
// TLS profile of the preset to carry the trust_anchors extension (Chrome's
// do, Firefox's and Safari's do not) and rules out synthetic TLS
// randomization. Requires a library exporting
// lk_client_builder_set_trust_anchor_policy_json.
func (b *ClientBuilder) SetTrustAnchorPolicy(policy TrustAnchorPolicy) *ClientBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_client_builder_set_trust_anchor_policy_json == nil {
		return b.setUnsupported("client builder set trust anchor policy")
	}

	payload, err := trustAnchorPolicyJSON(policy)
	if err != nil {
		b.firstErr = err
		return b
	}
	status := ffi_lk_client_builder_set_trust_anchor_policy_json(b.ptr, bytesPtr(payload), uintptr(len(payload)))
	runtime.KeepAlive(payload)
	return b.setStatus(status, "client builder set trust anchor policy")
}

// AddTrustAnchorBinding binds trust anchor identifier id, in hexadecimal, to
// a root the client trusts — a built-in root, or one added with AddCACertFile
// or AddCACertMemory — named by its TrustAnchorIdentity. The binding is what
// TrustAnchorAdvertiseTrusted and recovery know the root by; it adds no
// trust. Obtain the identifier from the CA or the deployment that assigned
// it; it is not derived from the certificate. Build reports a malformed
// identifier, a root the client does not trust, and an identifier bound to
// two different roots. Requires a library exporting
// lk_client_builder_add_trust_anchor_binding_json.
func (b *ClientBuilder) AddTrustAnchorBinding(anchorIdentity, id string) *ClientBuilder {
	if !b.check() {
		return b
	}
	if ffi_lk_client_builder_add_trust_anchor_binding_json == nil {
		return b.setUnsupported("client builder add trust anchor binding")
	}

	payload, err := json.Marshal(struct {
		AnchorIdentity string `json:"anchor_identity"`
		ID             string `json:"id"`
	}{anchorIdentity, id})
	if err != nil {
		b.firstErr = err
		return b
	}
	status := ffi_lk_client_builder_add_trust_anchor_binding_json(b.ptr, bytesPtr(payload), uintptr(len(payload)))
	runtime.KeepAlive(payload)
	return b.setStatus(status, "client builder add trust anchor binding")
}

// TrustAnchorIdentity returns the identity the library gives the root
// certificate certDER (DER encoded): a SHA-256 digest over its subject, public
// key and name constraints, as 64 lowercase hexadecimal characters. It names
// the root in AddTrustAnchorBinding, and as "verified_anchor" in the
// "trust_anchor" object of DiagnosticsJSON. Computing it adds no trust.
// Requires a library exporting lk_trust_anchor_identity_from_der.
func TrustAnchorIdentity(certDER []byte) (string, error) {
	if ffi_lk_trust_anchor_identity_from_der == nil {
		return "", unsupportedError("trust anchor identity")
	}

	// The library writes the identity and a terminating NUL.
	out := make([]byte, trustAnchorIdentityLen+1)
	status := ffi_lk_trust_anchor_identity_from_der(
		bytesPtr(certDER),
		uintptr(len(certDER)),
		bytesPtr(out),
		uintptr(len(out)),
	)
	runtime.KeepAlive(certDER)
	runtime.KeepAlive(out)
	if err := statusError(status, "trust anchor identity"); err != nil {
		return "", err
	}
	if n := bytes.IndexByte(out, 0); n >= 0 {
		out = out[:n]
	}
	return string(out), nil
}

// trustAnchorPolicyJSON encodes policy in the form the library parses. Values
// outside the declared constants have no encoding and are an error.
func trustAnchorPolicyJSON(policy TrustAnchorPolicy) ([]byte, error) {
	var payload struct {
		Advertisement string `json:"advertisement"`
		Recovery      string `json:"recovery"`
	}
	switch policy.Advertisement {
	case TrustAnchorAdvertiseProfile:
		payload.Advertisement = "profile"
	case TrustAnchorAdvertiseTrusted:
		payload.Advertisement = "trusted"
	default:
		return nil, fmt.Errorf("lk: invalid trust anchor advertisement %d", policy.Advertisement)
	}
	switch policy.Recovery {
	case TrustAnchorRecoveryDisabled:
		payload.Recovery = "disabled"
	case TrustAnchorRecoveryOnce:
		payload.Recovery = "once"
	default:
		return nil, fmt.Errorf("lk: invalid trust anchor recovery %d", policy.Recovery)
	}
	return json.Marshal(payload)
}
