// Regression tests for multi-address DNS fallback (upstream adc0145,
// "fix(network): retry resolved addresses for TCP and SOCKS5").
//
// Before that fix the connector kept only the first resolved address and
// dropped the rest, so a dead first candidate (e.g. an IPv6 address without a
// route) failed the whole request even when a reachable address was next in
// the list. These tests mirror the upstream unit tests
// connect_falls_back_when_first_dns_address_is_unreachable and
// socks5_local_dns_retries_next_address through the public Go API.
package lkrequest

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"
)

// skipWithoutMultiAddressRetry skips the test when the platform library
// predates the upstream multi-address retry fix. The fix changed behaviour
// without adding symbols and without bumping the reported crate version, so
// the only reliable probe is an error-message literal that release builds of
// the fixed library always embed. CI currently fetches the latest tagged
// library, which may be older than the fix; these tests then skip and start
// running for real once a post-fix tag is synced.
func skipWithoutMultiAddressRetry(t testing.TB) {
	t.Helper()

	const marker = "address attempt exceeded its serial fallback window"

	names := map[string]string{
		"darwin":  "liblkrequest_ffi.dylib",
		"linux":   "liblkrequest_ffi.so",
		"windows": "lkrequest_ffi.dll",
	}
	name, ok := names[runtime.GOOS]
	if !ok {
		t.Skipf("no embedded library mapping for GOOS %q", runtime.GOOS)
	}

	path := filepath.Join("lib", runtime.GOOS+"_"+runtime.GOARCH, name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("cannot inspect platform library %s: %v", path, err)
	}
	if !bytes.Contains(data, []byte(marker)) {
		t.Skip("platform library predates multi-address retry (upstream adc0145)")
	}
}

// multiAddrResolver returns a fixed address list for one target host and
// resolves every other host (e.g. a proxy address) to itself.
type multiAddrResolver struct {
	targetHost  string
	targetAddrs []string
}

func (r multiAddrResolver) Resolve(host string, port uint16) ([]string, error) {
	if host == r.targetHost {
		return r.targetAddrs, nil
	}
	return []string{net.JoinHostPort(host, strconv.Itoa(int(port)))}, nil
}

// TestDNSMultiAddressFallback verifies that when the resolver returns several
// addresses and the first one is unreachable, the request succeeds via the
// next address instead of failing outright.
func TestDNSMultiAddressFallback(t *testing.T) {
	skipWithoutMultiAddressRetry(t)

	server := newLocalFeatureServer(t)

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}

	// Port 1 on loopback is refused immediately, which makes the failure of
	// the first candidate deterministic (unlike [::1], which depends on the
	// host's IPv6 setup).
	resolver := multiAddrResolver{
		targetHost: "lkgo-multi-addr-probe.internal",
		targetAddrs: []string{
			"127.0.0.1:1",
			net.JoinHostPort("127.0.0.1", u.Port()),
		},
	}

	client, err := NewClientBuilder().SetDNSResolver(resolver).Build()
	if err != nil {
		t.Fatalf("ClientBuilder.SetDNSResolver().Build() error = %v", err)
	}
	defer client.Close()

	session, err := NewSession(client)
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	defer session.Close()

	target := fmt.Sprintf("http://lkgo-multi-addr-probe.internal:%s/echo", u.Port())
	req, err := NewRequest(session, "GET", target)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	resp, err := req.Send()
	if err != nil {
		t.Fatalf("Send() error = %v, want fallback to the second address", err)
	}
	cleanupResponse(t, resp)

	if got := resp.StatusCode(); got != 200 {
		t.Fatalf("StatusCode() = %d, want 200", got)
	}
}

// socks5AcceptConnect performs the server side of a SOCKS5 handshake on conn
// up to and including the CONNECT request, and returns the address type the
// client sent. The reply is left to the caller.
func socks5AcceptConnect(conn net.Conn) (byte, error) {
	reader := bufio.NewReader(conn)

	if err := socks5ReadGreeting(reader); err != nil {
		return 0, err
	}
	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		return 0, err
	}

	header := make([]byte, 4)
	if _, err := io.ReadFull(reader, header); err != nil {
		return 0, err
	}
	if header[0] != 0x05 {
		return 0, fmt.Errorf("unexpected SOCKS version %d", header[0])
	}
	if header[1] != 0x01 {
		return 0, fmt.Errorf("unexpected SOCKS5 command 0x%02x", header[1])
	}
	if err := socks5ReadAddress(reader, header[3]); err != nil {
		return 0, err
	}
	return header[3], nil
}

// TestSOCKS5LocalDNSRetriesNextAddress verifies that with a socks5:// proxy
// (local target resolution) a retryable SOCKS5 reply for the first resolved
// address triggers a fresh proxy connection for the next address instead of
// failing the request.
func TestSOCKS5LocalDNSRetriesNextAddress(t *testing.T) {
	skipWithoutMultiAddressRetry(t)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	atypCh := make(chan byte, 2)
	serverErr := make(chan error, 1)

	go func() {
		serverErr <- func() error {
			// First proxy connection: report "network unreachable" (0x03), a
			// retryable reply, and close.
			conn, err := listener.Accept()
			if err != nil {
				return err
			}
			atyp, err := socks5AcceptConnect(conn)
			if err != nil {
				conn.Close()
				return err
			}
			atypCh <- atyp
			_, err = conn.Write([]byte{0x05, 0x03, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
			conn.Close()
			if err != nil {
				return err
			}

			// Second proxy connection: accept the CONNECT and act as the HTTP
			// origin on the same stream.
			conn, err = listener.Accept()
			if err != nil {
				return err
			}
			defer conn.Close()
			reader := bufio.NewReader(conn)
			if err := socks5ReadGreeting(reader); err != nil {
				return err
			}
			if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
				return err
			}
			header := make([]byte, 4)
			if _, err := io.ReadFull(reader, header); err != nil {
				return err
			}
			if header[1] != 0x01 {
				return fmt.Errorf("unexpected SOCKS5 command 0x%02x", header[1])
			}
			if err := socks5ReadAddress(reader, header[3]); err != nil {
				return err
			}
			atypCh <- header[3]
			if _, err := conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}); err != nil {
				return err
			}

			// Consume the HTTP request head, then answer with a fixed 200.
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				if line == "\r\n" || line == "\n" {
					break
				}
			}
			response := "HTTP/1.1 200 OK\r\nContent-Length: 2\r\nConnection: close\r\n\r\nok"
			_, err = conn.Write([]byte(response))
			return err
		}()
	}()

	// The resolver hands out an IPv6 candidate first and an IPv4 candidate
	// second; neither is dialed for real (the proxy replies without connecting
	// out), so this stays independent of the host's IPv6 setup.
	resolver := multiAddrResolver{
		targetHost: "lkgo-socks5-multi-addr.internal",
		targetAddrs: []string{
			"[::1]:80",
			"127.0.0.1:80",
		},
	}

	client, err := NewClientBuilder().SetDNSResolver(resolver).Build()
	if err != nil {
		t.Fatalf("ClientBuilder.SetDNSResolver().Build() error = %v", err)
	}
	defer client.Close()

	session, err := NewSessionBuilder(client).
		SetProxy("socks5://" + listener.Addr().String()).
		Build()
	if err != nil {
		t.Fatalf("SessionBuilder.SetProxy().Build() error = %v", err)
	}
	defer session.Close()

	req, err := NewRequest(session, "GET", "http://lkgo-socks5-multi-addr.internal/")
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	resp, err := req.Send()
	if err != nil {
		t.Fatalf("Send() error = %v, want retry with the next resolved address", err)
	}
	cleanupResponse(t, resp)

	if got := resp.StatusCode(); got != 200 {
		t.Fatalf("StatusCode() = %d, want 200", got)
	}

	select {
	case err := <-serverErr:
		if err != nil {
			t.Fatalf("SOCKS5 test server error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the SOCKS5 test server to finish")
	}

	var atyps []byte
	for len(atyps) < 2 {
		select {
		case atyp := <-atypCh:
			atyps = append(atyps, atyp)
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for CONNECT attempts, got %v", atyps)
		}
	}
	if atyps[0] != 0x04 || atyps[1] != 0x01 {
		t.Fatalf("CONNECT address types = [0x%02x 0x%02x], want [0x04 0x01] (IPv6 then IPv4)", atyps[0], atyps[1])
	}
}
