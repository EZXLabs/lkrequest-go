package lkrequest

import (
	"encoding/json"
	"net/http"
	"sort"
	"testing"
)

// decodeLocalEcho reads the /echo payload of the local feature server.
func decodeLocalEcho(t testing.TB, resp *Response) localEchoResponse {
	t.Helper()

	var echo localEchoResponse
	if err := json.Unmarshal(resp.Body(), &echo); err != nil {
		t.Fatalf("decode echo payload %q: %v", resp.Body(), err)
	}
	return echo
}

// TestSessionVerbMethods checks every verb helper sends its own method, and
// that Request accepts a method the helpers do not cover.
func TestSessionVerbMethods(t *testing.T) {
	server := newLocalFeatureServer(t)
	client, session := newLocalSession(t)
	t.Cleanup(client.Close)
	t.Cleanup(session.Close)

	cases := []struct {
		verb string
		call func(string) *Request
	}{
		{"GET", session.Get},
		{"POST", session.Post},
		{"PUT", session.Put},
		{"DELETE", session.Delete},
		{"PATCH", session.Patch},
		{"OPTIONS", session.Options},
	}

	for _, tc := range cases {
		t.Run(tc.verb, func(t *testing.T) {
			resp, err := tc.call(server.URL + "/echo").Send()
			if err != nil {
				t.Fatalf("%s send error = %v", tc.verb, err)
			}
			cleanupResponse(t, resp)

			if got := resp.StatusCode(); got != http.StatusOK {
				t.Fatalf("%s StatusCode() = %d, want %d", tc.verb, got, http.StatusOK)
			}
			if got := decodeLocalEcho(t, resp).Method; got != tc.verb {
				t.Fatalf("server saw method %q, want %q", got, tc.verb)
			}
		})
	}

	// HEAD has no body to echo, so it is checked on the status line only.
	t.Run("HEAD", func(t *testing.T) {
		resp, err := session.Head(server.URL + "/echo").Send()
		if err != nil {
			t.Fatalf("HEAD send error = %v", err)
		}
		cleanupResponse(t, resp)

		if got := resp.StatusCode(); got != http.StatusOK {
			t.Fatalf("HEAD StatusCode() = %d, want %d", got, http.StatusOK)
		}
	})

	// Request is the escape hatch for methods without a helper.
	t.Run("Request", func(t *testing.T) {
		resp, err := session.Request("PROPFIND", server.URL+"/echo").Send()
		if err != nil {
			t.Fatalf("PROPFIND send error = %v", err)
		}
		cleanupResponse(t, resp)

		if got := decodeLocalEcho(t, resp).Method; got != "PROPFIND" {
			t.Fatalf("server saw method %q, want PROPFIND", got)
		}
	})
}

// TestSessionVerbChainsAfterFailure is the load-bearing test for the deferred
// error design: the verb helpers return *Request with no error, so a failed
// construction has to survive every setter chained onto it and only report at
// send time. Anything else would dereference a nil handle.
func TestSessionVerbChainsAfterFailure(t *testing.T) {
	client, session := newLocalSession(t)
	t.Cleanup(client.Close)
	session.Close()

	req := session.Post("http://127.0.0.1:1/echo").
		AddHeader("x-demo", "v").
		AddQuery("k", "v").
		SetJSON(map[string]string{"a": "b"}).
		SetTimeout(1000)

	if req == nil {
		t.Fatal("verb helper returned a nil Request")
	}

	resp, err := req.Send()
	if err == nil {
		cleanupResponse(t, resp)
		t.Fatal("Send() on a closed session succeeded, want an error")
	}
	if resp != nil {
		t.Fatal("Send() returned both a response and an error")
	}
}

// TestRequestSetJSON checks the marshalling setter, that Clone replays the
// encoded document, and that a value encoding/json rejects is reported at send
// time instead of panicking.
func TestRequestSetJSON(t *testing.T) {
	server := newLocalFeatureServer(t)
	client, session := newLocalSession(t)
	t.Cleanup(client.Close)
	t.Cleanup(session.Close)

	payload := struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}{Name: "zero", N: 7}

	req := session.Post(server.URL + "/echo").SetJSON(payload)
	clone, err := req.Clone()
	if err != nil {
		t.Fatalf("Clone() error = %v", err)
	}

	for i, r := range []*Request{req, clone} {
		resp, err := r.Send()
		if err != nil {
			t.Fatalf("send %d error = %v", i, err)
		}
		cleanupResponse(t, resp)

		var decoded struct {
			Name string `json:"name"`
			N    int    `json:"n"`
		}
		body := decodeLocalEcho(t, resp).Body
		if err := json.Unmarshal([]byte(body), &decoded); err != nil {
			t.Fatalf("send %d body %q is not JSON: %v", i, body, err)
		}
		if decoded.Name != "zero" || decoded.N != 7 {
			t.Fatalf("send %d decoded = %+v, want {zero 7}", i, decoded)
		}
	}

	// A channel has no JSON encoding; the failure must ride along to Send.
	resp, err := session.Post(server.URL + "/echo").SetJSON(make(chan int)).Send()
	if err == nil {
		cleanupResponse(t, resp)
		t.Fatal("SetJSON() with an unencodable value succeeded, want an error")
	}
}

// TestResponseAggregateViews checks Cookies and Redirects agree with the
// indexed accessors they wrap.
func TestResponseAggregateViews(t *testing.T) {
	server := newLocalFeatureServer(t)
	client, session := newLocalSession(t)
	t.Cleanup(client.Close)
	t.Cleanup(session.Close)

	t.Run("cookies", func(t *testing.T) {
		resp, err := session.Get(server.URL + "/set-cookies").Send()
		if err != nil {
			t.Fatalf("Send() error = %v", err)
		}
		cleanupResponse(t, resp)

		cookies := resp.Cookies()
		if len(cookies) != resp.CookieCount() {
			t.Fatalf("len(Cookies()) = %d, CookieCount() = %d", len(cookies), resp.CookieCount())
		}

		names := make([]string, 0, len(cookies))
		for i, c := range cookies {
			wantName, wantValue := resp.CookieAt(i)
			if c.Name != wantName || c.Value != wantValue {
				t.Fatalf("Cookies()[%d] = %+v, CookieAt(%d) = (%q, %q)", i, c, i, wantName, wantValue)
			}
			names = append(names, c.Name)
		}
		sort.Strings(names)
		if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
			t.Fatalf("cookie names = %v, want [alpha beta]", names)
		}
	})

	t.Run("redirects", func(t *testing.T) {
		resp, err := session.Get(server.URL + "/redirect/one").Send()
		if err != nil {
			t.Fatalf("Send() error = %v", err)
		}
		cleanupResponse(t, resp)

		if !resp.WasRedirected() {
			t.Fatal("WasRedirected() = false, want true")
		}

		hops := resp.Redirects()
		if len(hops) != resp.RedirectCount() {
			t.Fatalf("len(Redirects()) = %d, RedirectCount() = %d", len(hops), resp.RedirectCount())
		}
		if len(hops) == 0 {
			t.Fatal("Redirects() is empty, want at least one hop")
		}
		for i, hop := range hops {
			wantURL, wantStatus := resp.RedirectAt(i)
			if hop.URL != wantURL || hop.Status != wantStatus {
				t.Fatalf("Redirects()[%d] = %+v, RedirectAt(%d) = (%q, %d)", i, hop, i, wantURL, wantStatus)
			}
			if hop.Status < 300 || hop.Status > 399 {
				t.Fatalf("Redirects()[%d].Status = %d, want a 3xx", i, hop.Status)
			}
		}
	})

	t.Run("empty", func(t *testing.T) {
		resp, err := session.Get(server.URL + "/text").Send()
		if err != nil {
			t.Fatalf("Send() error = %v", err)
		}
		cleanupResponse(t, resp)

		if got := resp.Redirects(); got != nil {
			t.Fatalf("Redirects() = %+v, want nil for a direct response", got)
		}
	})
}

// TestSessionGetCookies checks the decoded jar view matches the raw JSON one.
func TestSessionGetCookies(t *testing.T) {
	server := newLocalFeatureServer(t)
	client, session := newLocalSession(t)
	t.Cleanup(client.Close)
	t.Cleanup(session.Close)

	resp, err := session.Get(server.URL + "/set-cookies").Send()
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	cleanupResponse(t, resp)

	cookies, err := session.GetCookies(server.URL)
	if err != nil {
		t.Fatalf("GetCookies() error = %v", err)
	}

	raw, err := session.GetCookiesJSON(server.URL)
	if err != nil {
		t.Fatalf("GetCookiesJSON() error = %v", err)
	}
	var want []Cookie
	if err := json.Unmarshal([]byte(raw), &want); err != nil {
		t.Fatalf("unmarshal %q: %v", raw, err)
	}

	if len(cookies) != len(want) {
		t.Fatalf("GetCookies() = %+v, GetCookiesJSON() = %s", cookies, raw)
	}
	for i := range cookies {
		if cookies[i] != want[i] {
			t.Fatalf("GetCookies()[%d] = %+v, want %+v", i, cookies[i], want[i])
		}
	}

	jar := make(map[string]string, len(cookies))
	for _, c := range cookies {
		jar[c.Name] = c.Value
	}
	if jar["alpha"] != "one" || jar["beta"] != "two" {
		t.Fatalf("session jar = %#v, want alpha=one beta=two", jar)
	}
}

// TestSessionPoolGuardVerbMethods mirrors the session verb test for the pooled
// guard, whose requests come from a different FFI constructor.
func TestSessionPoolGuardVerbMethods(t *testing.T) {
	server := newLocalFeatureServer(t)
	client, session := newLocalSession(t)
	t.Cleanup(client.Close)
	t.Cleanup(session.Close)

	pool, err := NewSessionPoolBuilder(client).SetMaxSessions(2).Build()
	if err != nil {
		t.Skipf("session pool unavailable in the loaded lkrequest library: %v", err)
	}
	t.Cleanup(pool.Close)

	guard, err := pool.Acquire()
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	t.Cleanup(guard.Close)

	resp, err := guard.Post(server.URL + "/echo").SetJSON(map[string]int{"n": 1}).Send()
	if err != nil {
		t.Fatalf("guard.Post send error = %v", err)
	}
	cleanupResponse(t, resp)

	echo := decodeLocalEcho(t, resp)
	if echo.Method != "POST" {
		t.Fatalf("server saw method %q, want POST", echo.Method)
	}
	if echo.Body != `{"n":1}` {
		t.Fatalf("server saw body %q, want {\"n\":1}", echo.Body)
	}
}
