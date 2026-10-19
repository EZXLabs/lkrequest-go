package lkrequest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"io"
	"math/big"
	"net"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"
)

// TestH2PriorityWeightReachesTheWire checks SetH2PriorityWeight end to end on
// Firefox 156, whose navigation HEADERS carry wire weight 41 and whose first
// request opens stream 3. The weight a request sets is the one its HEADERS
// frame carries, on the request that opens a connection as on later ones,
// buffered or streaming; a request that sets none goes back to the profile's;
// both ends of the wire range pass through; and Clone keeps the weight.
func TestH2PriorityWeightReachesTheWire(t *testing.T) {
	if ffi_lk_request_set_h2_priority_weight == nil {
		t.Skip("the loaded lkrequest library predates lk_request_set_h2_priority_weight")
	}
	probe := startH2Probe(t)

	client, err := NewClientBuilder().SetPreset("firefox_156").SetVerify(false).Build()
	if err != nil {
		t.Fatalf("NewClientBuilder with firefox_156 preset error = %v", err)
	}
	t.Cleanup(client.Close)
	newSession := func() *Session {
		t.Helper()
		session, err := NewSession(client)
		if err != nil {
			t.Fatalf("NewSession() error = %v", err)
		}
		t.Cleanup(session.Close)
		return session
	}
	newRequest := func(session *Session, path string) *Request {
		t.Helper()
		req, err := NewRequest(session, "GET", probe.url(path))
		if err != nil {
			t.Fatalf("NewRequest(%s) error = %v", path, err)
		}
		return req
	}
	send := func(req *Request) {
		t.Helper()
		resp, err := req.Send()
		if err != nil {
			t.Fatalf("Send() error = %v", err)
		}
		defer resp.Close()
		if got := resp.StatusCode(); got != http.StatusOK {
			t.Fatalf("StatusCode() = %d, want %d", got, http.StatusOK)
		}
		if got := resp.NegotiatedVersion(); got != HttpVersion2 {
			t.Fatalf("NegotiatedVersion() = %v, want %v", got, HttpVersion2)
		}
	}

	session := newSession()
	send(newRequest(session, "/opens").SetH2PriorityWeight(21))
	send(newRequest(session, "/profile"))
	send(newRequest(session, "/lowest").SetH2PriorityWeight(0))
	send(newRequest(session, "/highest").SetH2PriorityWeight(255))
	original := newRequest(session, "/clone").SetH2PriorityWeight(7)
	clone, err := original.Clone()
	original.release()
	if err != nil {
		t.Fatalf("Clone() error = %v", err)
	}
	send(clone)

	// A second session opens a connection of its own, with a streaming request.
	stream, err := newRequest(newSession(), "/stream").SetH2PriorityWeight(21).SendStreaming()
	if err != nil {
		t.Fatalf("SendStreaming() error = %v", err)
	}
	_, err = io.Copy(io.Discard, stream)
	stream.Close()
	if err != nil {
		t.Fatalf("read streaming body error = %v", err)
	}

	want := []h2Headers{
		{stream: 3, weight: 21},
		{stream: 5, weight: 41},
		{stream: 7, weight: 0},
		{stream: 9, weight: 255},
		{stream: 11, weight: 7},
		{stream: 3, weight: 21},
	}
	if got := probe.recorded(); !reflect.DeepEqual(got, want) {
		t.Fatalf("HEADERS frames seen = %+v, want %+v", got, want)
	}
}

// h2Headers is what h2Probe notes about one HEADERS frame: the stream it
// opened and the weight in its priority field, -1 when it carried none.
type h2Headers struct {
	stream uint32
	weight int
}

// h2Probe is an HTTP/2 server over TLS that records every HEADERS frame it
// receives and answers each with an empty 200. An ordinary HTTP/2 server does
// not expose the priority field, and reading it takes no more of the protocol
// than frame headers: no HPACK decoding.
type h2Probe struct {
	listener net.Listener
	mu       sync.Mutex
	headers  []h2Headers
}

const (
	h2ClientPreface  = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
	h2FrameHeaders   = 0x1
	h2FrameSettings  = 0x4
	h2FramePing      = 0x6
	h2FrameGoAway    = 0x7
	h2FlagAck        = 0x1 // SETTINGS and PING
	h2FlagEndStream  = 0x1 // HEADERS
	h2FlagEndHeaders = 0x4
	h2FlagPadded     = 0x8
	h2FlagPriority   = 0x20
	// hpackStatus200 is the HPACK static-table entry for ":status: 200".
	hpackStatus200 = 0x88
)

// startH2Probe starts an h2Probe on a loopback port, stopped at test cleanup.
func startH2Probe(t *testing.T) *h2Probe {
	t.Helper()

	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{selfSignedTLSCert(t)},
		NextProtos:   []string{"h2"},
	})
	if err != nil {
		t.Fatalf("tls.Listen() error = %v", err)
	}
	t.Cleanup(func() { listener.Close() })

	probe := &h2Probe{listener: listener}
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
func (p *h2Probe) url(path string) string {
	return "https://" + p.listener.Addr().String() + path
}

// recorded returns the HEADERS frames seen so far, in arrival order.
func (p *h2Probe) recorded() []h2Headers {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]h2Headers(nil), p.headers...)
}

// serve speaks just enough HTTP/2 on conn for a client to complete requests.
func (p *h2Probe) serve(conn net.Conn) {
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
			p.record(stream, flags, payload)
			err = writeH2Frame(conn, h2FrameHeaders, h2FlagEndStream|h2FlagEndHeaders, stream, []byte{hpackStatus200})
		case h2FrameGoAway:
			return
		}
		if err != nil {
			return
		}
	}
}

// record notes a HEADERS frame and the weight in its priority field, which
// follows the optional pad length and the 4-byte stream dependency.
func (p *h2Probe) record(stream uint32, flags byte, payload []byte) {
	weight := -1
	if flags&h2FlagPriority != 0 {
		offset := 4
		if flags&h2FlagPadded != 0 {
			offset++
		}
		if offset < len(payload) {
			weight = int(payload[offset])
		}
	}

	p.mu.Lock()
	p.headers = append(p.headers, h2Headers{stream: stream, weight: weight})
	p.mu.Unlock()
}

// writeH2Frame writes one HTTP/2 frame.
func writeH2Frame(w io.Writer, frameType, flags byte, stream uint32, payload []byte) error {
	frame := make([]byte, 9+len(payload))
	frame[0], frame[1], frame[2] = byte(len(payload)>>16), byte(len(payload)>>8), byte(len(payload))
	frame[3], frame[4] = frameType, flags
	binary.BigEndian.PutUint32(frame[5:], stream)
	copy(frame[9:], payload)
	_, err := w.Write(frame)
	return err
}

// selfSignedTLSCert returns a throwaway certificate for 127.0.0.1, for local
// TLS servers that a client built with SetVerify(false) connects to.
func selfSignedTLSCert(t *testing.T) tls.Certificate {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}
