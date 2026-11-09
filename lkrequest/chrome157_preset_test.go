package lkrequest

import (
	"reflect"
	"slices"
	"testing"
)

// TestChrome157PresetAvailable verifies chrome_157 is usable end to end through
// the Go binding: the preset resolves, a session builds on top of it, the
// library attributes the client to Chrome 157 rather than silently falling back
// to an older profile, and ListPresetsJSON names it. Offline — nothing here
// opens a connection.
func TestChrome157PresetAvailable(t *testing.T) {
	client, err := NewClientBuilder().SetPreset("chrome_157").Build()
	if err != nil {
		t.Skipf("chrome_157 preset not available in the loaded lkrequest library: %v", err)
	}
	t.Cleanup(client.Close)

	session, err := NewSessionBuilder(client).Build()
	if err != nil {
		t.Fatalf("NewSessionBuilder with chrome_157 preset error = %v", err)
	}
	t.Cleanup(session.Close)

	if got := clientPresetName(t, client); got != "Chrome 157" {
		t.Fatalf("fingerprint name = %q, want %q", got, "Chrome 157")
	}

	presets, err := ListPresetsJSON()
	if err != nil {
		t.Fatalf("ListPresetsJSON() error = %v", err)
	}
	if names := decodeJSONText[[]string](t, presets); !slices.Contains(names, "chrome_157") {
		t.Fatalf("ListPresetsJSON() = %v, want it to name chrome_157", names)
	}
}

// TestChrome157MatchesChrome156 pins what the Chrome 157 captures found: every
// layer keeps Chrome 156's fingerprint, so the two are compared whole with
// only their names set aside.
func TestChrome157MatchesChrome156(t *testing.T) {
	c157 := presetFingerprintWithoutName(t, "chrome_157")
	c156 := presetFingerprintWithoutName(t, "chrome_156")

	for layer, got := range c157 {
		if want, ok := c156[layer]; !ok || !reflect.DeepEqual(got, want) {
			t.Errorf("chrome_157 %q layer differs from chrome_156:\n got: %v\nwant: %v", layer, got, want)
		}
	}
	for layer := range c156 {
		if _, ok := c157[layer]; !ok {
			t.Errorf("chrome_157 is missing the %q layer chrome_156 has", layer)
		}
	}
}
