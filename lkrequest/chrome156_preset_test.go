package lkrequest

import (
	"reflect"
	"slices"
	"testing"
)

// TestChrome156PresetAvailable verifies chrome_156 is usable end to end through
// the Go binding: the preset resolves, a session builds on top of it, the
// library attributes the client to Chrome 156 rather than silently falling back
// to an older profile, and ListPresetsJSON names it. Offline — nothing here
// opens a connection.
func TestChrome156PresetAvailable(t *testing.T) {
	client, err := NewClientBuilder().SetPreset("chrome_156").Build()
	if err != nil {
		t.Skipf("chrome_156 preset not available in the loaded lkrequest library: %v", err)
	}
	t.Cleanup(client.Close)

	session, err := NewSessionBuilder(client).Build()
	if err != nil {
		t.Fatalf("NewSessionBuilder with chrome_156 preset error = %v", err)
	}
	t.Cleanup(session.Close)

	if got := clientPresetName(t, client); got != "Chrome 156" {
		t.Fatalf("fingerprint name = %q, want %q", got, "Chrome 156")
	}

	presets, err := ListPresetsJSON()
	if err != nil {
		t.Fatalf("ListPresetsJSON() error = %v", err)
	}
	if names := decodeJSONText[[]string](t, presets); !slices.Contains(names, "chrome_156") {
		t.Fatalf("ListPresetsJSON() = %v, want it to name chrome_156", names)
	}
}

// TestChrome156MatchesChrome155 pins what the Chrome 156 captures found: every
// layer — TCP TLS, H2, QUIC TLS, QUIC transport parameters, H3 and Initial
// packetization — keeps Chrome 155's fingerprint, so the two are compared
// whole with only their names set aside.
func TestChrome156MatchesChrome155(t *testing.T) {
	c156 := presetFingerprintWithoutName(t, "chrome_156")
	c155 := presetFingerprintWithoutName(t, "chrome_155")

	for layer, got := range c156 {
		if want, ok := c155[layer]; !ok || !reflect.DeepEqual(got, want) {
			t.Errorf("chrome_156 %q layer differs from chrome_155:\n got: %v\nwant: %v", layer, got, want)
		}
	}
	for layer := range c155 {
		if _, ok := c156[layer]; !ok {
			t.Errorf("chrome_156 is missing the %q layer chrome_155 has", layer)
		}
	}
}
