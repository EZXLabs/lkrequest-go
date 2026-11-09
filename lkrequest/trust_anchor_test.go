package lkrequest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// customTrustAnchorID is a private-deployment identifier the tests bind to a
// root of their own.
const customTrustAnchorID = "81fd5901"

// TestChromeSendsItsTrustAnchorIDsInOneSortedOrder pins how chrome_155,
// chrome_156 and chrome_157 send their trust anchor identifiers: the 28 each
// lists, in the sorted order Chromium encodes them in, go out in that order on
// every handshake.
func TestChromeSendsItsTrustAnchorIDsInOneSortedOrder(t *testing.T) {
	for _, preset := range []string{"chrome_155", "chrome_156", "chrome_157"} {
		t.Run(preset, func(t *testing.T) {
			listed := profileTrustAnchorIDs(t, preset)
			if len(listed) != 28 || !slices.IsSorted(listed) {
				t.Fatalf("%s lists %d trust anchor identifiers %v, want 28 in sorted order", preset, len(listed), listed)
			}

			client := buildTrustAnchorClient(t, NewClientBuilder().SetPreset(preset).SetVerify(false))
			for i, ids := range sentTrustAnchorIDs(t, client, 3) {
				if !slices.Equal(ids, listed) {
					t.Errorf("ClientHello %d sent %v, want the listed order %v", i, ids, listed)
				}
			}
		})
	}
}

// TestChrome154ShufflesItsTrustAnchorIDs is the control for the test above:
// chrome_154 sends the same identifiers, shuffled anew for every handshake,
// so the capture below can tell a fixed order from a shuffled one.
func TestChrome154ShufflesItsTrustAnchorIDs(t *testing.T) {
	want := slices.Clone(profileTrustAnchorIDs(t, "chrome_154"))
	slices.Sort(want)

	client := buildTrustAnchorClient(t, NewClientBuilder().SetPreset("chrome_154").SetVerify(false))
	orders := map[string]bool{}
	for i, ids := range sentTrustAnchorIDs(t, client, 4) {
		orders[strings.Join(ids, ",")] = true
		got := slices.Clone(ids)
		slices.Sort(got)
		if !slices.Equal(got, want) {
			t.Errorf("ClientHello %d sent %v, want the identifiers %v in some order", i, ids, want)
		}
	}
	// 28 identifiers have 28! orders; four handshakes sharing one is a bug.
	if len(orders) == 1 {
		t.Errorf("every ClientHello sent the identifiers in one order, want them shuffled")
	}
}

// TestTrustedAdvertisementSendsTheIDsOfTrustedRoots checks
// TrustAnchorAdvertiseTrusted on the wire: sorted, unique identifiers, the
// custom root's among them, and otherwise bundled identifiers of the built-in
// roots, which Chrome 157 lists too, besides those of intermediates.
func TestTrustedAdvertisementSendsTheIDsOfTrustedRoots(t *testing.T) {
	requireTrustAnchorAPI(t)
	listed := profileTrustAnchorIDs(t, "chrome_157")
	rootDER := selfSignedTLSCert(t).Certificate[0]

	client := buildTrustAnchorClient(t, NewClientBuilder().
		SetPreset("chrome_157").
		AddCACertMemory(rootDER).
		AddTrustAnchorBinding(trustAnchorIdentity(t, rootDER), customTrustAnchorID).
		SetTrustAnchorPolicy(TrustAnchorPolicy{Advertisement: TrustAnchorAdvertiseTrusted}))
	for i, ids := range sentTrustAnchorIDs(t, client, 1) {
		if !slices.IsSorted(ids) || len(slices.Compact(slices.Clone(ids))) != len(ids) {
			t.Errorf("ClientHello %d sent %v, want sorted, unique identifiers", i, ids)
		}
		if !slices.Contains(ids, customTrustAnchorID) {
			t.Errorf("ClientHello %d sent %v, want the custom root's %s among them", i, ids, customTrustAnchorID)
		}
		for _, id := range ids {
			if id != customTrustAnchorID && !slices.Contains(listed, id) {
				t.Errorf("ClientHello %d sent %s, which neither the custom root nor chrome_157 lists", i, id)
			}
		}
		if len(ids)-1 >= len(listed) {
			t.Errorf("ClientHello %d sent %d bundled identifiers, want fewer than the %d chrome_157 lists", i, len(ids)-1, len(listed))
		}
	}
}

// TestFailedHandshakeReportsItsTrustAnchors checks the trust_anchor object of
// an error's DiagnosticsJSON: it reports the identifiers the failed handshake
// requested, and, the server having listed none, nothing it could serve.
func TestFailedHandshakeReportsItsTrustAnchors(t *testing.T) {
	requireTrustAnchorAPI(t)
	listed := profileTrustAnchorIDs(t, "chrome_157")
	url, _ := catchClientHellos(t)

	client := buildTrustAnchorClient(t, NewClientBuilder().SetPreset("chrome_157").SetVerify(false))
	resp, err := newTrustAnchorSession(t, client).Get(url).SetTimeout(5000).Send()
	if err == nil {
		resp.Close()
		t.Fatal("Send() succeeded against a server that hangs up after the ClientHello")
	}
	var lkErr *LkError
	if !errors.As(err, &lkErr) {
		t.Fatalf("Send() error = %v (%T), want an *LkError", err, err)
	}

	info := trustAnchorDiagnostics(t, lkErr.DiagnosticsJSON())
	if info == nil {
		t.Fatalf("error diagnostics carry no trust_anchor object: %s", lkErr.DiagnosticsJSON())
	}
	if got := jsonStrings(info["requested_ids"]); !slices.Equal(got, listed) {
		t.Errorf("requested_ids = %v, want %v", got, listed)
	}
	if got := jsonStrings(info["available_ids"]); len(got) != 0 {
		t.Errorf("available_ids = %v, want none", got)
	}
	if info["authenticated"] != false {
		t.Errorf("authenticated = %v, want false", info["authenticated"])
	}
}

// TestVerifiedConnectionNamesItsTrustAnchor checks the trust_anchor object of
// a response's DiagnosticsJSON: a verified connection names the root that
// verified it by its TrustAnchorIdentity, with nothing to recover from, and a
// request on that connection when pooled reports no new negotiation. Both the
// default policy and trusted advertisement with recovery reach a server that
// does not negotiate trust anchors.
func TestVerifiedConnectionNamesItsTrustAnchor(t *testing.T) {
	requireTrustAnchorAPI(t)
	listed := profileTrustAnchorIDs(t, "chrome_157")
	cert := selfSignedTLSCert(t)
	rootDER := cert.Certificate[0]
	identity := trustAnchorIdentity(t, rootDER)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	server.StartTLS()
	t.Cleanup(server.Close)

	for _, tc := range []struct {
		name      string
		configure func(*ClientBuilder) *ClientBuilder
	}{
		{name: "profile", configure: func(b *ClientBuilder) *ClientBuilder { return b }},
		{name: "trusted with recovery", configure: func(b *ClientBuilder) *ClientBuilder {
			return b.AddTrustAnchorBinding(identity, customTrustAnchorID).SetTrustAnchorPolicy(TrustAnchorPolicy{
				Advertisement: TrustAnchorAdvertiseTrusted,
				Recovery:      TrustAnchorRecoveryOnce,
			})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := buildTrustAnchorClient(t, tc.configure(NewClientBuilder().SetPreset("chrome_157").AddCACertMemory(rootDER)))
			session := newTrustAnchorSession(t, client)

			info := trustAnchorOfRequest(t, session, server.URL)
			if info == nil {
				t.Fatal("the request that opened the connection reports no trust_anchor object")
			}
			requested := jsonStrings(info["requested_ids"])
			if tc.name == "profile" {
				if !slices.Equal(requested, listed) {
					t.Errorf("requested_ids = %v, want the chrome_157 list %v", requested, listed)
				}
			} else if !slices.IsSorted(requested) || !slices.Contains(requested, customTrustAnchorID) {
				t.Errorf("requested_ids = %v, want sorted identifiers including %s", requested, customTrustAnchorID)
			}
			delete(info, "requested_ids")

			want := map[string]any{
				"available_ids":     []any{},
				"matched_request":   false,
				"verified_anchor":   identity,
				"authenticated":     true,
				"ech_rejected":      false,
				"recovered":         false,
				"failure":           nil,
				"recovery_eligible": false,
				"previous_attempt":  nil,
			}
			if !reflect.DeepEqual(info, want) {
				t.Errorf("trust_anchor = %v, want %v", info, want)
			}

			if pooled := trustAnchorOfRequest(t, session, server.URL); pooled != nil {
				t.Errorf("a request on the pooled connection reports trust_anchor %v, want none", pooled)
			}
		})
	}
}

// TestTrustAnchorRecoveryUsesTheBundledIDs checks that recovery needs no
// binding of its own: the bundled identifiers of the built-in roots are
// usable, with the profile's advertisement or with trusted advertisement.
func TestTrustAnchorRecoveryUsesTheBundledIDs(t *testing.T) {
	requireTrustAnchorAPI(t)

	for name, builder := range map[string]*ClientBuilder{
		"chrome_157": NewClientBuilder().SetPreset("chrome_157").
			SetTrustAnchorPolicy(TrustAnchorPolicy{Recovery: TrustAnchorRecoveryOnce}),
		"default preset, trusted": NewClientBuilder().SetTrustAnchorPolicy(TrustAnchorPolicy{
			Advertisement: TrustAnchorAdvertiseTrusted,
			Recovery:      TrustAnchorRecoveryOnce,
		}),
	} {
		client, err := builder.Build()
		if err != nil {
			t.Errorf("%s: Build() error = %v", name, err)
			continue
		}
		client.Close()
	}
}

// TestTrustAnchorSettingsBuildCannotHonour checks that settings the library
// cannot honour fail Build, with an invalid-config error where the library
// reports them and a Go error where they never reach it.
func TestTrustAnchorSettingsBuildCannotHonour(t *testing.T) {
	requireTrustAnchorAPI(t)
	recoverOnce := TrustAnchorPolicy{Recovery: TrustAnchorRecoveryOnce}
	rootDER := selfSignedTLSCert(t).Certificate[0]
	otherDER := selfSignedTLSCert(t).Certificate[0]
	rootID := trustAnchorIdentity(t, rootDER)

	cases := []struct {
		name      string
		configure func(*ClientBuilder) *ClientBuilder
		code      ErrorCode // zero for an error raised before the library
		message   string
	}{
		{
			name: "advertisement out of range",
			configure: func(b *ClientBuilder) *ClientBuilder {
				return b.SetTrustAnchorPolicy(TrustAnchorPolicy{Advertisement: 2})
			},
			message: "invalid trust anchor advertisement 2",
		},
		{
			name: "recovery out of range",
			configure: func(b *ClientBuilder) *ClientBuilder {
				return b.SetTrustAnchorPolicy(TrustAnchorPolicy{Recovery: 2})
			},
			message: "invalid trust anchor recovery 2",
		},
		{
			name: "recovery without verification",
			configure: func(b *ClientBuilder) *ClientBuilder {
				return b.SetVerify(false).SetTrustAnchorPolicy(recoverOnce)
			},
			code:    ErrInvalidConfig,
			message: "requires certificate verification",
		},
		{
			name: "recovery with direct fallback",
			configure: func(b *ClientBuilder) *ClientBuilder {
				return b.SetFallbackProxyToDirect(true).SetTrustAnchorPolicy(recoverOnce)
			},
			code:    ErrInvalidConfig,
			message: "proxy-to-direct fallback",
		},
		{
			name: "preset without the extension",
			configure: func(b *ClientBuilder) *ClientBuilder {
				return b.SetPreset("firefox_147").SetTrustAnchorPolicy(recoverOnce)
			},
			code:    ErrInvalidConfig,
			message: "trust_anchors extension slot",
		},
		{
			name: "malformed identifier",
			configure: func(b *ClientBuilder) *ClientBuilder {
				return b.AddCACertMemory(rootDER).AddTrustAnchorBinding(rootID, "zz")
			},
			message: "client builder add trust anchor binding failed",
		},
		{
			name: "root not trusted",
			configure: func(b *ClientBuilder) *ClientBuilder {
				return b.AddTrustAnchorBinding(strings.Repeat("00", 32), customTrustAnchorID)
			},
			code:    ErrInvalidConfig,
			message: "untrusted root",
		},
		{
			name: "identifier bound to two roots",
			configure: func(b *ClientBuilder) *ClientBuilder {
				return b.AddCACertMemory(rootDER).
					AddCACertMemory(otherDER).
					AddTrustAnchorBinding(rootID, customTrustAnchorID).
					AddTrustAnchorBinding(trustAnchorIdentity(t, otherDER), customTrustAnchorID)
			},
			code:    ErrInvalidConfig,
			message: "conflicting trust anchor ID bindings",
		},
	}
	// TLS synthesis needs a synthetic-fp library; the case runs only there.
	if synthetic, err := NewClientBuilder().SetRandomize(RandomizeRecombine, FingerprintLayersAll).Build(); err == nil {
		synthetic.Close()
		cases = append(cases, struct {
			name      string
			configure func(*ClientBuilder) *ClientBuilder
			code      ErrorCode
			message   string
		}{
			name: "TLS synthesis",
			configure: func(b *ClientBuilder) *ClientBuilder {
				return b.SetRandomize(RandomizeRecombine, FingerprintLayersAll).SetTrustAnchorPolicy(recoverOnce)
			},
			code:    ErrInvalidConfig,
			message: "TLS synthesis is incompatible",
		})
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := tc.configure(NewClientBuilder()).Build()
			if err == nil {
				client.Close()
				t.Fatalf("Build() succeeded, want an error containing %q", tc.message)
			}
			if !strings.Contains(err.Error(), tc.message) {
				t.Errorf("Build() error = %v, want it to contain %q", err, tc.message)
			}
			var lkErr *LkError
			if isLkErr := errors.As(err, &lkErr); tc.code != 0 && (!isLkErr || lkErr.Code() != tc.code) {
				t.Errorf("Build() error = %v (%T), want an *LkError with code %v", err, err, tc.code)
			}
		})
	}
}

// TestTrustAnchorIdentity checks the identity against one computed here from
// the certificate, independently of the library: it names the root — its
// subject and public key — not the certificate's bytes, and anything but a
// certificate is an error.
func TestTrustAnchorIdentity(t *testing.T) {
	requireTrustAnchorAPI(t)

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	root := rootCertDER(t, key, 1)
	identity := trustAnchorIdentity(t, root)
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(identity) {
		t.Fatalf("TrustAnchorIdentity() = %q, want 64 lowercase hexadecimal characters", identity)
	}
	if want := anchorIdentityOf(t, root); identity != want {
		t.Errorf("TrustAnchorIdentity() = %s, want %s", identity, want)
	}
	if again := trustAnchorIdentity(t, root); again != identity {
		t.Errorf("TrustAnchorIdentity() = %s on a second call, want %s again", again, identity)
	}
	if reissued := trustAnchorIdentity(t, rootCertDER(t, key, 2)); reissued != identity {
		t.Errorf("identity of the root reissued with another serial = %s, want %s", reissued, identity)
	}
	if other := trustAnchorIdentity(t, selfSignedTLSCert(t).Certificate[0]); other == identity {
		t.Errorf("a root with another key shares identity %s", identity)
	}

	for name, input := range map[string][]byte{"not a certificate": []byte("not a certificate"), "empty": nil} {
		if got, err := TrustAnchorIdentity(input); err == nil {
			t.Errorf("TrustAnchorIdentity(%s) = %q, want an error", name, got)
		}
	}
}

// requireTrustAnchorAPI skips when the loaded library predates Trust Anchor
// Identifiers negotiation.
func requireTrustAnchorAPI(t *testing.T) {
	t.Helper()
	if ffi_lk_client_builder_set_trust_anchor_policy_json == nil {
		t.Skip("trust anchor identifiers are not supported by the loaded lkrequest library")
	}
}

// trustAnchorIdentity returns TrustAnchorIdentity(certDER), failing the test
// on an error.
func trustAnchorIdentity(t *testing.T, certDER []byte) string {
	t.Helper()
	identity, err := TrustAnchorIdentity(certDER)
	if err != nil {
		t.Fatalf("TrustAnchorIdentity() error = %v", err)
	}
	return identity
}

// anchorIdentityOf computes the identity the library gives a root without
// name constraints: SHA-256 over a version label and, for the subject, the
// public key and the absent name constraints in turn, a presence byte, a
// 64-bit big-endian length and the DER contents.
func anchorIdentityOf(t *testing.T, certDER []byte) string {
	t.Helper()
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("ParseCertificate() error = %v", err)
	}

	digest := sha256.New()
	digest.Write([]byte("lkrequest trust anchor v1\x00"))
	for _, field := range [][]byte{cert.RawSubject, cert.RawSubjectPublicKeyInfo} {
		var value asn1.RawValue
		if _, err := asn1.Unmarshal(field, &value); err != nil {
			t.Fatalf("asn1.Unmarshal() error = %v", err)
		}
		digest.Write([]byte{1})
		_ = binary.Write(digest, binary.BigEndian, uint64(len(value.Bytes)))
		digest.Write(value.Bytes)
	}
	digest.Write([]byte{0})
	_ = binary.Write(digest, binary.BigEndian, uint64(0))
	return hex.EncodeToString(digest.Sum(nil))
}

// rootCertDER issues a self-signed root for key, with the given serial and a
// fixed subject.
func rootCertDER(t *testing.T, key *ecdsa.PrivateKey, serial int64) []byte {
	t.Helper()
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(serial),
		Subject:               pkix.Name{CommonName: "lkrequest-go test root"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Duration(serial) * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}
	return der
}

// trustAnchorSource returns the source of the trust anchor identifiers
// extension in a decoded fingerprint.
func trustAnchorSource(t *testing.T, preset string, layers map[string]any) map[string]any {
	t.Helper()
	tlsLayer, _ := layers["tls"].(map[string]any)
	extensions, _ := tlsLayer["extensions"].([]any)
	for _, raw := range extensions {
		ext, _ := raw.(map[string]any)
		if ext["extension_type"] == float64(extTrustAnchorIds) {
			source, _ := ext["source"].(map[string]any)
			return source
		}
	}
	t.Fatalf("%s does not advertise extension 0x%04x", preset, extTrustAnchorIds)
	return nil
}

// profileTrustAnchorIDs returns the trust anchor identifiers preset's TLS
// profile lists, in the listed order.
func profileTrustAnchorIDs(t *testing.T, preset string) []string {
	t.Helper()
	return jsonStrings(trustAnchorSource(t, preset, presetFingerprintWithoutName(t, preset))["ids"])
}

// jsonStrings converts a decoded JSON array of strings.
func jsonStrings(value any) []string {
	items, _ := value.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, fmt.Sprint(item))
	}
	return out
}

// buildTrustAnchorClient builds builder into a client closed when the test
// ends.
func buildTrustAnchorClient(t *testing.T, builder *ClientBuilder) *Client {
	t.Helper()
	client, err := builder.Build()
	if err != nil {
		t.Fatalf("ClientBuilder.Build() error = %v", err)
	}
	t.Cleanup(client.Close)
	return client
}

// newTrustAnchorSession returns a session of client closed when the test ends.
func newTrustAnchorSession(t *testing.T, client *Client) *Session {
	t.Helper()
	session, err := NewSessionBuilder(client).Build()
	if err != nil {
		t.Fatalf("SessionBuilder.Build() error = %v", err)
	}
	t.Cleanup(session.Close)
	return session
}

// trustAnchorOfRequest sends a GET to url and returns the trust_anchor object
// of the response's DiagnosticsJSON, nil when it reports none.
func trustAnchorOfRequest(t *testing.T, session *Session, url string) map[string]any {
	t.Helper()
	resp, err := session.Get(url).SetTimeout(5000).Send()
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	defer resp.Close()
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode())
	}
	return trustAnchorDiagnostics(t, resp.DiagnosticsJSON())
}

// trustAnchorDiagnostics returns the trust_anchor object of a diagnostics
// payload, nil when it is null.
func trustAnchorDiagnostics(t *testing.T, raw string) map[string]any {
	t.Helper()
	var diagnostics map[string]any
	if err := json.Unmarshal([]byte(raw), &diagnostics); err != nil {
		t.Fatalf("unmarshal diagnostics error = %v; payload = %s", err, raw)
	}
	if _, ok := diagnostics["trust_anchor"]; !ok {
		t.Fatalf("diagnostics have no trust_anchor key: %s", raw)
	}
	info, _ := diagnostics["trust_anchor"].(map[string]any)
	return info
}

// sentTrustAnchorIDs sends requests requests from client to a server that
// hangs up after the ClientHello, and returns the trust anchor identifiers of
// every ClientHello the server received, in wire order.
func sentTrustAnchorIDs(t *testing.T, client *Client, requests int) [][]string {
	t.Helper()
	url, hellos := catchClientHellos(t)
	session := newTrustAnchorSession(t, client)
	for i := 0; i < requests; i++ {
		if resp, err := session.Get(url).SetTimeout(5000).Send(); err == nil {
			resp.Close()
			t.Fatalf("request %d succeeded against a server that hangs up after the ClientHello", i)
		}
	}

	var sent [][]string
	for _, record := range hellos() {
		ids, ok := clientHelloTrustAnchorIDs(t, record)
		if !ok {
			t.Fatal("a ClientHello carries no trust_anchors extension")
		}
		sent = append(sent, ids)
	}
	if len(sent) < requests {
		t.Fatalf("the server received %d ClientHellos, want at least %d", len(sent), requests)
	}
	return sent
}

// catchClientHellos runs a TCP server that keeps the first TLS record of
// every connection, the ClientHello, and then hangs up, failing the
// handshake. It returns the URL to request and a function reporting the
// records so far.
func catchClientHellos(t *testing.T) (string, func() [][]byte) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	t.Cleanup(func() { listener.Close() })

	var (
		mu      sync.Mutex
		records [][]byte
	)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
				header := make([]byte, 5)
				if _, err := io.ReadFull(conn, header); err != nil {
					return
				}
				body := make([]byte, binary.BigEndian.Uint16(header[3:]))
				if _, err := io.ReadFull(conn, body); err != nil {
					return
				}
				mu.Lock()
				records = append(records, append(header, body...))
				mu.Unlock()
			}(conn)
		}
	}()

	return "https://" + listener.Addr().String() + "/", func() [][]byte {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(records)
	}
}

// clientHelloTrustAnchorIDs returns the identifiers of the trust_anchors
// extension in a ClientHello record, hex encoded and in wire order; ok is
// false when the ClientHello has no such extension.
func clientHelloTrustAnchorIDs(t *testing.T, record []byte) (ids []string, ok bool) {
	t.Helper()

	// Skip the record and handshake headers, legacy_version and random.
	pos := 5 + 4 + 2 + 32
	next := func(n int) []byte {
		if n < 0 || pos+n > len(record) {
			t.Fatalf("truncated ClientHello record of %d bytes", len(record))
		}
		field := record[pos : pos+n]
		pos += n
		return field
	}
	vector := func(lengthBytes int) []byte {
		n := 0
		for _, b := range next(lengthBytes) {
			n = n<<8 | int(b)
		}
		return next(n)
	}
	vector(1) // legacy_session_id
	vector(2) // cipher_suites
	vector(1) // legacy_compression_methods

	extensions := vector(2)
	for len(extensions) >= 4 {
		kind := binary.BigEndian.Uint16(extensions)
		n := int(binary.BigEndian.Uint16(extensions[2:]))
		if 4+n > len(extensions) {
			t.Fatalf("truncated ClientHello extension 0x%04x", kind)
		}
		data := extensions[4 : 4+n]
		extensions = extensions[4+n:]
		if kind != extTrustAnchorIds {
			continue
		}

		// A two-byte list length, then identifiers with a one-byte length each.
		if len(data) < 2 {
			t.Fatalf("truncated trust_anchors extension %x", data)
		}
		for data = data[2:]; len(data) > 0; {
			n := int(data[0])
			if 1+n > len(data) {
				t.Fatalf("truncated trust anchor identifier in %x", data)
			}
			ids = append(ids, hex.EncodeToString(data[1:1+n]))
			data = data[1+n:]
		}
		return ids, true
	}
	return nil, false
}
