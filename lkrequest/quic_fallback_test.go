package lkrequest

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestQUICHandshakeTimeoutFallsBackToTCP covers a site that advertises HTTP/3
// on a UDP port that never answers, which is what a network that silently
// drops UDP looks like. Once a session has learnt of h3 from Alt-Svc, its next
// request tries QUIC first; when that handshake times out, the request must
// still be served over TCP instead of failing with the timeout. Buffered and
// streaming requests reach the fallback by different paths, so each gets a
// client of its own.
func TestQUICHandshakeTimeoutFallsBackToTCP(t *testing.T) {
	blackhole, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.ListenPacket() error = %v", err)
	}
	t.Cleanup(func() { blackhole.Close() })
	go func() {
		buf := make([]byte, 2048)
		for {
			if _, _, err := blackhole.ReadFrom(buf); err != nil {
				return
			}
		}
	}()
	altSvc := fmt.Sprintf(`h3=":%d"; ma=86400`, blackhole.LocalAddr().(*net.UDPAddr).Port)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Alt-Svc", altSvc)
		fmt.Fprint(w, "ok")
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)

	for _, streaming := range []bool{false, true} {
		name := map[bool]string{false: "buffered", true: "streaming"}[streaming]
		t.Run(name, func(t *testing.T) {
			client, err := NewClientBuilder().SetVerify(false).SetTimeoutQUICConnect(300).Build()
			if err != nil {
				t.Fatalf("NewClientBuilder().Build() error = %v", err)
			}
			t.Cleanup(client.Close)
			session, err := NewSession(client)
			if err != nil {
				t.Fatalf("NewSession() error = %v", err)
			}
			t.Cleanup(session.Close)

			// The first request goes over TCP and learns of h3 from Alt-Svc.
			resp, err := session.Get(server.URL + "/discover").Send()
			if err != nil {
				t.Fatalf("discovery request error = %v", err)
			}
			resp.Close()

			req := session.Get(server.URL + "/after").SetTimeout(10_000)
			var status int
			var body string
			if streaming {
				stream, err := req.SendStreaming()
				if err != nil {
					t.Fatalf("SendStreaming() after discovery error = %v", err)
				}
				data, err := io.ReadAll(stream)
				status = stream.StatusCode()
				stream.Close()
				if err != nil {
					t.Fatalf("read streaming body error = %v", err)
				}
				body = string(data)
			} else {
				resp, err := req.Send()
				if err != nil {
					t.Fatalf("Send() after discovery error = %v", err)
				}
				status, body = resp.StatusCode(), resp.String()
				resp.Close()
			}
			if status != http.StatusOK || body != "ok" {
				t.Fatalf("response after discovery = %d %q, want 200 \"ok\"", status, body)
			}
		})
	}
}
