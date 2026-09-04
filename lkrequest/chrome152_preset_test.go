package lkrequest

import (
	"encoding/json"
	"testing"
)

// Extension type numbers that distinguish Chrome 152 from Chrome 151.
const (
	// extTrustAnchorIds is Chromium/BoringSSL's experimental trust-anchor
	// identifiers extension (0xca34).
	extTrustAnchorIds = 51764
	// quicParamGoogleInitialRtt is the QUIC transport parameter (0x3127)
	// Chrome 152 stopped sending.
	quicParamGoogleInitialRtt = 12583
)

// fingerprintInfo is the subset of ClientFingerprintInfoJSON these tests assert
// on. Everything else in the payload is left as raw JSON on purpose: the
// library owns that schema, and pinning fields we do not check here would turn
// every unrelated upstream addition into a Go-side failure.
type fingerprintInfo struct {
	TLS struct {
		Name   string `json:"name"`
		Grease struct {
			SignatureAlgorithms bool `json:"signature_algorithms"`
		} `json:"grease"`
		Extensions []struct {
			ExtensionType int `json:"extension_type"`
			Source        struct {
				Type    string   `json:"type"`
				IDs     []string `json:"ids"`
				Shuffle bool     `json:"shuffle"`
			} `json:"source"`
		} `json:"extensions"`
	} `json:"tls"`
	Quic *struct {
		TransportParams struct {
			Order []int64             `json:"transport_parameter_order"`
			Extra [][]json.RawMessage `json:"extra_transport_parameters"`
		} `json:"transport_params"`
	} `json:"quic"`
}

// chrome152Info builds a chrome_152 client and decodes the fingerprint it
// reports. Presets are resolved by name inside the loaded library, so a library
// predating chrome_152 fails at Build() — that is a skip, not a failure, the
// same way chrome151_preset_test.go handles it.
func chrome152Info(t *testing.T) fingerprintInfo {
	t.Helper()

	client, err := NewClientBuilder().SetPreset("chrome_152").Build()
	if err != nil {
		t.Skipf("chrome_152 preset not available in the loaded lkrequest library: %v", err)
	}
	t.Cleanup(client.Close)

	raw, err := client.FingerprintInfoJSON()
	if err != nil {
		t.Fatalf("FingerprintInfoJSON with chrome_152 preset error = %v", err)
	}

	var info fingerprintInfo
	if err := json.Unmarshal([]byte(raw), &info); err != nil {
		t.Fatalf("unmarshal chrome_152 fingerprint info error = %v; payload = %s", err, raw)
	}
	return info
}

// TestChrome152PresetAvailable verifies chrome_152 is usable end to end through
// the Go binding: the preset resolves, and a session builds on top of it.
// Offline — nothing here opens a connection.
func TestChrome152PresetAvailable(t *testing.T) {
	client, err := NewClientBuilder().SetPreset("chrome_152").Build()
	if err != nil {
		t.Skipf("chrome_152 preset not available in the loaded lkrequest library: %v", err)
	}
	t.Cleanup(client.Close)

	session, err := NewSessionBuilder(client).Build()
	if err != nil {
		t.Fatalf("NewSessionBuilder with chrome_152 preset error = %v", err)
	}
	t.Cleanup(session.Close)

	if got := clientPresetName(t, client); got != "Chrome 152" {
		t.Fatalf("fingerprint name = %q, want %q", got, "Chrome 152")
	}
}

// TestChrome152SignatureAlgorithmsGrease pins the fix that shipped alongside
// the preset: Chrome 152 prepends an independently randomized GREASE code point
// to signature_algorithms. The initial preset omitted it, which left
// peetprint_hash diverging even though JA3/JA3N, JA4 and the H2 Akamai
// fingerprint all matched — precisely the kind of gap a smoke test misses.
// Chrome 151 has this flag off, so it also guards against the preset silently
// falling back to the 151 profile.
func TestChrome152SignatureAlgorithmsGrease(t *testing.T) {
	info := chrome152Info(t)

	if !info.TLS.Grease.SignatureAlgorithms {
		t.Fatal("chrome_152 grease.signature_algorithms = false, want true")
	}
}

// TestChrome152TrustAnchorIds checks the other TCP-TLS addition: Chromium's
// experimental trust-anchor identifiers vector (0xca34), whose order is
// reshuffled per ClientHello.
func TestChrome152TrustAnchorIds(t *testing.T) {
	info := chrome152Info(t)

	for _, ext := range info.TLS.Extensions {
		if ext.ExtensionType != extTrustAnchorIds {
			continue
		}
		if ext.Source.Type != "trust_anchor_ids" {
			t.Fatalf("extension 0x%04x source type = %q, want %q",
				extTrustAnchorIds, ext.Source.Type, "trust_anchor_ids")
		}
		if len(ext.Source.IDs) == 0 {
			t.Fatalf("extension 0x%04x carries no trust-anchor identifiers", extTrustAnchorIds)
		}
		if !ext.Source.Shuffle {
			t.Fatalf("extension 0x%04x shuffle = false, want true", extTrustAnchorIds)
		}
		return
	}

	t.Fatalf("chrome_152 does not advertise extension 0x%04x; extensions = %+v",
		extTrustAnchorIds, info.TLS.Extensions)
}

// TestChrome152QUICDropsGoogleInitialRtt covers where Chrome 152 diverges from
// Chrome 151 on the QUIC side: the google_initial_rtt transport parameter
// (0x3127) is no longer sent. The rest of the QUIC baseline is inherited from
// 151, so this parameter is the whole difference and worth pinning on its own.
// Skipped when the loaded library was built without quic-h3.
func TestChrome152QUICDropsGoogleInitialRtt(t *testing.T) {
	if !FeatureSupported("quic-h3") {
		t.Skip("quic-h3 is not supported by the loaded lkrequest library")
	}

	info := chrome152Info(t)
	if info.Quic == nil {
		t.Fatal("chrome_152 reports no QUIC profile")
	}

	for _, id := range info.Quic.TransportParams.Order {
		if id == quicParamGoogleInitialRtt {
			t.Fatalf("chrome_152 still orders QUIC transport parameter 0x%04x (google_initial_rtt)",
				quicParamGoogleInitialRtt)
		}
	}

	// extra_transport_parameters entries are [id, hex_payload] pairs; only the
	// id matters here.
	for _, pair := range info.Quic.TransportParams.Extra {
		if len(pair) == 0 {
			continue
		}
		var id int64
		if err := json.Unmarshal(pair[0], &id); err != nil {
			continue
		}
		if id == quicParamGoogleInitialRtt {
			t.Fatalf("chrome_152 still sends QUIC transport parameter 0x%04x (google_initial_rtt)",
				quicParamGoogleInitialRtt)
		}
	}
}

// clientPresetName reports the fingerprint name the library attributes to the
// client, used to confirm a preset resolved to the profile we asked for.
func clientPresetName(t *testing.T, client *Client) string {
	t.Helper()

	raw, err := client.FingerprintInfoJSON()
	if err != nil {
		t.Fatalf("FingerprintInfoJSON() error = %v", err)
	}

	var info fingerprintInfo
	if err := json.Unmarshal([]byte(raw), &info); err != nil {
		t.Fatalf("unmarshal fingerprint info error = %v; payload = %s", err, raw)
	}
	return info.TLS.Name
}
