package lkrequest

import "testing"

// What Firefox 156 dropped from Firefox 147's ClientHello.
var (
	// TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA and TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA.
	firefox156DroppedCipherSuites = []int{0xc00a, 0xc009}
	// ffdhe2048 and ffdhe3072.
	firefox156DroppedGroups = []int{256, 257}
)

// hpkeAeadAES128GCM is the HPKE AEAD identifier Firefox 156 switched its ECH
// GREASE to; Firefox 147 used ChaCha20-Poly1305 (3).
const hpkeAeadAES128GCM = 1

// TestFirefox156PresetAvailable verifies firefox_156 is usable end to end
// through the Go binding and is attributed to Firefox 156 rather than silently
// falling back to an older profile. Offline — nothing here opens a connection.
func TestFirefox156PresetAvailable(t *testing.T) {
	client, err := NewClientBuilder().SetPreset("firefox_156").Build()
	if err != nil {
		t.Skipf("firefox_156 preset not available in the loaded lkrequest library: %v", err)
	}
	t.Cleanup(client.Close)

	session, err := NewSessionBuilder(client).Build()
	if err != nil {
		t.Fatalf("NewSessionBuilder with firefox_156 preset error = %v", err)
	}
	t.Cleanup(session.Close)

	if got := clientPresetName(t, client); got != "Firefox 156" {
		t.Fatalf("fingerprint name = %q, want %q", got, "Firefox 156")
	}
}

// TestFirefox156ClientHelloChanges pins the three places Firefox 156 diverges
// from Firefox 147 on TCP TLS: two ECDSA CBC suites and both FFDHE groups are
// gone, and ECH GREASE moved to AES-128-GCM. Each is asserted as an absence or
// an exact value, since a profile that kept 147's lists would still handshake
// fine and simply stop looking like Firefox 156.
//
// QUIC is also asserted off: upstream keeps it disabled until the stack can
// reproduce Firefox's transport and Initial-packet fields exactly, and a
// preset quietly gaining a guessed QUIC profile would be worse than none.
func TestFirefox156ClientHelloChanges(t *testing.T) {
	info := presetFingerprint(t, "firefox_156")

	for _, dropped := range firefox156DroppedCipherSuites {
		for _, suite := range info.TLS.CipherSuites {
			if suite == dropped {
				t.Errorf("firefox_156 still offers cipher suite 0x%04x", dropped)
			}
		}
	}
	for _, dropped := range firefox156DroppedGroups {
		for _, group := range info.TLS.SupportedGroups {
			if group == dropped {
				t.Errorf("firefox_156 still offers supported group %d", dropped)
			}
		}
	}

	if info.TLS.ECH == nil || info.TLS.ECH.Type != "grease" {
		t.Fatalf("firefox_156 ech = %+v, want GREASE ECH", info.TLS.ECH)
	}
	if info.TLS.ECH.AeadID != hpkeAeadAES128GCM {
		t.Errorf("firefox_156 ECH GREASE aead_id = %d, want %d (AES-128-GCM)", info.TLS.ECH.AeadID, hpkeAeadAES128GCM)
	}

	if info.Quic != nil {
		t.Errorf("firefox_156 reports a QUIC profile; want none until upstream can reproduce Firefox's")
	}
}
