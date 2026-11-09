package lkrequest

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// http1CaseBody is the upload every request sends. It reads like request
// heads, Host lines included, so a Host rewrite that strayed past the head
// into the body would show up as a corrupted upload.
const http1CaseBody = "host: body\r\n\r\nPOST / HTTP/1.1\r\nhost: body\r\n\r\n"

// TestHTTP1HeaderCaseReachesTheWire checks the header-name casing an HTTP/1.1
// request puts on the wire, read from the raw bytes before any HTTP parsing:
// Title-Case by default or with SetHTTP1TitleCaseHeaders(true), lowercase but
// for Host with false. Every case runs over plain HTTP and TLS, buffered and
// streaming, on a connection that is reused, on new connections, and on a
// preconnected one. All but the last case pin the session to HTTP/1.1; the
// last leaves it free, so the TLS server's ALPN is what settles on HTTP/1.1.
// Values keep the spelling the caller gave them either way.
func TestHTTP1HeaderCaseReachesTheWire(t *testing.T) {
	for _, tc := range []struct {
		name       string
		titleCase  *bool
		preconnect bool
		http1Only  bool
	}{
		{name: "lowercase", titleCase: ptrTo(false), http1Only: true},
		{name: "lowercase preconnected", titleCase: ptrTo(false), preconnect: true, http1Only: true},
		{name: "title case by default", http1Only: true},
		{name: "explicit title case preconnected", titleCase: ptrTo(true), preconnect: true, http1Only: true},
		{name: "lowercase without http1 only", titleCase: ptrTo(false)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.titleCase != nil && ffi_lk_client_builder_set_http1_title_case_headers == nil {
				t.Skip("HTTP/1.1 header casing is not supported by the loaded lkrequest library")
			}
			for _, useTLS := range []bool{false, true} {
				for _, streaming := range []bool{false, true} {
					t.Run(fmt.Sprintf("tls=%v streaming=%v", useTLS, streaming), func(t *testing.T) {
						checkHTTP1HeaderCase(t, tc.titleCase, tc.preconnect, tc.http1Only, useTLS, streaming)
					})
				}
			}
		})
	}
}

// checkHTTP1HeaderCase sends two requests through one session and checks the
// heads the server received. Buffered requests share one keep-alive
// connection; streaming ones each get a new connection, as the server closes
// every connection after its first response.
func checkHTTP1HeaderCase(t *testing.T, titleCase *bool, preconnect, http1Only, useTLS, streaming bool) {
	perConn := 2
	if streaming {
		perConn = 1
	}
	url, heads := captureHTTP1Heads(t, useTLS, perConn)

	builder := NewClientBuilder().
		SetVerify(false).
		AddDefaultHeader("X-Default", "Keep-MiXeD")
	for _, name := range []string{"host", "x-default", "x-mixed", "accept", "cookie"} {
		builder.AddHeaderOrder(name)
	}
	if titleCase != nil {
		builder.SetHTTP1TitleCaseHeaders(*titleCase)
	}
	client, err := builder.Build()
	if err != nil {
		t.Fatalf("ClientBuilder.Build() error = %v", err)
	}
	t.Cleanup(client.Close)

	sessionBuilder := NewSessionBuilder(client)
	if http1Only {
		sessionBuilder.SetHTTP1Only()
	}
	session, err := sessionBuilder.Build()
	if err != nil {
		t.Fatalf("SessionBuilder.Build() error = %v", err)
	}
	t.Cleanup(session.Close)

	if preconnect {
		if err := session.Preconnect(url); err != nil {
			t.Fatalf("Preconnect() error = %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		req := session.Post(url).
			AddHeader("x-MIXED", "Value-MiXeD").
			AddHeader("accept", "text/plain").
			SetCookie("test", "Cookie-MiXeD").
			SetBodyBytes([]byte(http1CaseBody)).
			SetTimeout(10_000)
		if got := sendForBody(t, req, streaming); got != "OK" {
			t.Fatalf("request %d body = %q, want %q", i, got, "OK")
		}
	}

	got := heads()
	if len(got) != 2 {
		t.Fatalf("server received %d requests, want 2: %+v", len(got), got)
	}
	if !streaming && got[0].conn != got[1].conn {
		t.Errorf("buffered requests on connections %d and %d, want the second to reuse the first", got[0].conn, got[1].conn)
	}
	if streaming && got[0].conn == got[1].conn {
		t.Errorf("streaming requests both on connection %d, want a new one each", got[0].conn)
	}

	expected := []string{"Host", "X-Default", "X-Mixed", "Accept", "Cookie"}
	lengthName := "Content-Length"
	if titleCase != nil && !*titleCase {
		expected = []string{"Host", "x-default", "x-mixed", "accept", "cookie"}
		lengthName = "content-length"
	}
	for i, head := range got {
		if head.body != http1CaseBody {
			t.Errorf("request %d body = %q, want %q", i, head.body, http1CaseBody)
		}
		var names []string
		values := map[string]string{}
		for _, field := range head.fields {
			names = append(names, field[0])
			values[field[0]] = field[1]
		}
		if len(names) < len(expected) || strings.Join(names[:len(expected)], ",") != strings.Join(expected, ",") {
			t.Errorf("request %d header names = %v, want them to start with %v", i, names, expected)
		}
		for name, want := range map[string]string{
			expected[1]: "Keep-MiXeD",
			expected[2]: "Value-MiXeD",
			expected[4]: "test=Cookie-MiXeD",
			lengthName:  strconv.Itoa(len(http1CaseBody)),
		} {
			if got, ok := values[name]; !ok || got != want {
				t.Errorf("request %d header %q = %q (present %v), want %q; names %v", i, name, got, ok, want, names)
			}
		}
		if titleCase != nil && !*titleCase {
			for _, name := range names {
				if name != "Host" && name != strings.ToLower(name) {
					t.Errorf("request %d sent %q, want every name but Host in lowercase", i, name)
				}
			}
		}
	}
}

// sendForBody sends req, buffered or streaming, and returns the response body.
func sendForBody(t *testing.T, req *Request, streaming bool) string {
	t.Helper()
	if !streaming {
		resp, err := req.Send()
		if err != nil {
			t.Fatalf("Send() error = %v", err)
		}
		defer resp.Close()
		return string(resp.Body())
	}
	resp, err := req.SendStreaming()
	if err != nil {
		t.Fatalf("SendStreaming() error = %v", err)
	}
	defer resp.Close()
	body, err := io.ReadAll(resp)
	if err != nil {
		t.Fatalf("reading the streaming body error = %v", err)
	}
	return string(body)
}

// http1Head is a request as an HTTP/1.1 server received it: the connection it
// came on, its header fields with names spelled as sent, and its body.
type http1Head struct {
	conn   int
	fields [][2]string
	body   string
}

// captureHTTP1Heads runs an HTTP/1.1 server, over TLS offering only
// http/1.1 when useTLS is set, that records every request it reads and
// answers "OK", closing each connection after perConn responses. It returns
// the URL to post to and a function reporting the requests so far.
func captureHTTP1Heads(t *testing.T, useTLS bool, perConn int) (string, func() []http1Head) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	scheme := "http"
	if useTLS {
		scheme = "https"
		listener = tls.NewListener(listener, &tls.Config{
			Certificates: []tls.Certificate{selfSignedTLSCert(t)},
			NextProtos:   []string{"http/1.1"},
		})
	}

	var (
		mu    sync.Mutex
		heads []http1Head
		conns []net.Conn
	)
	t.Cleanup(func() {
		listener.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, conn := range conns {
			conn.Close()
		}
	})

	serve := func(conn net.Conn, id int) {
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
		reader := bufio.NewReader(conn)
		for served := 1; served <= perConn; served++ {
			head, err := readHTTP1Head(reader)
			if err != nil {
				return
			}
			head.conn = id
			mu.Lock()
			heads = append(heads, head)
			mu.Unlock()

			connection := "keep-alive"
			if served == perConn {
				connection = "close"
			}
			if _, err := fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\nContent-Length: 2\r\nConnection: %s\r\n\r\nOK", connection); err != nil {
				return
			}
		}
	}
	go func() {
		for id := 0; ; id++ {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			conns = append(conns, conn)
			mu.Unlock()
			go serve(conn, id)
		}
	}()

	return scheme + "://" + listener.Addr().String() + "/headers", func() []http1Head {
		mu.Lock()
		defer mu.Unlock()
		return append([]http1Head(nil), heads...)
	}
}

// readHTTP1Head reads one request: its request line, its header fields as
// sent, and a body of the length Content-Length gives in any casing.
func readHTTP1Head(reader *bufio.Reader) (http1Head, error) {
	var head http1Head
	if _, err := reader.ReadString('\n'); err != nil {
		return head, err
	}
	length := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return head, err
		}
		line = strings.TrimSuffix(line, "\r\n")
		if line == "" {
			break
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return head, fmt.Errorf("malformed header line %q", line)
		}
		value = strings.TrimSpace(value)
		head.fields = append(head.fields, [2]string{name, value})
		if strings.EqualFold(name, "content-length") {
			if length, err = strconv.Atoi(value); err != nil {
				return head, err
			}
		}
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(reader, body); err != nil {
		return head, err
	}
	head.body = string(body)
	return head, nil
}
