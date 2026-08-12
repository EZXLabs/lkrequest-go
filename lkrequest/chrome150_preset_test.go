package lkrequest

import "testing"

// TestChrome150PresetAvailable verifies the chrome_150 fingerprint preset is
// usable through the Go binding. Chrome 150 adds ML-DSA signature algorithms
// over Chrome 149 (TLS/H2 only upstream; H3 pending).
//
// Presets are resolved by name inside the loaded library, so there is no
// binding code to add for a new preset — but exercising it needs a library
// that actually ships chrome_150. Against an older embedded library that
// predates the preset, SetPreset(...).Build() fails with a "set preset failed"
// status, so the test skips instead of failing. It activates automatically
// once a library carrying chrome_150 is embedded.
func TestChrome150PresetAvailable(t *testing.T) {
	const preset = "chrome_150"

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
