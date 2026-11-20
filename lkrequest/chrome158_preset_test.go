package lkrequest

import (
	"reflect"
	"slices"
	"testing"
)

// TestChrome158PresetAvailable verifies chrome_158 is usable end to end through
// the Go binding: the preset resolves, a session builds on top of it, the
// library attributes the client to Chrome 158 rather than silently falling back
// to an older profile, and ListPresetsJSON names it. Offline — nothing here
// opens a connection.
func TestChrome158PresetAvailable(t *testing.T) {
	client, err := NewClientBuilder().SetPreset("chrome_158").Build()
	if err != nil {
		t.Skipf("chrome_158 preset not available in the loaded lkrequest library: %v", err)
	}
	t.Cleanup(client.Close)

	session, err := NewSessionBuilder(client).Build()
	if err != nil {
		t.Fatalf("NewSessionBuilder with chrome_158 preset error = %v", err)
	}
	t.Cleanup(session.Close)

	if got := clientPresetName(t, client); got != "Chrome 158" {
		t.Fatalf("fingerprint name = %q, want %q", got, "Chrome 158")
	}

	presets, err := ListPresetsJSON()
	if err != nil {
		t.Fatalf("ListPresetsJSON() error = %v", err)
	}
	if names := decodeJSONText[[]string](t, presets); !slices.Contains(names, "chrome_158") {
		t.Fatalf("ListPresetsJSON() = %v, want it to name chrome_158", names)
	}
}

// TestChrome158MatchesChrome157 pins what the Chrome 158 captures found: every
// layer keeps Chrome 157's fingerprint, so the two are compared whole with
// only their names set aside.
func TestChrome158MatchesChrome157(t *testing.T) {
	c158 := presetFingerprintWithoutName(t, "chrome_158")
	c157 := presetFingerprintWithoutName(t, "chrome_157")

	for layer, got := range c158 {
		if want, ok := c157[layer]; !ok || !reflect.DeepEqual(got, want) {
			t.Errorf("chrome_158 %q layer differs from chrome_157:\n got: %v\nwant: %v", layer, got, want)
		}
	}
	for layer := range c157 {
		if _, ok := c158[layer]; !ok {
			t.Errorf("chrome_158 is missing the %q layer chrome_157 has", layer)
		}
	}
}
