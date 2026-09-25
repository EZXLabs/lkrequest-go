package lkrequest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"
)

// newMasqueConfig creates a MasqueConfig that is closed when the test ends.
// Skipped when the loaded library predates the MASQUE symbols or was built
// without the masque feature — the official release builds are the latter.
func newMasqueConfig(t *testing.T) *MasqueConfig {
	t.Helper()
	if ffi_lk_masque_config_new == nil {
		t.Skip("MASQUE configuration is not supported by the loaded lkrequest library")
	}

	config, err := NewMasqueConfig()
	if err != nil {
		t.Skipf("the loaded lkrequest library was built without MASQUE: %v", err)
	}
	t.Cleanup(config.Close)
	return config
}

// TestMasqueUnavailableIsExplicit covers libraries built without the masque
// feature, which is how the official libraries ship: NewMasqueConfig must say
// why it failed, and a masque:// URL must be refused rather than parsed into a
// configuration no request could use.
func TestMasqueUnavailableIsExplicit(t *testing.T) {
	if ffi_lk_masque_config_new == nil {
		t.Skip("MASQUE configuration is not supported by the loaded lkrequest library")
	}

	config, err := NewMasqueConfig()
	if err == nil {
		config.Close()
		t.Skip("the loaded lkrequest library was built with MASQUE; covered by the tests below")
	}
	if !strings.Contains(err.Error(), "masque feature") {
		t.Errorf("NewMasqueConfig() error = %v, want it to name the missing masque feature", err)
	}

	if proxy, err := ParseProxyConfig("masque://proxy.example:443"); err == nil {
		proxy.Close()
		t.Error("ParseProxyConfig accepted a masque:// URL on a library without MASQUE")
	}
}

// TestMasqueConsumersRejectNilConfig checks that every consumer reports a
// missing MasqueConfig at Build — which is what a caller ignoring
// NewMasqueConfig's error on a library without MASQUE would hand them.
func TestMasqueConsumersRejectNilConfig(t *testing.T) {
	if ffi_lk_client_builder_set_masque_config == nil {
		t.Skip("MASQUE configuration is not supported by the loaded lkrequest library")
	}

	if _, err := NewClientBuilder().SetMasqueConfig(nil).Build(); !errors.Is(err, ErrNilHandle) {
		t.Errorf("ClientBuilder.SetMasqueConfig(nil) Build error = %v, want ErrNilHandle", err)
	}
	if _, err := NewProxyPoolBuilder().AddProxy("http://proxy.example:8080").SetMasqueTunnelProbe(nil).Build(); !errors.Is(err, ErrNilHandle) {
		t.Errorf("ProxyPoolBuilder.SetMasqueTunnelProbe(nil) Build error = %v, want ErrNilHandle", err)
	}

	client, err := NewClientBuilder().Build()
	if err != nil {
		t.Fatalf("NewClientBuilder().Build() error = %v", err)
	}
	t.Cleanup(client.Close)
	if _, err := NewSessionPoolBuilder(client).AddProxy("http://proxy.example:8080").SetMasqueTunnelProbe(nil).Build(); !errors.Is(err, ErrNilHandle) {
		t.Errorf("SessionPoolBuilder.SetMasqueTunnelProbe(nil) Build error = %v, want ErrNilHandle", err)
	}
}

// TestMasqueConfigAppliesToClient drives every setter with a valid value — the
// private-CA shape: a CA in both encodings, the OS store off, a certificate
// name override — and checks the client accepts the result. It also checks a
// masque:// URL parses, with the port defaulting to 443.
func TestMasqueConfigAppliesToClient(t *testing.T) {
	config := newMasqueConfig(t)
	certPEM, certDER := selfSignedCA(t)

	config.AddCACertsPEM(certPEM).
		AddCACertDER(certDER).
		SetUseNativeCerts(false).
		SetServerName("proxy.internal").
		SetTunnelIdleTimeout(0).
		SetMaxIdleTunnels(0).
		SetVerify(true)

	client, err := NewClientBuilder().SetMasqueConfig(config).Build()
	if err != nil {
		t.Fatalf("ClientBuilder.SetMasqueConfig Build error = %v", err)
	}
	client.Close()

	proxy := newProxyConfig(t, "masque://proxy.example")
	if got, err := proxy.URL(); err != nil || got != "masque://proxy.example:443" {
		t.Fatalf("masque proxy URL() = %q, %v; want the port to default to 443", got, err)
	}
}

// TestMasqueConfigRejectsBadInput checks the two inputs the library validates
// up front: a PEM bundle without a certificate (a path passed by mistake would
// otherwise leave the trust store silently unchanged) and a blank server name.
func TestMasqueConfigRejectsBadInput(t *testing.T) {
	for name, apply := range map[string]func(*MasqueConfig){
		"pem without certificate": func(c *MasqueConfig) { c.AddCACertsPEM([]byte("/etc/ssl/proxy-ca.pem")) },
		"blank server name":       func(c *MasqueConfig) { c.SetServerName("   ") },
	} {
		t.Run(name, func(t *testing.T) {
			config := newMasqueConfig(t)
			apply(config)
			if client, err := NewClientBuilder().SetMasqueConfig(config).Build(); err == nil {
				client.Close()
				t.Fatal("client built from a rejected MasqueConfig; want the config's error")
			}
		})
	}
}

// TestMasqueTunnelProbeNeedsHealthCheck checks both pools: a tunnel probe with
// no health check would never run, so Build refuses it rather than silently
// never probing; with a health check it builds, in either call order.
func TestMasqueTunnelProbeNeedsHealthCheck(t *testing.T) {
	config := newMasqueConfig(t)
	const proxy = "masque://127.0.0.1:1"

	if _, err := NewProxyPoolBuilder().AddProxy(proxy).SetMasqueTunnelProbe(config).Build(); err == nil {
		t.Error("ProxyPoolBuilder with a tunnel probe and no health check built; want an error")
	}
	pool, err := NewProxyPoolBuilder().AddProxy(proxy).
		SetMasqueTunnelProbe(config).
		SetHealthCheck("example.com", 443, 60_000, 1_000).
		Build()
	if err != nil {
		t.Fatalf("ProxyPoolBuilder with a tunnel probe and health check error = %v", err)
	}
	pool.Close()

	client, err := NewClientBuilder().SetMasqueConfig(config).Build()
	if err != nil {
		t.Fatalf("NewClientBuilder().Build() error = %v", err)
	}
	t.Cleanup(client.Close)

	if _, err := NewSessionPoolBuilder(client).AddProxy(proxy).SetMasqueTunnelProbe(config).Build(); err == nil {
		t.Error("SessionPoolBuilder with a tunnel probe and no health check built; want an error")
	}
	sessionPool, err := NewSessionPoolBuilder(client).AddProxy(proxy).
		SetMasqueTunnelProbe(config).
		SetHealthCheck("example.com", 443, 60_000, 1_000).
		Build()
	if err != nil {
		t.Fatalf("SessionPoolBuilder with a tunnel probe and health check error = %v", err)
	}
	sessionPool.Close()
}

// selfSignedCA returns a throwaway CA certificate in PEM and DER form.
func selfSignedCA(t *testing.T) ([]byte, []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "lkrequest-go test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), der
}
