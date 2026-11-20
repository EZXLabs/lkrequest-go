package lkrequest

import (
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// h2FrameData is the HTTP/2 DATA frame type.
const h2FrameData = 0x0

// TestProfileBundleFeaturesAreReported checks the feature names a library
// with custom profiles and profile bundles reports.
func TestProfileBundleFeaturesAreReported(t *testing.T) {
	requireProfileBundleAPI(t)

	for _, name := range []string{"tls-profile-json", "h2-profile-json", "preset-bundle-v2"} {
		if !FeatureSupported(name) {
			t.Errorf("FeatureSupported(%q) = false, want true", name)
		}
	}
	if got, want := FeatureSupported("quic-tls-profile-json"), FeatureSupported("quic-h3"); got != want {
		t.Errorf(`FeatureSupported("quic-tls-profile-json") = %v, want %v as for "quic-h3"`, got, want)
	}
}

// TestExportedBundlesReproduceThePresets checks every built-in preset end to
// end: its exported bundle declares the format and version, and loading it
// gives a client the very fingerprint the named preset gives.
func TestExportedBundlesReproduceThePresets(t *testing.T) {
	requireProfileBundleAPI(t)

	presets, err := ListPresetsJSON()
	if err != nil {
		t.Fatalf("ListPresetsJSON() error = %v", err)
	}
	for _, name := range decodeJSONText[[]string](t, presets) {
		t.Run(name, func(t *testing.T) {
			bundle := exportBundle(t, name)
			var header struct {
				Format        string `json:"format"`
				SchemaVersion int    `json:"schema_version"`
				Name          string `json:"name"`
			}
			if err := json.Unmarshal([]byte(bundle), &header); err != nil {
				t.Fatalf("unmarshal exported bundle error = %v", err)
			}
			if header.Format != "lkrequest-profile-bundle" || header.SchemaVersion != 2 || header.Name != name {
				t.Errorf("bundle header = %+v, want format lkrequest-profile-bundle, schema_version 2, name %s", header, name)
			}

			got := builderFingerprint(t, NewClientBuilder().SetPresetJSON(bundle))
			if want := builderFingerprint(t, NewClientBuilder().SetPreset(name)); !reflect.DeepEqual(got, want) {
				t.Errorf("fingerprint from the bundle differs from %s's:\n got: %v\nwant: %v", name, got, want)
			}
		})
	}

	if _, err := ExportPresetJSON("chrome_0"); err == nil {
		t.Error(`ExportPresetJSON("chrome_0") succeeded, want an error for an unknown preset`)
	}
}

// TestPresetFileLoadsTheBundle checks SetPresetFile against SetPreset, from a
// path that is not ASCII.
func TestPresetFileLoadsTheBundle(t *testing.T) {
	requireProfileBundleAPI(t)

	path := filepath.Join(t.TempDir(), "预设-firefox_156.json")
	if err := os.WriteFile(path, []byte(exportBundle(t, "firefox_156")), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	got := builderFingerprint(t, NewClientBuilder().SetPresetFile(path))
	if want := builderFingerprint(t, NewClientBuilder().SetPreset("firefox_156")); !reflect.DeepEqual(got, want) {
		t.Errorf("fingerprint from the file differs from firefox_156's:\n got: %v\nwant: %v", got, want)
	}
}

// TestBundleRuntimeReachesTheWire checks that a bundle's runtime section
// governs the connection, beyond the profile data a fingerprint shows. With
// separate_end_stream_data, a request body's END_STREAM moves from its DATA
// frame to an empty DATA frame of its own.
func TestBundleRuntimeReachesTheWire(t *testing.T) {
	requireProfileBundleAPI(t)

	bundle := exportBundle(t, "firefox_156")
	for _, tc := range []struct {
		name   string
		bundle string
		want   []h2Data
	}{
		{name: "as exported", bundle: bundle, want: []h2Data{{length: 7, endStream: true}}},
		{
			name:   "separate END_STREAM",
			bundle: withJSONField(t, bundle, true, "runtime", "h2_behavior", "write", "separate_end_stream_data"),
			want:   []h2Data{{length: 7}, {length: 0, endStream: true}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			probe := startH2DataProbe(t)
			client := buildBundleClient(t, NewClientBuilder().SetPresetJSON(tc.bundle).SetVerify(false))
			session, err := NewSessionBuilder(client).Build()
			if err != nil {
				t.Fatalf("SessionBuilder.Build() error = %v", err)
			}
			t.Cleanup(session.Close)

			resp, err := session.Post(probe.url("/")).SetBodyBytes([]byte("payload")).SetTimeout(5000).Send()
			if err != nil {
				t.Fatalf("Send() error = %v", err)
			}
			defer resp.Close()
			if resp.StatusCode() != http.StatusOK {
				t.Fatalf("status = %d, want 200", resp.StatusCode())
			}
			if got := probe.recorded(); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("DATA frames = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestBundleErrorsCarryTheirCause checks how Build reports a bundle the
// library rejects: an *LkError with the matching code and, for a document,
// its preset_error with a stable code and the JSON Pointer of the failure.
func TestBundleErrorsCarryTheirCause(t *testing.T) {
	requireProfileBundleAPI(t)
	snapshot, err := GetPresetDetailJSON("chrome_157")
	if err != nil {
		t.Fatalf("GetPresetDetailJSON() error = %v", err)
	}

	for _, tc := range []struct {
		name      string
		configure func(*ClientBuilder) *ClientBuilder
		code      ErrorCode
		cause     string // preset_error code; empty when there is none
		path      string
	}{
		{
			name: "duplicate key",
			configure: func(b *ClientBuilder) *ClientBuilder {
				return b.SetPresetJSON(`{"schema_version":2,"format":"lkrequest-profile-bundle","name":"a","name":"b"}`)
			},
			code: ErrInvalidConfig, cause: "duplicate_key", path: "/name",
		},
		{
			name:      "preset detail snapshot",
			configure: func(b *ClientBuilder) *ClientBuilder { return b.SetPresetJSON(snapshot) },
			code:      ErrInvalidConfig, cause: "legacy_bundle_incomplete", path: "/",
		},
		{
			name:      "over 1 MiB",
			configure: func(b *ClientBuilder) *ClientBuilder { return b.SetPresetJSON(strings.Repeat(" ", 1<<20+1)) },
			code:      ErrResourceLimitExceeded, cause: "resource_limit", path: "/",
		},
		{
			name: "missing file",
			configure: func(b *ClientBuilder) *ClientBuilder {
				return b.SetPresetFile(filepath.Join(t.TempDir(), "missing.json"))
			},
			code: ErrIO, cause: "io", path: "/",
		},
		{
			name:      "empty path",
			configure: func(b *ClientBuilder) *ClientBuilder { return b.SetPresetFile("") },
			code:      ErrInvalidArgument,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, err := tc.configure(NewClientBuilder()).Build()
			if err == nil {
				client.Close()
				t.Fatal("Build() succeeded, want an error")
			}
			var lkErr *LkError
			if !errors.As(err, &lkErr) || lkErr.Code() != tc.code {
				t.Fatalf("Build() error = %v (%T), want an *LkError with code %v", err, err, tc.code)
			}

			var diagnostics struct {
				PresetError *struct {
					Code    string `json:"code"`
					Path    string `json:"path"`
					Message string `json:"message"`
				} `json:"preset_error"`
			}
			if err := json.Unmarshal([]byte(lkErr.DiagnosticsJSON()), &diagnostics); err != nil {
				t.Fatalf("unmarshal diagnostics error = %v; payload = %s", err, lkErr.DiagnosticsJSON())
			}
			cause := diagnostics.PresetError
			switch {
			case tc.cause == "" && cause != nil:
				t.Errorf("preset_error = %+v, want none", *cause)
			case tc.cause != "" && cause == nil:
				t.Errorf("diagnostics carry no preset_error: %s", lkErr.DiagnosticsJSON())
			case cause != nil && (cause.Code != tc.cause || cause.Path != tc.path || cause.Message == ""):
				t.Errorf("preset_error = %+v, want code %s at %s with a message", *cause, tc.cause, tc.path)
			}
		})
	}
}

// TestBundleGivesWayToNamedPresets checks how a bundle and SetPreset replace
// each other: an empty bundle restores the named preset, and a later
// SetPreset replaces the bundle.
func TestBundleGivesWayToNamedPresets(t *testing.T) {
	requireProfileBundleAPI(t)
	firefox := exportBundle(t, "firefox_156")
	chrome := builderFingerprint(t, NewClientBuilder().SetPreset("chrome_157"))

	for name, builder := range map[string]*ClientBuilder{
		"bundle removed": NewClientBuilder().SetPreset("chrome_157").SetPresetJSON(firefox).SetPresetJSON(""),
		"preset after":   NewClientBuilder().SetPresetJSON(firefox).SetPreset("chrome_157"),
	} {
		if got := builderFingerprint(t, builder); !reflect.DeepEqual(got, chrome) {
			t.Errorf("%s: fingerprint differs from chrome_157's:\n got: %v\nwant: %v", name, got, chrome)
		}
	}
}

// TestComponentProfilesTakePrecedence checks the precedence of the profile
// setters: a custom TLS profile wins over a bundle and a custom H2 profile
// over both H2 presets, whichever is set first, and an empty profile hands
// the layer back.
func TestComponentProfilesTakePrecedence(t *testing.T) {
	requireProfileBundleAPI(t)
	firefoxBundle := exportBundle(t, "firefox_156")
	firefox := builderFingerprint(t, NewClientBuilder().SetPreset("firefox_156"))

	t.Run("tls", func(t *testing.T) {
		tls150 := fingerprintLayerJSON(t, "chrome_150", "tls")
		for name, builder := range map[string]*ClientBuilder{
			"set before the bundle": NewClientBuilder().SetTLSProfileJSON(tls150).SetPresetJSON(firefoxBundle),
			"set after the bundle":  NewClientBuilder().SetPresetJSON(firefoxBundle).SetTLSProfileJSON(tls150),
		} {
			got := builderFingerprint(t, builder)
			if !reflect.DeepEqual(got["tls"], decodeJSONValue(t, tls150)) {
				t.Errorf("%s: tls = %v, want chrome_150's", name, got["tls"])
			}
			if !reflect.DeepEqual(got["h2"], firefox["h2"]) {
				t.Errorf("%s: h2 = %v, want the bundle's", name, got["h2"])
			}
		}
		removed := NewClientBuilder().SetPresetJSON(firefoxBundle).SetTLSProfileJSON(tls150).SetTLSProfileJSON("")
		if got := builderFingerprint(t, removed); !reflect.DeepEqual(got, firefox) {
			t.Errorf("fingerprint after removing the TLS profile differs from the bundle's:\n got: %v\nwant: %v", got, firefox)
		}
	})

	t.Run("h2", func(t *testing.T) {
		custom := withJSONField(t, fingerprintLayerJSON(t, "firefox_156", "h2"), 123456, "window_update")
		build := func() *ClientBuilder {
			return NewClientBuilder().SetH2ProfileJSON(custom).SetPreset("firefox_147").SetH2Preset("chrome_150")
		}
		got := builderFingerprint(t, build())
		if !reflect.DeepEqual(got["h2"], decodeJSONValue(t, custom)) {
			t.Errorf("h2 = %v, want the custom profile", got["h2"])
		}
		if name := got["tls"].(map[string]any)["name"]; name != "Firefox 147" {
			t.Errorf("tls name = %v, want the preset's Firefox 147", name)
		}
		removed := builderFingerprint(t, build().SetH2ProfileJSON(""))
		if want := fingerprintLayerJSON(t, "chrome_150", "h2"); !reflect.DeepEqual(removed["h2"], decodeJSONValue(t, want)) {
			t.Errorf("h2 after removing the custom profile = %v, want the H2 preset's", removed["h2"])
		}
	})
}

// TestQUICTLSProfileIsSetApartFromTCP checks the QUIC TLS setters: a custom
// QUIC TLS profile stands beside a custom TCP one and the preset's QUIC
// transport, an empty one restores the preset's, and ClearQUICFingerprint
// makes QUIC use the TCP profile, the last of the two calls winning.
func TestQUICTLSProfileIsSetApartFromTCP(t *testing.T) {
	requireProfileBundleAPI(t)
	if !FeatureSupported("quic-tls-profile-json") {
		t.Skip("QUIC TLS profiles need a QUIC/H3-capable lkrequest library")
	}

	tcp := decodeJSONValue(t, fingerprintLayerJSON(t, "chrome_131", "tls"))
	quicTLS := withJSONField(t, fingerprintLayerJSON(t, "chrome_156", "quic_tls"), "independent QUIC TLS", "name")
	chrome157 := builderFingerprint(t, NewClientBuilder().SetPreset("chrome_157"))
	build := func() *ClientBuilder {
		return NewClientBuilder().
			SetTLSProfileJSON(fingerprintLayerJSON(t, "chrome_131", "tls")).
			SetQUICTLSProfileJSON(quicTLS).
			SetPreset("chrome_157")
	}

	got := builderFingerprint(t, build())
	if !reflect.DeepEqual(got["tls"], tcp) || !reflect.DeepEqual(got["quic_tls"], decodeJSONValue(t, quicTLS)) {
		t.Errorf("tls / quic_tls = %v / %v, want the two custom profiles", got["tls"], got["quic_tls"])
	}
	if !reflect.DeepEqual(got["quic"], chrome157["quic"]) {
		t.Errorf("quic = %v, want chrome_157's transport", got["quic"])
	}

	for _, tc := range []struct {
		name    string
		builder *ClientBuilder
		want    any
	}{
		{name: "custom profile removed", builder: build().SetQUICTLSProfileJSON(""), want: chrome157["quic_tls"]},
		{name: "cleared", builder: build().ClearQUICFingerprint(), want: nil},
		{name: "cleared, then set", builder: build().ClearQUICFingerprint().SetQUICTLSProfileJSON(quicTLS), want: decodeJSONValue(t, quicTLS)},
	} {
		got := builderFingerprint(t, tc.builder)
		if !reflect.DeepEqual(got["quic_tls"], tc.want) {
			t.Errorf("%s: quic_tls = %v, want %v", tc.name, got["quic_tls"], tc.want)
		}
		if !reflect.DeepEqual(got["tls"], tcp) {
			t.Errorf("%s: tls = %v, want the custom TCP profile", tc.name, got["tls"])
		}
	}
}

// TestComponentProfilesRejectInvalidInput checks that a profile the library
// rejects fails Build, naming the setter.
func TestComponentProfilesRejectInvalidInput(t *testing.T) {
	requireProfileBundleAPI(t)
	tlsProfile := fingerprintLayerJSON(t, "chrome_155", "tls")
	h2 := fingerprintLayerJSON(t, "firefox_156", "h2")
	// The unmodified profiles load, so each case fails for its change alone.
	buildBundleClient(t, NewClientBuilder().SetTLSProfileJSON(tlsProfile).SetH2ProfileJSON(h2))

	type invalid struct {
		name      string
		configure func(*ClientBuilder) *ClientBuilder
		action    string
	}
	setTLS := func(name, profile string) invalid {
		return invalid{name, func(b *ClientBuilder) *ClientBuilder { return b.SetTLSProfileJSON(profile) }, "set tls profile json"}
	}
	setH2 := func(name, profile string) invalid {
		return invalid{name, func(b *ClientBuilder) *ClientBuilder { return b.SetH2ProfileJSON(profile) }, "set h2 profile json"}
	}
	cases := []invalid{
		setTLS("tls without cipher suites", withJSONField(t, tlsProfile, []any{}, "cipher_suites")),
		setTLS("tls session ID over 32 bytes", withJSONField(t, tlsProfile, 33, "session_id_length")),
		setTLS("tls without groups", withJSONField(t, tlsProfile, []any{}, "supported_groups")),
		setTLS("tls unknown key share", withJSONField(t, tlsProfile, []any{0xffff}, "key_share_curves")),
		setTLS("tls ClientHello mirror", withJSONField(t, tlsProfile, map[string]any{"input": "00"}, "client_hello_mirror")),
		setTLS("tls duplicate key", `{"session_id_length":0,`+tlsProfile[1:]),
		setTLS("tls not UTF-8", "\xff"),
		setTLS("tls over 1 MiB", strings.Repeat(" ", 1<<20+1)),
		setH2("h2 without fields", "{}"),
		setH2("h2 behavior", withJSONField(t, h2, map[string]any{}, "behavior")),
	}
	if FeatureSupported("quic-tls-profile-json") {
		quicTLS := fingerprintLayerJSON(t, "chrome_156", "quic_tls")
		buildBundleClient(t, NewClientBuilder().SetQUICTLSProfileJSON(quicTLS))
		tls12 := withJSONField(t, quicTLS, "tls12", "tls_max_version")
		cases = append(cases, invalid{
			"quic tls without TLS 1.3",
			func(b *ClientBuilder) *ClientBuilder { return b.SetQUICTLSProfileJSON(tls12) },
			"set quic tls profile json",
		})
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := tc.configure(NewClientBuilder()).Build()
			if err == nil {
				client.Close()
				t.Fatal("Build() succeeded, want an error")
			}
			if want := "client builder " + tc.action + " failed"; !strings.Contains(err.Error(), want) {
				t.Errorf("Build() error = %v, want it to contain %q", err, want)
			}
		})
	}
}

// requireProfileBundleAPI skips when the loaded library predates custom
// profiles and profile bundles.
func requireProfileBundleAPI(t *testing.T) {
	t.Helper()
	if ffi_lk_client_builder_set_preset_json == nil {
		t.Skip("profile bundles are not supported by the loaded lkrequest library")
	}
}

// exportBundle returns ExportPresetJSON(name), failing the test on an error.
func exportBundle(t *testing.T, name string) string {
	t.Helper()
	bundle, err := ExportPresetJSON(name)
	if err != nil {
		t.Fatalf("ExportPresetJSON(%q) error = %v", name, err)
	}
	return bundle
}

// buildBundleClient builds builder into a client closed when the test ends.
func buildBundleClient(t *testing.T, builder *ClientBuilder) *Client {
	t.Helper()
	client, err := builder.Build()
	if err != nil {
		t.Fatalf("ClientBuilder.Build() error = %v", err)
	}
	t.Cleanup(client.Close)
	return client
}

// builderFingerprint builds builder and decodes the client's fingerprint,
// keyed by layer.
func builderFingerprint(t *testing.T, builder *ClientBuilder) map[string]any {
	t.Helper()
	raw, err := buildBundleClient(t, builder).FingerprintInfoJSON()
	if err != nil {
		t.Fatalf("FingerprintInfoJSON() error = %v", err)
	}
	var layers map[string]any
	if err := json.Unmarshal([]byte(raw), &layers); err != nil {
		t.Fatalf("unmarshal fingerprint info error = %v; payload = %s", err, raw)
	}
	return layers
}

// fingerprintLayerJSON returns one layer of preset's fingerprint as JSON, the
// form the profile setters take.
func fingerprintLayerJSON(t *testing.T, preset, layer string) string {
	t.Helper()
	raw, err := buildBundleClient(t, NewClientBuilder().SetPreset(preset)).FingerprintInfoJSON()
	if err != nil {
		t.Fatalf("FingerprintInfoJSON() error = %v", err)
	}
	var layers map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &layers); err != nil {
		t.Fatalf("unmarshal fingerprint info error = %v; payload = %s", err, raw)
	}
	if _, ok := layers[layer]; !ok {
		t.Fatalf("%s fingerprint has no %q layer", preset, layer)
	}
	return string(layers[layer])
}

// decodeJSONValue decodes a JSON document the way fingerprints are decoded,
// for comparison with one of their layers.
func decodeJSONValue(t *testing.T, document string) any {
	t.Helper()
	var value any
	if err := json.Unmarshal([]byte(document), &value); err != nil {
		t.Fatalf("unmarshal %.80s error = %v", document, err)
	}
	return value
}

// withJSONField returns document with the field at path, a chain of object
// keys, set to value. Numbers keep their exact text.
func withJSONField(t *testing.T, document string, value any, path ...string) string {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(document))
	decoder.UseNumber()
	var root map[string]any
	if err := decoder.Decode(&root); err != nil {
		t.Fatalf("decode %.80s error = %v", document, err)
	}
	object := root
	for _, key := range path[:len(path)-1] {
		next, ok := object[key].(map[string]any)
		if !ok {
			t.Fatalf("%v: no object at %q", path, key)
		}
		object = next
	}
	object[path[len(path)-1]] = value
	encoded, err := json.Marshal(root)
	if err != nil {
		t.Fatalf("encode document error = %v", err)
	}
	return string(encoded)
}

// h2Data is what h2DataProbe notes about one DATA frame.
type h2Data struct {
	length    int
	endStream bool
}

// h2DataProbe is an HTTP/2 server over TLS that records every DATA frame it
// receives and answers each request with an empty 200 once the request has
// ended, so the whole body reaches it.
type h2DataProbe struct {
	listener net.Listener
	mu       sync.Mutex
	frames   []h2Data
}

// startH2DataProbe starts an h2DataProbe on a loopback port, stopped at test
// cleanup.
func startH2DataProbe(t *testing.T) *h2DataProbe {
	t.Helper()

	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{selfSignedTLSCert(t)},
		NextProtos:   []string{"h2"},
	})
	if err != nil {
		t.Fatalf("tls.Listen() error = %v", err)
	}
	t.Cleanup(func() { listener.Close() })

	probe := &h2DataProbe{listener: listener}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go probe.serve(conn)
		}
	}()
	return probe
}

// url returns the probe's https URL for path.
func (p *h2DataProbe) url(path string) string {
	return "https://" + p.listener.Addr().String() + path
}

// recorded returns the DATA frames seen so far, in arrival order.
func (p *h2DataProbe) recorded() []h2Data {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]h2Data(nil), p.frames...)
}

// serve speaks just enough HTTP/2 on conn to take requests in: it settles
// SETTINGS and PING, notes DATA frames, and answers a stream that has ended.
func (p *h2DataProbe) serve(conn net.Conn) {
	defer conn.Close()

	preface := make([]byte, len(h2ClientPreface))
	if _, err := io.ReadFull(conn, preface); err != nil || string(preface) != h2ClientPreface {
		return
	}
	if writeH2Frame(conn, h2FrameSettings, 0, 0, nil) != nil {
		return
	}

	header := make([]byte, 9)
	for {
		if _, err := io.ReadFull(conn, header); err != nil {
			return
		}
		length := int(header[0])<<16 | int(header[1])<<8 | int(header[2])
		frameType, flags := header[3], header[4]
		stream := binary.BigEndian.Uint32(header[5:]) & 0x7fffffff
		payload := make([]byte, length)
		if _, err := io.ReadFull(conn, payload); err != nil {
			return
		}

		var err error
		ended := false
		switch frameType {
		case h2FrameSettings:
			if flags&h2FlagAck == 0 {
				err = writeH2Frame(conn, h2FrameSettings, h2FlagAck, 0, nil)
			}
		case h2FramePing:
			if flags&h2FlagAck == 0 {
				err = writeH2Frame(conn, h2FramePing, h2FlagAck, 0, payload)
			}
		case h2FrameHeaders:
			ended = flags&h2FlagEndStream != 0
		case h2FrameData:
			data := len(payload)
			if flags&h2FlagPadded != 0 && len(payload) > 0 {
				data -= 1 + int(payload[0])
			}
			ended = flags&h2FlagEndStream != 0
			p.mu.Lock()
			p.frames = append(p.frames, h2Data{length: data, endStream: ended})
			p.mu.Unlock()
		case h2FrameGoAway:
			return
		}
		if err == nil && ended {
			err = writeH2Frame(conn, h2FrameHeaders, h2FlagEndStream|h2FlagEndHeaders, stream, []byte{hpackStatus200})
		}
		if err != nil {
			return
		}
	}
}
