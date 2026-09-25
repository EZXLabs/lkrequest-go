package lkrequest

import "testing"

// Chrome 153 sends the same 0xca34 trust-anchor identifiers extension as
// Chrome 152, with four values removed.
const chrome153TrustAnchorIDCount = 28

var chrome153DroppedTrustAnchorIDs = []string{"d6790902", "d6790903", "d6790909", "d679090e"}

// TestChrome153PresetAvailable verifies chrome_153 is usable end to end through
// the Go binding: the preset resolves, a session builds on top of it, and the
// library attributes the client to Chrome 153 rather than silently falling back
// to an older profile. Offline — nothing here opens a connection.
func TestChrome153PresetAvailable(t *testing.T) {
	client, err := NewClientBuilder().SetPreset("chrome_153").Build()
	if err != nil {
		t.Skipf("chrome_153 preset not available in the loaded lkrequest library: %v", err)
	}
	t.Cleanup(client.Close)

	session, err := NewSessionBuilder(client).Build()
	if err != nil {
		t.Fatalf("NewSessionBuilder with chrome_153 preset error = %v", err)
	}
	t.Cleanup(session.Close)

	if got := clientPresetName(t, client); got != "Chrome 153" {
		t.Fatalf("fingerprint name = %q, want %q", got, "Chrome 153")
	}
}

// TestChrome153TrustAnchorIdsReduced pins where Chrome 153 diverges from
// Chrome 152 on the TCP-TLS side: the shuffled 0xca34 vector drops from 32
// identifiers to 28. Both the count and the four dropped values are asserted,
// because a capture that gets the count right with the wrong members is still
// the wrong ClientHello. Chrome 152 keeps all 32, so this also catches the
// preset resolving to the 152 profile.
func TestChrome153TrustAnchorIdsReduced(t *testing.T) {
	info := presetFingerprint(t, "chrome_153")

	for _, ext := range info.TLS.Extensions {
		if ext.ExtensionType != extTrustAnchorIds {
			continue
		}
		if ext.Source.Type != "trust_anchor_ids" {
			t.Fatalf("extension 0x%04x source type = %q, want %q",
				extTrustAnchorIds, ext.Source.Type, "trust_anchor_ids")
		}
		if got := len(ext.Source.IDs); got != chrome153TrustAnchorIDCount {
			t.Fatalf("chrome_153 trust-anchor identifier count = %d, want %d",
				got, chrome153TrustAnchorIDCount)
		}
		if !ext.Source.Shuffle {
			t.Fatalf("extension 0x%04x shuffle = false, want true", extTrustAnchorIds)
		}

		present := make(map[string]bool, len(ext.Source.IDs))
		for _, id := range ext.Source.IDs {
			present[id] = true
		}
		for _, dropped := range chrome153DroppedTrustAnchorIDs {
			if present[dropped] {
				t.Errorf("chrome_153 still advertises trust-anchor identifier %s, which Chrome 153 dropped", dropped)
			}
		}
		return
	}

	t.Fatalf("chrome_153 does not advertise extension 0x%04x; extensions = %+v",
		extTrustAnchorIds, info.TLS.Extensions)
}

// TestChromiumQUICWireGeneration covers the QUIC/H3 generation rules that
// arrived with Chrome 153 but apply to the earlier Chrome presets too: H3 emits
// Chromium-shaped GREASE, and client Initials split, pad and shuffle their
// CRYPTO data. Both profile fields default to false, so a preset that lost them
// would fall back to the old wire image silently — nothing about the handshake
// would fail, it would just stop looking like Chrome.
//
// chrome_152 is checked alongside chrome_153 because this changed an already
// released preset; asserting only the new one would miss that regression.
// chrome_154 inherits both rules from chrome_153.
//
// chrome_152 is also why the skip here cannot key off the preset resolving:
// unlike chrome_153, it exists in libraries that predate these rules, so
// SetPreset succeeds and only the two fields are missing. Skipped when the
// loaded library was built without quic-h3, or when it reports no such fields.
func TestChromiumQUICWireGeneration(t *testing.T) {
	if !FeatureSupported("quic-h3") {
		t.Skip("quic-h3 is not supported by the loaded lkrequest library")
	}

	for _, preset := range []string{"chrome_152", "chrome_153", "chrome_154"} {
		t.Run(preset, func(t *testing.T) {
			info := presetFingerprint(t, preset)
			if info.Quic == nil {
				t.Fatalf("%s reports no QUIC profile", preset)
			}

			grease := info.Quic.H3.ChromiumGrease
			chaos := info.Quic.Packetization.InitialPacketChaos
			if grease == nil || chaos == nil {
				t.Skipf("%s: the loaded lkrequest library predates Chromium QUIC wire generation", preset)
			}
			if !*grease {
				t.Errorf("%s h3.chromium_grease = false, want true", preset)
			}
			if !*chaos {
				t.Errorf("%s packetization.initial_packet_chaos = false, want true", preset)
			}
		})
	}
}
