package lkrequest

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestChrome151PresetAvailable verifies the chrome_151 fingerprint preset is
// usable through the Go binding. Chrome 151 keeps Chrome 150's stable TCP
// TLS/H2 fields; the difference lives in the QUIC layer, where real captures
// drop the three ML-DSA signature algorithms and keep the classic list plus
// rsa_pkcs1_sha1, with QUIC transport params and H3 SETTINGS still aligned
// with Chrome 150.
//
// Presets are resolved by name inside the loaded library, so there is no
// binding code to add for a new preset — but exercising it needs a library
// that actually ships chrome_151. Against an older embedded library that
// predates the preset, SetPreset(...).Build() fails with a "set preset failed"
// status, so the test skips instead of failing. It activates automatically
// once a library carrying chrome_151 is embedded.
func TestChrome151PresetAvailable(t *testing.T) {
	const preset = "chrome_151"

	client, err := NewClientBuilder().SetPreset(preset).Build()
	if err != nil {
		t.Skipf("%s preset not available in the loaded lkrequest library: %v", preset, err)
	}
	t.Cleanup(client.Close)

	// The preset is recognized by the library; confirm the rest of the
	// request-construction path accepts a client built from it (offline).
	session, err := NewSessionBuilder(client).Build()
	if err != nil {
		t.Fatalf("NewSessionBuilder with %s preset error = %v", preset, err)
	}
	t.Cleanup(session.Close)
}

// TestChrome151QUICProfile checks that chrome_151 carries a QUIC/H3 profile,
// which is where it actually diverges from chrome_150. Skipped when the loaded
// library lacks quic-h3 support or predates the preset. Fully offline: it only
// inspects the fingerprint description the library reports for the client.
func TestChrome151QUICProfile(t *testing.T) {
	const preset = "chrome_151"

	if !FeatureSupported("quic-h3") {
		t.Skip("quic-h3 is not supported by the loaded lkrequest library")
	}

	client, err := NewClientBuilder().SetPreset(preset).Build()
	if err != nil {
		t.Skipf("%s preset not available in the loaded lkrequest library: %v", preset, err)
	}
	t.Cleanup(client.Close)

	info, err := client.FingerprintInfoJSON()
	if err != nil {
		t.Fatalf("FingerprintInfoJSON with %s preset error = %v", preset, err)
	}

	var payload struct {
		Quic json.RawMessage `json:"quic"`
	}
	if err := json.Unmarshal([]byte(info), &payload); err != nil {
		t.Fatalf("unmarshal fingerprint info for %s error = %v", preset, err)
	}

	raw := strings.TrimSpace(string(payload.Quic))
	if raw == "" || raw == "null" {
		t.Fatalf("%s reports no QUIC profile; fingerprint info = %s", preset, info)
	}
}
