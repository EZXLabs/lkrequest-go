// QUIC/H3 + SOCKS5 UDP integration tests.
//
// These tests require:
// - LKREQUEST_RUN_NETWORK_TESTS=1
// - LKREQUEST_TEST_H3_URL=https://<known-h3-endpoint>
package lkrequest

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	neturl "net/url"
	"os"
	"sync"
	"testing"
	"time"
)

type socksTargetKind string

const (
	socksTargetIP     socksTargetKind = "ip"
	socksTargetDomain socksTargetKind = "domain"
)

type testSocks5UDPProxy struct {
	addr     string
	listener net.Listener
	expected socksTargetKind
	observed chan socksTargetKind

	closeOnce sync.Once
	done      chan struct{}
}

func newTestSocks5UDPProxy(t testing.TB, expected socksTargetKind) *testSocks5UDPProxy {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}

	proxy := &testSocks5UDPProxy{
		addr:     listener.Addr().String(),
		listener: listener,
		expected: expected,
		observed: make(chan socksTargetKind, 1),
		done:     make(chan struct{}),
	}

	go proxy.serve()
	t.Cleanup(func() {
		proxy.Close()
	})
	return proxy
}

func (p *testSocks5UDPProxy) URL(scheme string) string {
	return fmt.Sprintf("%s://%s", scheme, p.addr)
}

func (p *testSocks5UDPProxy) Close() {
	p.closeOnce.Do(func() {
		_ = p.listener.Close()
		<-p.done
	})
}

func (p *testSocks5UDPProxy) awaitObservedKind(t testing.TB) socksTargetKind {
	t.Helper()

	select {
	case kind := <-p.observed:
		return kind
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for SOCKS5 UDP target kind")
		return ""
	}
}

func (p *testSocks5UDPProxy) serve() {
	defer close(p.done)

	conn, err := p.listener.Accept()
	if err != nil {
		return
	}
	defer conn.Close()

	_ = p.handleConn(conn)
}

func (p *testSocks5UDPProxy) handleConn(conn net.Conn) error {
	reader := bufio.NewReader(conn)

	if err := socks5ReadGreeting(reader); err != nil {
		return err
	}
	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		return err
	}

	cmd, err := socks5ReadCommand(reader)
	if err != nil {
		return err
	}
	if cmd != 0x03 {
		_, _ = conn.Write([]byte{0x05, 0x07, 0x00, 0x01, 127, 0, 0, 1, 0, 0})
		return fmt.Errorf("unexpected SOCKS5 command 0x%02x", cmd)
	}

	relay, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		return err
	}
	defer relay.Close()

	outbound4, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero})
	if err != nil {
		return err
	}
	defer outbound4.Close()

	var outbound6 *net.UDPConn
	if udp6, udp6Err := net.ListenUDP("udp6", &net.UDPAddr{IP: net.IPv6unspecified}); udp6Err == nil {
		outbound6 = udp6
		defer outbound6.Close()
	}

	relayAddr, ok := relay.LocalAddr().(*net.UDPAddr)
	if !ok {
		return fmt.Errorf("unexpected relay local addr %T", relay.LocalAddr())
	}
	reply := []byte{0x05, 0x00, 0x00, 0x01}
	reply = append(reply, relayAddr.IP.To4()...)
	reply = append(reply, byte(relayAddr.Port>>8), byte(relayAddr.Port))
	if _, err := conn.Write(reply); err != nil {
		return err
	}

	var (
		closeOnce   sync.Once
		clientAddr  *net.UDPAddr
		clientMu    sync.RWMutex
		observeOnce sync.Once
		done        = make(chan struct{})
	)

	closeAll := func() {
		closeOnce.Do(func() {
			close(done)
			_ = relay.Close()
			_ = outbound4.Close()
			if outbound6 != nil {
				_ = outbound6.Close()
			}
			_ = conn.Close()
		})
	}

	sendObserved := func(kind socksTargetKind) {
		observeOnce.Do(func() {
			select {
			case p.observed <- kind:
			default:
			}
		})
	}

	go p.forwardRemoteToClient(outbound4, relay, &clientAddr, &clientMu, done)
	if outbound6 != nil {
		go p.forwardRemoteToClient(outbound6, relay, &clientAddr, &clientMu, done)
	}

	go func() {
		buf := make([]byte, 64*1024)
		for {
			n, from, err := relay.ReadFromUDP(buf)
			if err != nil {
				return
			}

			clientMu.RLock()
			currentClient := clientAddr
			clientMu.RUnlock()
			if currentClient == nil || sameUDPAddr(currentClient, from) {
				clientMu.Lock()
				clientAddr = cloneUDPAddr(from)
				clientMu.Unlock()

				target, payload, kind, err := parseSocks5UDPFrame(buf[:n])
				if err != nil {
					closeAll()
					return
				}
				sendObserved(kind)
				if kind != p.expected {
					closeAll()
					return
				}

				addr := target
				if addr.IP.To4() != nil {
					if _, err := outbound4.WriteToUDP(payload, addr); err != nil {
						closeAll()
						return
					}
					continue
				}
				if outbound6 == nil {
					closeAll()
					return
				}
				if _, err := outbound6.WriteToUDP(payload, addr); err != nil {
					closeAll()
					return
				}
			}
		}
	}()

	_, _ = io.Copy(io.Discard, conn)
	closeAll()
	return nil
}

func (p *testSocks5UDPProxy) forwardRemoteToClient(
	outbound *net.UDPConn,
	relay *net.UDPConn,
	clientAddr **net.UDPAddr,
	clientMu *sync.RWMutex,
	done <-chan struct{},
) {
	buf := make([]byte, 64*1024)
	for {
		n, from, err := outbound.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-done:
				return
			default:
				return
			}
		}

		clientMu.RLock()
		currentClient := *clientAddr
		clientMu.RUnlock()
		if currentClient == nil {
			continue
		}

		frame, err := encodeSocks5UDPFrame(from, buf[:n])
		if err != nil {
			continue
		}
		_, _ = relay.WriteToUDP(frame, currentClient)
	}
}

func socks5ReadGreeting(reader *bufio.Reader) error {
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return err
	}
	if header[0] != 0x05 {
		return fmt.Errorf("unexpected SOCKS version %d", header[0])
	}

	methods := make([]byte, int(header[1]))
	_, err := io.ReadFull(reader, methods)
	return err
}

func socks5ReadCommand(reader *bufio.Reader) (byte, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(reader, header); err != nil {
		return 0, err
	}
	if header[0] != 0x05 {
		return 0, fmt.Errorf("unexpected SOCKS version %d", header[0])
	}
	if err := socks5ReadAddress(reader, header[3]); err != nil {
		return 0, err
	}
	return header[1], nil
}

func socks5ReadAddress(reader *bufio.Reader, atyp byte) error {
	var addrLen int
	switch atyp {
	case 0x01:
		addrLen = 4
	case 0x04:
		addrLen = 16
	case 0x03:
		size, err := reader.ReadByte()
		if err != nil {
			return err
		}
		addrLen = int(size)
	default:
		return fmt.Errorf("unsupported SOCKS address type 0x%02x", atyp)
	}

	buf := make([]byte, addrLen+2)
	_, err := io.ReadFull(reader, buf)
	return err
}

func parseSocks5UDPFrame(frame []byte) (*net.UDPAddr, []byte, socksTargetKind, error) {
	if len(frame) < 4 {
		return nil, nil, "", fmt.Errorf("frame too short")
	}
	if frame[2] != 0x00 {
		return nil, nil, "", fmt.Errorf("fragmented SOCKS5 UDP frame is unsupported")
	}

	switch frame[3] {
	case 0x01:
		if len(frame) < 10 {
			return nil, nil, "", fmt.Errorf("IPv4 frame too short")
		}
		ip := net.IPv4(frame[4], frame[5], frame[6], frame[7])
		port := int(frame[8])<<8 | int(frame[9])
		return &net.UDPAddr{IP: ip, Port: port}, frame[10:], socksTargetIP, nil
	case 0x04:
		if len(frame) < 22 {
			return nil, nil, "", fmt.Errorf("IPv6 frame too short")
		}
		ip := make(net.IP, net.IPv6len)
		copy(ip, frame[4:20])
		port := int(frame[20])<<8 | int(frame[21])
		return &net.UDPAddr{IP: ip, Port: port}, frame[22:], socksTargetIP, nil
	case 0x03:
		if len(frame) < 7 {
			return nil, nil, "", fmt.Errorf("domain frame too short")
		}
		size := int(frame[4])
		portOffset := 5 + size
		if len(frame) < portOffset+2 {
			return nil, nil, "", fmt.Errorf("domain frame missing port")
		}
		host := string(frame[5:portOffset])
		port := int(frame[portOffset])<<8 | int(frame[portOffset+1])
		ip, err := resolveProxyUDPHost(host)
		if err != nil {
			return nil, nil, "", err
		}
		return &net.UDPAddr{IP: ip, Port: port}, frame[portOffset+2:], socksTargetDomain, nil
	default:
		return nil, nil, "", fmt.Errorf("unsupported UDP address type 0x%02x", frame[3])
	}
}

func resolveProxyUDPHost(host string) (net.IP, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		if v4 := addr.IP.To4(); v4 != nil {
			return v4, nil
		}
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no IP addresses for %s", host)
	}
	return addrs[0].IP, nil
}

func encodeSocks5UDPFrame(addr *net.UDPAddr, payload []byte) ([]byte, error) {
	frame := []byte{0x00, 0x00, 0x00}
	if v4 := addr.IP.To4(); v4 != nil {
		frame = append(frame, 0x01)
		frame = append(frame, v4...)
	} else if v6 := addr.IP.To16(); v6 != nil {
		frame = append(frame, 0x04)
		frame = append(frame, v6...)
	} else {
		return nil, fmt.Errorf("unsupported remote IP %v", addr.IP)
	}
	frame = append(frame, byte(addr.Port>>8), byte(addr.Port))
	frame = append(frame, payload...)
	return frame, nil
}

func sameUDPAddr(a, b *net.UDPAddr) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Port == b.Port && a.IP.Equal(b.IP)
}

func cloneUDPAddr(addr *net.UDPAddr) *net.UDPAddr {
	if addr == nil {
		return nil
	}
	ip := make(net.IP, len(addr.IP))
	copy(ip, addr.IP)
	return &net.UDPAddr{IP: ip, Port: addr.Port, Zone: addr.Zone}
}

func requireH3NetworkURL(t testing.TB) string {
	t.Helper()
	requireNetworkTests(t)
	if !FeatureSupported("quic-h3") {
		t.Skip("quic-h3 is not supported by the loaded lkrequest library")
	}

	targetURL := os.Getenv("LKREQUEST_TEST_H3_URL")
	if targetURL == "" {
		t.Skip("set LKREQUEST_TEST_H3_URL to run QUIC/H3 SOCKS5 UDP integration tests")
	}

	parsed, err := neturl.Parse(targetURL)
	if err != nil {
		t.Fatalf("invalid LKREQUEST_TEST_H3_URL %q: %v", targetURL, err)
	}
	if parsed.Scheme != "https" || parsed.Host == "" {
		t.Fatalf("LKREQUEST_TEST_H3_URL = %q, want an absolute https URL", targetURL)
	}
	return targetURL
}

func newH3TestClient(t testing.TB) *Client {
	t.Helper()

	client, err := NewClientBuilder().
		SetPreset("chrome_144").
		SetTimeoutTotal(20_000).
		Build()
	if err != nil {
		t.Fatalf("ClientBuilder.Build() error = %v", err)
	}
	return client
}

func TestE2EHTTP3OverLocalSocks5UDPProxy(t *testing.T) {
	targetURL := requireH3NetworkURL(t)
	proxy := newTestSocks5UDPProxy(t, socksTargetIP)

	client := newH3TestClient(t)
	t.Cleanup(client.Close)

	session, err := NewSessionBuilder(client).
		SetHTTP3Only().
		SetProxy(proxy.URL("socks5")).
		Build()
	if err != nil {
		t.Fatalf("SessionBuilder.Build() error = %v", err)
	}
	t.Cleanup(session.Close)

	req := mustNewRequest(t, session, "GET", targetURL).SetTimeout(20_000)
	resp, err := req.Send()
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	cleanupResponse(t, resp)

	if got := resp.StatusCode(); got != 200 {
		t.Fatalf("StatusCode() = %d, want 200", got)
	}
	if got := resp.Version(); got != HttpVersion3 {
		t.Fatalf("Version() = %s, want %s", got, HttpVersion3)
	}
	if body, err := resp.Text(); err != nil || body == "" {
		t.Fatalf("Text() = (%q, %v), want non-empty body and nil error", body, err)
	}

	if got := proxy.awaitObservedKind(t); got != socksTargetIP {
		t.Fatalf("observed SOCKS5 UDP target kind = %q, want %q", got, socksTargetIP)
	}
}

func TestE2EHTTP3OverLocalSocks5HUDPProxy(t *testing.T) {
	targetURL := requireH3NetworkURL(t)
	proxy := newTestSocks5UDPProxy(t, socksTargetDomain)

	client := newH3TestClient(t)
	t.Cleanup(client.Close)

	session, err := NewSessionBuilder(client).
		SetHTTP3Only().
		Build()
	if err != nil {
		t.Fatalf("SessionBuilder.Build() error = %v", err)
	}
	t.Cleanup(session.Close)

	req := mustNewRequest(t, session, "GET", targetURL).
		SetProxy(proxy.URL("socks5h")).
		SetTimeout(20_000)

	resp, err := req.Send()
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	cleanupResponse(t, resp)

	if got := resp.StatusCode(); got != 200 {
		t.Fatalf("StatusCode() = %d, want 200", got)
	}
	if got := resp.Version(); got != HttpVersion3 {
		t.Fatalf("Version() = %s, want %s", got, HttpVersion3)
	}
	if body, err := resp.Text(); err != nil || body == "" {
		t.Fatalf("Text() = (%q, %v), want non-empty body and nil error", body, err)
	}

	if got := proxy.awaitObservedKind(t); got != socksTargetDomain {
		t.Fatalf("observed SOCKS5 UDP target kind = %q, want %q", got, socksTargetDomain)
	}
}
