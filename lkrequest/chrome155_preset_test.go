package lkrequest

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestChrome155PresetAvailable verifies chrome_155 is usable end to end through
// the Go binding: the preset resolves, a session builds on top of it, and the
// library attributes the client to Chrome 155 rather than silently falling back
// to an older profile. Offline — nothing here opens a connection.
func TestChrome155PresetAvailable(t *testing.T) {
	client, err := NewClientBuilder().SetPreset("chrome_155").Build()
	if err != nil {
		t.Skipf("chrome_155 preset not available in the loaded lkrequest library: %v", err)
	}
	t.Cleanup(client.Close)

	session, err := NewSessionBuilder(client).Build()
	if err != nil {
		t.Fatalf("NewSessionBuilder with chrome_155 preset error = %v", err)
	}
	t.Cleanup(session.Close)

	if got := clientPresetName(t, client); got != "Chrome 155" {
		t.Fatalf("fingerprint name = %q, want %q", got, "Chrome 155")
	}
}

// chrome155QUICAdditions are the switches Chrome 155's QUIC profile turns on
// and Chrome 154's lacks, by section of the fingerprint's "quic" layer:
// Chrome 155's HTTP/3 request handling, among them a PRIORITY_UPDATE sent for
// each request stream before its HEADERS, and its loss-recovery and
// packetization timing.
var chrome155QUICAdditions = map[string][]string{
	"h3": {
		"accept_truncated_response_payloads",
		"cancel_response_before_upload",
		"defer_qpack_stream_prefixes",
		"request_priority_updates",
		"wait_for_incomplete_response_frame_headers",
		"wait_for_response_after_upload_stop",
	},
	"packetization": {
		"ack_obligation_only",
		"chrome_loss_timing",
		"chrome_pto_timing",
		"chrome_rtt_quantization",
		"discard_blocked_close",
		"pto_single_probe_with_gap",
		"queue_pto_while_blocked",
		"reset_finished_on_stop",
		"reuse_rejected_0rtt_streams",
		"stop_before_matching_reset",
	},
}

// TestChrome155RetainsChrome154Shape pins what the Chrome 155 captures found:
// TCP TLS, H2, QUIC TLS, QUIC transport parameters, H3 SETTINGS and Initial
// packetization all keep Chrome 154's shape. What Chrome 155 adds are the QUIC
// behaviour switches in chrome155QUICAdditions, each on. With those set aside
// the two fingerprints are compared whole with only their names set aside, as
// chrome_154 is against chrome_153.
func TestChrome155RetainsChrome154Shape(t *testing.T) {
	c155 := presetFingerprintWithoutName(t, "chrome_155")
	c154 := presetFingerprintWithoutName(t, "chrome_154")

	// A library built without QUIC/H3 reports no quic layer for either preset.
	if quic155, ok := c155["quic"].(map[string]any); ok {
		quic154, _ := c154["quic"].(map[string]any)
		for section, names := range chrome155QUICAdditions {
			added, _ := quic155[section].(map[string]any)
			inherited, _ := quic154[section].(map[string]any)
			for _, name := range names {
				if added[name] != true {
					t.Errorf("chrome_155 quic.%s.%s = %v, want true", section, name, added[name])
				}
				if value, ok := inherited[name]; ok {
					t.Errorf("chrome_154 quic.%s.%s = %v, want it absent", section, name, value)
				}
				delete(added, name)
			}
		}
	}

	for layer, got := range c155 {
		if want, ok := c154[layer]; !ok || !reflect.DeepEqual(got, want) {
			t.Errorf("chrome_155 %q layer differs from chrome_154:\n got: %v\nwant: %v", layer, got, want)
		}
	}
	for layer := range c154 {
		if _, ok := c155[layer]; !ok {
			t.Errorf("chrome_155 is missing the %q layer chrome_154 has", layer)
		}
	}
}

// TestDefaultClientsUseChrome155 covers the library's default-preset switch
// from Chrome 144 to Chrome 155: NewDefaultClient (and with it the package-level
// helpers), a builder with no preset, and the "default" alias must all report
// the complete chrome_155 fingerprint, while an explicit chrome_144 keeps
// working for callers who relied on the old default.
func TestDefaultClientsUseChrome155(t *testing.T) {
	want := clientFingerprint(t, "chrome_155", func() (*Client, error) {
		return NewClientBuilder().SetPreset("chrome_155").Build()
	})

	defaults := map[string]func() (*Client, error){
		"NewDefaultClient":         NewDefaultClient,
		"NewClientBuilder().Build": NewClientBuilder().Build,
		`NewClient("default")`:     func() (*Client, error) { return NewClient("default") },
	}
	for name, build := range defaults {
		if got := clientFingerprint(t, name, build); !reflect.DeepEqual(got, want) {
			t.Errorf("%s fingerprint differs from chrome_155:\n got: %v\nwant: %v", name, got, want)
		}
	}

	client, err := NewClient("chrome_144")
	if err != nil {
		t.Fatalf(`NewClient("chrome_144") error = %v`, err)
	}
	t.Cleanup(client.Close)
	if got := clientPresetName(t, client); got != "Chrome 144" {
		t.Fatalf(`NewClient("chrome_144") fingerprint name = %q, want "Chrome 144"`, got)
	}
}

// clientFingerprint builds a client and decodes its full fingerprint payload,
// keyed by layer. Skipped when chrome_155 is unavailable: a library predating
// it still defaults to Chrome 144.
func clientFingerprint(t *testing.T, name string, build func() (*Client, error)) map[string]any {
	t.Helper()

	client, err := build()
	if err != nil {
		if name == "chrome_155" {
			t.Skipf("chrome_155 preset not available in the loaded lkrequest library: %v", err)
		}
		t.Fatalf("%s error = %v", name, err)
	}
	t.Cleanup(client.Close)

	raw, err := client.FingerprintInfoJSON()
	if err != nil {
		t.Fatalf("%s FingerprintInfoJSON error = %v", name, err)
	}

	var layers map[string]any
	if err := json.Unmarshal([]byte(raw), &layers); err != nil {
		t.Fatalf("unmarshal %s fingerprint info error = %v; payload = %s", name, err, raw)
	}
	return layers
}
