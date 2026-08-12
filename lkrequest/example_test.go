package lkrequest_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	. "github.com/EZ-XLabs/lkrequest-go/lkrequest"
)

func ExampleGet() {
	resp, err := Get("https://httpbin.org/get")
	if err != nil {
		return
	}
	defer resp.Close()

	_ = resp.StatusCode()
}

func ExampleNewClient() {
	client, err := NewDefaultClient()
	if err != nil {
		return
	}
	defer client.Close()

	session, err := NewSession(client)
	if err != nil {
		return
	}
	defer session.Close()

	req, err := NewRequest(session, "GET", "https://httpbin.org/get")
	if err != nil {
		return
	}

	resp, err := req.Send()
	if err != nil {
		return
	}
	defer resp.Close()
}

func ExampleClientBuilder() {
	client, err := NewClientBuilder().
		SetVerify(true).
		SetTimeoutTotal(5000).
		SetMaxOutstandingOps(16).
		AddH3HeaderOrder("priority").
		Build()
	if err != nil {
		return
	}
	defer client.Close()
}

func ExampleRequest_Send() {
	client, err := NewDefaultClient()
	if err != nil {
		return
	}
	defer client.Close()

	session, err := NewSession(client)
	if err != nil {
		return
	}
	defer session.Close()

	req, err := NewRequest(session, "POST", "https://httpbin.org/post")
	if err != nil {
		return
	}

	resp, err := req.
		AddHeader("accept", "application/json").
		SetJSONBody(`{"hello":"world"}`).
		Send()
	if err != nil {
		return
	}
	defer resp.Close()
}

func ExampleRequest_SendWithContext() {
	client, err := NewDefaultClient()
	if err != nil {
		return
	}
	defer client.Close()

	session, err := NewSession(client)
	if err != nil {
		return
	}
	defer session.Close()

	req, err := NewRequest(session, "GET", "https://httpbin.org/delay/1")
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := req.SendWithContext(ctx)
	if err != nil {
		return
	}
	defer resp.Close()
}

func ExamplePostJSON() {
	resp, err := PostJSON("https://httpbin.org/post", `{"hello":"world"}`)
	if err != nil {
		return
	}
	defer resp.Close()

	_ = resp.StatusCode()
}

func ExampleSessionBuilder() {
	client, err := NewDefaultClient()
	if err != nil {
		return
	}
	defer client.Close()

	session, err := NewSessionBuilder(client).
		SetProxy("http://127.0.0.1:8888").
		SetMaxRedirects(5).
		SetHTTP2Only().
		AddH3HeaderOrder("priority").
		SetDefaultAcceptEncoding(AcceptEncodingGzip|AcceptEncodingBr).
		SetMaxConnections(4).
		SetIdleTimeout(1000).
		SetRetryFixed(2, 100).
		Build()
	if err != nil {
		return
	}
	defer session.Close()
}

func ExampleRequest_SendStreaming() {
	client, err := NewDefaultClient()
	if err != nil {
		return
	}
	defer client.Close()

	session, err := NewSession(client)
	if err != nil {
		return
	}
	defer session.Close()

	req, err := NewRequest(session, "GET", "https://httpbin.org/stream/3")
	if err != nil {
		return
	}

	stream, err := req.SendStreaming()
	if err != nil {
		return
	}
	defer stream.Close()

	body, err := io.ReadAll(stream)
	if err != nil {
		return
	}

	_ = len(body)
}

func ExampleRequest_SendAsync() {
	client, err := NewDefaultClient()
	if err != nil {
		return
	}
	defer client.Close()

	session, err := NewSession(client)
	if err != nil {
		return
	}
	defer session.Close()

	req, err := NewRequest(session, "GET", "https://httpbin.org/get")
	if err != nil {
		return
	}

	respCh, errCh := req.SendAsync(context.Background())
	select {
	case resp := <-respCh:
		if resp == nil {
			return
		}
		defer resp.Close()
		_ = resp.StatusCode()
	case <-errCh:
		return
	}
}

func ExampleListPresetsJSON() {
	presets, err := ListPresetsJSON()
	if err != nil {
		return
	}

	_ = presets
}

func ExampleMetrics() {
	snap, err := Metrics()
	if err != nil {
		return
	}

	// Counters are all zero unless the library was built with the telemetry feature.
	_ = snap.RequestsTotal
	_ = snap.BytesIn
}

func ExampleResponse_Headers() {
	client, err := NewDefaultClient()
	if err != nil {
		return
	}
	defer client.Close()

	session, err := NewSession(client)
	if err != nil {
		return
	}
	defer session.Close()

	req, err := NewRequest(session, "GET", "https://httpbin.org/json")
	if err != nil {
		return
	}

	resp, err := req.Send()
	if err != nil {
		return
	}
	defer resp.Close()

	headers := resp.Headers()
	_ = headers.Get("Content-Type")
}

func ExampleNewMultipart() {
	mp := NewMultipart()
	mp.AddText("field1", "value1").
		AddFile("avatar", "photo.jpg", "image/jpeg", []byte{0xFF, 0xD8})
	defer mp.Close()

	fmt.Println("Multipart created")
	// Output: Multipart created
}

func ExampleNewProxyPoolBuilder() {
	builder := NewProxyPoolBuilder()
	builder.AddProxies([]string{
		"http://proxy1.example.com:8080",
		"http://proxy2.example.com:8080",
	}).SetRotation(RotationRoundRobin).
		SetMaxProxies(100)

	// pool, err := builder.Build()
	_ = builder
	fmt.Println("ProxyPoolBuilder created")
	// Output: ProxyPoolBuilder created
}

func ExampleNewSessionPoolBuilder() {
	client, err := NewDefaultClient()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer client.Close()

	builder := NewSessionPoolBuilder(client)
	builder.AddProxy("http://proxy1.example.com:8080").
		SetMaxSessions(10).
		SetIdleTimeout(30000).
		SetRotation(RotationRandom)

	_ = builder
	fmt.Println("SessionPoolBuilder created")
	// Output: SessionPoolBuilder created
}

func ExampleClientBuilder_SetDNS() {
	client, err := NewClientBuilder().
		SetDNS(DnsCloudflareHTTPS).
		SetUseNativeCerts(true).
		Build()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer client.Close()

	fmt.Println("Client with custom DNS created")
	// Output: Client with custom DNS created
}

func ExampleClientBuilder_SetRandomize() {
	// Tier 1: per-connection TLS extension-order jitter. This drifts JA3 to
	// dodge per-fingerprint rate limits while staying a byte-exact real
	// browser, and is always available. The synthetic tiers
	// (RandomizeRecombine / RandomizeFull) produce fingerprints that match no
	// real browser and require a library built with the synthetic-fp feature;
	// the layers bitmask (FingerprintLayersAll, or LayerTLS|LayerH2|LayerQUIC)
	// selects which layers they vary and is ignored here.
	client, err := NewClientBuilder().
		SetRandomize(RandomizeExtensionOrder, FingerprintLayersAll).
		Build()
	if err != nil {
		// Returned when the loaded library predates the randomize export.
		fmt.Println("randomize unsupported:", err)
		return
	}
	defer client.Close()

	fmt.Println("client with extension-order randomization created")
}

func ExampleInitLog() {
	logPath := filepath.Join(os.TempDir(), "lkrequest.log")
	if err := InitLog("info", logPath); err != nil {
		return
	}
}

func ExampleNewSessionWithConfig() {
	client, err := NewDefaultClient()
	if err != nil {
		return
	}
	defer client.Close()

	session, err := NewSessionWithConfig(client, "", 5)
	if err != nil {
		return
	}
	defer session.Close()
}

func ExampleSession_SetCookie() {
	client, err := NewDefaultClient()
	if err != nil {
		return
	}
	defer client.Close()

	session, err := NewSession(client)
	if err != nil {
		return
	}
	defer session.Close()

	_ = session.SetCookie("https://example.com/app", "session", "cookie-value")
}

func ExampleSession_Preconnect() {
	client, err := NewDefaultClient()
	if err != nil {
		return
	}
	defer client.Close()

	session, err := NewSession(client)
	if err != nil {
		return
	}
	defer session.Close()

	_ = session.Preconnect("https://example.com")
}

func ExampleSession_ConnectionPoolStats() {
	client, err := NewDefaultClient()
	if err != nil {
		return
	}
	defer client.Close()

	session, err := NewSession(client)
	if err != nil {
		return
	}
	defer session.Close()

	stats, err := session.ConnectionPoolStats()
	if err != nil {
		return
	}

	_, _, _, _ = stats.H1, stats.H2, stats.H3, stats.Total
}

func ExampleRequest_SetCookieOverride() {
	client, err := NewDefaultClient()
	if err != nil {
		return
	}
	defer client.Close()

	session, err := NewSession(client)
	if err != nil {
		return
	}
	defer session.Close()

	req, err := NewRequest(session, "GET", "https://example.com/profile")
	if err != nil {
		return
	}

	req.SetCookieOverride("session", "fresh-token")
}

func ExampleRequest_SetMultipart() {
	client, err := NewDefaultClient()
	if err != nil {
		return
	}
	defer client.Close()

	session, err := NewSession(client)
	if err != nil {
		return
	}
	defer session.Close()

	req, err := NewRequest(session, "POST", "https://example.com/upload")
	if err != nil {
		return
	}

	mp := NewMultipart().
		AddText("name", "alice").
		AddFile("avatar", "avatar.txt", "text/plain", []byte("hello"))

	req.SetMultipart(mp)
}

func ExampleResponse_ErrorForStatus() {
	resp, err := Get("https://httpbin.org/status/418")
	if err != nil {
		return
	}
	defer resp.Close()

	_ = resp.ErrorForStatus()
}

func ExampleStreamingResponse_Header() {
	client, err := NewDefaultClient()
	if err != nil {
		return
	}
	defer client.Close()

	session, err := NewSession(client)
	if err != nil {
		return
	}
	defer session.Close()

	req, err := NewRequest(session, "GET", "https://httpbin.org/stream/1")
	if err != nil {
		return
	}

	stream, err := req.SendStreaming()
	if err != nil {
		return
	}
	defer stream.Close()

	_ = stream.Header("content-type")
	_ = stream.DiagnosticsJSON()
}

func ExampleProxyPool_Acquire() {
	pool, err := NewProxyPoolBuilder().
		AddProxy("http://proxy1.example.com:8080").
		AddProxy("http://proxy2.example.com:8080").
		SetRotation(RotationRoundRobin).
		Build()
	if err != nil {
		return
	}
	defer pool.Close()

	guard, err := pool.Acquire()
	if err != nil {
		return
	}
	defer guard.Close()

	_ = guard.URL()
}

func ExampleSessionPool_Acquire() {
	client, err := NewDefaultClient()
	if err != nil {
		return
	}
	defer client.Close()

	pool, err := NewSessionPoolBuilder(client).
		AddProxy("http://proxy1.example.com:8080").
		AddProxy("http://proxy2.example.com:8080").
		SetMaxSessions(8).
		Build()
	if err != nil {
		return
	}
	defer pool.Close()

	guard, err := pool.Acquire()
	if err != nil {
		return
	}
	defer guard.Close()

	req, err := guard.NewRequest("GET", "https://example.com")
	if err != nil {
		return
	}

	_ = req
}
