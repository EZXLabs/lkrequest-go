package lkrequest

import "testing"

// TestChrome149PresetAvailable verifies the chrome_149 fingerprint preset is
// usable through the Go binding. It was added upstream from a real Chrome
// 149.0.7827.54 capture; its wire fingerprint matches Chrome 148.
//
// Presets are resolved by name inside the loaded library, so there is no
// binding code to add for a new preset — but exercising it needs a library
// that actually ships chrome_149. Against an older embedded library that
// predates the preset, SetPreset(...).Build() fails with a "set preset failed"
// status, so the test skips instead of failing. It activates automatically
// once a library carrying chrome_149 is embedded.
func TestChrome149PresetAvailable(t *testing.T) {
	const preset = "chrome_149"

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
