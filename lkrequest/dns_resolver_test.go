package lkrequest

import (
	"fmt"
	"net"
	"net/url"
	"sync/atomic"
	"testing"
)

// testDNSResolver resolves every hostname to a fixed address and counts calls.
type testDNSResolver struct {
	addr   string
	called *atomic.Int64
}

func (r testDNSResolver) Resolve(host string, port uint16) ([]string, error) {
	r.called.Add(1)
	return []string{r.addr}, nil
}

// TestSetDNSResolverRoundTrip verifies the full custom DNS resolver path.
//
// It covers two previous blind spots:
//  1. The C signature of lk_client_builder_set_dns_resolver passes a struct by
//     value. Go previously passed a pointer, causing an ABI mismatch on
//     linux/amd64 where SysV structs larger than 16 bytes are passed on the
//     stack, with no test coverage to expose it.
//  2. The resolver callback synchronously re-enters Go from an internal library
//     thread during the request, also exercising the send path's stack-move
//     protection (see ffiOutCellPool).
func TestSetDNSResolverRoundTrip(t *testing.T) {
	server := newLocalFeatureServer(t)

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}

	var called atomic.Int64
	resolver := testDNSResolver{
		addr:   net.JoinHostPort("127.0.0.1", u.Port()),
		called: &called,
	}

	client, err := NewClientBuilder().SetDNSResolver(resolver).Build()
	if err != nil {
		// Before the Linux fix, the setter received a mislaid struct and Build
		// surfaced the registration failure.
		t.Fatalf("ClientBuilder.SetDNSResolver().Build() error = %v", err)
	}
	defer client.Close()

	session, err := NewSession(client)
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	defer session.Close()

	// Use a synthetic hostname so only the custom resolver can route it to the
	// local server.
	target := fmt.Sprintf("http://lkgo-dns-probe.internal:%s/echo", u.Port())
	resp := mustSendRequest(t, mustNewRequest(t, session, "GET", target))
	cleanupResponse(t, resp)

	if called.Load() == 0 {
		t.Fatal("custom DNS resolver was never invoked")
	}
}
