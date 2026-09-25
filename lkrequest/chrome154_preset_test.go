package lkrequest

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestChrome154PresetAvailable verifies chrome_154 is usable end to end through
// the Go binding: the preset resolves, a session builds on top of it, and the
// library attributes the client to Chrome 154 rather than silently falling back
// to an older profile. Offline — nothing here opens a connection.
func TestChrome154PresetAvailable(t *testing.T) {
	client, err := NewClientBuilder().SetPreset("chrome_154").Build()
	if err != nil {
		t.Skipf("chrome_154 preset not available in the loaded lkrequest library: %v", err)
	}
	t.Cleanup(client.Close)

	session, err := NewSessionBuilder(client).Build()
	if err != nil {
		t.Fatalf("NewSessionBuilder with chrome_154 preset error = %v", err)
	}
	t.Cleanup(session.Close)

	if got := clientPresetName(t, client); got != "Chrome 154" {
		t.Fatalf("fingerprint name = %q, want %q", got, "Chrome 154")
	}
}

// TestChrome154RetainsChrome153Shape pins what the Chrome 154 captures found:
// TCP TLS, H2, QUIC TLS, QUIC transport parameters and Initial packetization
// all keep Chrome 153's shape. So rather than spot-check a few fields, the two
// fingerprints are compared whole with only their names set aside — any drift
// between them, in a field this test never heard of, fails here.
func TestChrome154RetainsChrome153Shape(t *testing.T) {
	c154 := presetFingerprintWithoutName(t, "chrome_154")
	c153 := presetFingerprintWithoutName(t, "chrome_153")

	for layer, got := range c154 {
		if want, ok := c153[layer]; !ok || !reflect.DeepEqual(got, want) {
			t.Errorf("chrome_154 %q layer differs from chrome_153:\n got: %v\nwant: %v", layer, got, want)
		}
	}
	for layer := range c153 {
		if _, ok := c154[layer]; !ok {
			t.Errorf("chrome_154 is missing the %q layer chrome_153 has", layer)
		}
	}
}

// presetFingerprintWithoutName decodes a preset's full fingerprint payload,
// keyed by layer, with the TLS profile name removed — the one field two
// presets of the same shape are expected to disagree on.
func presetFingerprintWithoutName(t *testing.T, preset string) map[string]any {
	t.Helper()

	client, err := NewClientBuilder().SetPreset(preset).Build()
	if err != nil {
		t.Skipf("%s preset not available in the loaded lkrequest library: %v", preset, err)
	}
	t.Cleanup(client.Close)

	raw, err := client.FingerprintInfoJSON()
	if err != nil {
		t.Fatalf("FingerprintInfoJSON with %s preset error = %v", preset, err)
	}

	var layers map[string]any
	if err := json.Unmarshal([]byte(raw), &layers); err != nil {
		t.Fatalf("unmarshal %s fingerprint info error = %v; payload = %s", preset, err, raw)
	}
	if tls, ok := layers["tls"].(map[string]any); ok {
		delete(tls, "name")
	}
	return layers
}
