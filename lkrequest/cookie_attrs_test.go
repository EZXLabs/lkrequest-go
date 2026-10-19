package lkrequest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// newCookieAttrsSession returns a session that is closed when the test ends,
// skipping when the loaded library predates the cookie attribute readers.
func newCookieAttrsSession(t *testing.T) *Session {
	t.Helper()
	if !cookieAttrsSupported() {
		t.Skip("cookie attribute readers are not supported by the loaded lkrequest library")
	}
	return newCookieSession(t)
}

// newCookieSession returns a session that is closed when the test ends.
func newCookieSession(t *testing.T) *Session {
	t.Helper()
	client, err := NewClientBuilder().Build()
	if err != nil {
		t.Fatalf("NewClientBuilder().Build() error = %v", err)
	}
	t.Cleanup(client.Close)
	session, err := NewSession(client)
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	t.Cleanup(session.Close)
	return session
}

// cookieAttrsSupported reports whether the loaded library has both cookie
// attribute readers.
func cookieAttrsSupported() bool {
	return ffi_lk_session_get_cookies_with_attrs_json != nil && ffi_lk_session_get_all_cookies_json != nil
}

// TestCookieAttrsFromSetCookie reads back what a server set: expiry, HttpOnly
// and SameSite survive, an already-expired cookie is gone, and two cookies
// sharing a name at different paths stay two entries, longest path first —
// the case a name/value pair cannot tell apart.
func TestCookieAttrsFromSetCookie(t *testing.T) {
	session := newCookieAttrsSession(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Set-Cookie", "ttl=live; Path=/; Max-Age=3600; HttpOnly; SameSite=Lax")
		w.Header().Add("Set-Cookie", "session=present; Path=/; SameSite=Strict")
		w.Header().Add("Set-Cookie", "expired=gone; Path=/; Max-Age=0")
		w.Header().Add("Set-Cookie", "ttl=scoped; Path=/a")
	}))
	t.Cleanup(server.Close)

	url := server.URL + "/a/b"
	before := time.Now().Unix()
	resp, err := session.Get(url).Send()
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	resp.Close()
	after := time.Now().Unix()

	cookies, err := session.GetCookiesWithAttrs(url)
	if err != nil {
		t.Fatalf("GetCookiesWithAttrs() error = %v", err)
	}
	if got := cookieKeys(cookies); got != "ttl=scoped@/a ttl=live@/ session=present@/" {
		t.Fatalf("GetCookiesWithAttrs() = %s, want ttl=scoped@/a ttl=live@/ session=present@/", got)
	}

	scoped, live, transient := cookies[0], cookies[1], cookies[2]
	if !scoped.HostOnly || scoped.Domain != "127.0.0.1" || scoped.SameSite != "" || scoped.HTTPOnly || scoped.IsPersistent() {
		t.Errorf("ttl@/a = %+v, want a host-only 127.0.0.1 session cookie with no SameSite or HttpOnly", scoped)
	}
	if !live.HTTPOnly || live.SameSite != "Lax" || live.Secure {
		t.Errorf("ttl@/ = %+v, want HttpOnly, SameSite=Lax, not Secure", live)
	}
	if live.ExpiresUnix == nil || *live.ExpiresUnix < before+3600 || *live.ExpiresUnix > after+3600 {
		t.Errorf("ttl@/ ExpiresUnix = %v, want within [%d, %d]", live.ExpiresUnix, before+3600, after+3600)
	}
	if transient.SameSite != "Strict" || transient.HTTPOnly || transient.IsPersistent() {
		t.Errorf("session@/ = %+v, want a SameSite=Strict session cookie", transient)
	}

	all, err := session.GetAllCookies()
	if err != nil {
		t.Fatalf("GetAllCookies() error = %v", err)
	}
	if got := cookieKeys(all); got != "ttl=live@/ session=present@/ ttl=scoped@/a" {
		t.Errorf("GetAllCookies() = %s, want the same three cookies oldest first", got)
	}

	// The name/value reader still sees both ttl cookies, with nothing to say
	// which is which.
	legacy, err := session.GetCookies(url)
	if err != nil {
		t.Fatalf("GetCookies() error = %v", err)
	}
	if len(legacy) != 3 {
		t.Errorf("GetCookies() = %+v, want 3 name/value pairs", legacy)
	}
}

// TestCookieEmptyMaxAgeIsIgnored mirrors upstream's empty Max-Age round trip.
// An empty Max-Age counts as absent: a cookie that also has a past Expires is
// expired, one with no other expiry stays a session cookie instead of lasting
// until the year 9999, and a valid Max-Age before an empty one still applies,
// 3600 over a past Expires and 0 deleting the cookie.
func TestCookieEmptyMaxAgeIsIgnored(t *testing.T) {
	session := newCookieAttrsSession(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/echo" {
			fmt.Fprint(w, r.Header.Get("Cookie"))
			return
		}
		const past = "Expires=Thu, 01 Jan 1970 00:00:00 GMT"
		w.Header().Add("Set-Cookie", "expired=gone; Path=/; Max-Age=; "+past)
		w.Header().Add("Set-Cookie", "valid=live; Path=/; Max-Age=3600; Max-Age=; "+past)
		w.Header().Add("Set-Cookie", "session=live; Path=/; Max-Age=; Max-Age=")
		w.Header().Add("Set-Cookie", "deleted=gone; Path=/; Max-Age=0; Max-Age=")
	}))
	t.Cleanup(server.Close)

	url := server.URL + "/"
	resp, err := session.Get(url).Send()
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	resp.Close()

	for _, name := range []string{"valid", "session"} {
		if v, err := session.GetCookie(url, name); err != nil || v != "live" {
			t.Errorf("GetCookie(%q) = %q, %v; want \"live\"", name, v, err)
		}
	}
	for _, name := range []string{"expired", "deleted"} {
		if v, err := session.GetCookie(url, name); err == nil {
			t.Errorf("GetCookie(%q) = %q, want a miss", name, v)
		}
	}

	resp, err = session.Get(server.URL + "/echo").Send()
	if err != nil {
		t.Fatalf("Send(/echo) error = %v", err)
	}
	echoed, err := resp.Text()
	resp.Close()
	if err != nil || echoed != "valid=live; session=live" {
		t.Errorf("Cookie header sent = %q, %v; want \"valid=live; session=live\"", echoed, err)
	}

	all, err := session.GetAllCookies()
	if err != nil {
		t.Fatalf("GetAllCookies() error = %v", err)
	}
	if got := cookieKeys(all); got != "valid=live@/ session=live@/" {
		t.Fatalf("GetAllCookies() = %s, want valid=live@/ session=live@/", got)
	}
	if valid := all[0]; !valid.IsPersistent() {
		t.Errorf("valid = %+v, want the persistent cookie Max-Age=3600 makes", valid)
	}
	if transient := all[1]; !transient.HostOnly || transient.IsPersistent() {
		t.Errorf("session = %+v, want a host-only session cookie", transient)
	}
}

// TestCookieAttrsAcrossDomains mirrors upstream's scope test with cookies
// inserted directly: GetAllCookies spans domains, GetCookiesWithAttrs applies
// domain, path and Secure matching, and removals show up in both readers.
func TestCookieAttrsAcrossDomains(t *testing.T) {
	session := newCookieAttrsSession(t)

	for _, c := range []struct {
		url, name, value string
		attrs            CookieAttrs
	}{
		{"https://a.example.test/", "root", "1", CookieAttrs{Path: "/", HTTPOnly: true}},
		{"https://a.example.test/api", "api", "2", CookieAttrs{Path: "/api", Domain: "example.test", Secure: true}},
		{"https://b.other.test/", "other", "3", CookieAttrs{Path: "/"}},
	} {
		if err := session.SetCookieWithAttrs(c.url, c.name, c.value, c.attrs); err != nil {
			t.Fatalf("SetCookieWithAttrs(%s) error = %v", c.name, err)
		}
	}

	all, err := session.GetAllCookies()
	if err != nil {
		t.Fatalf("GetAllCookies() error = %v", err)
	}
	if got := cookieKeys(all); got != "root=1@/ api=2@/api other=3@/" {
		t.Fatalf("GetAllCookies() = %s, want root, api and other, oldest first", got)
	}
	if root := all[0]; !root.HostOnly || root.Domain != "a.example.test" || !root.HTTPOnly || root.Secure {
		t.Errorf("root = %+v, want host-only a.example.test, HttpOnly, not Secure", root)
	}
	if api := all[1]; api.HostOnly || api.Domain != "example.test" || !api.Secure {
		t.Errorf("api = %+v, want domain example.test matching subdomains, Secure", api)
	}

	for url, want := range map[string]string{
		"https://a.example.test/api/users": "api=2@/api root=1@/",
		"http://a.example.test/api/users":  "root=1@/",
		"https://sub.a.example.test/api":   "api=2@/api",
		"https://unrelated.test/":          "",
		"not a url":                        "",
	} {
		got, err := session.GetCookiesWithAttrs(url)
		if err != nil {
			t.Errorf("GetCookiesWithAttrs(%q) error = %v", url, err)
			continue
		}
		if keys := cookieKeys(got); keys != want {
			t.Errorf("GetCookiesWithAttrs(%q) = %q, want %q", url, keys, want)
		}
	}

	if err := session.RemoveCookie("https://a.example.test/", "root"); err != nil {
		t.Fatalf("RemoveCookie() error = %v", err)
	}
	if all, err := session.GetAllCookies(); err != nil || cookieKeys(all) != "api=2@/api other=3@/" {
		t.Errorf("GetAllCookies() after RemoveCookie = %s, %v; want api and other", cookieKeys(all), err)
	}
	if err := session.ClearCookies(); err != nil {
		t.Fatalf("ClearCookies() error = %v", err)
	}
	if all, err := session.GetAllCookies(); err != nil || len(all) != 0 {
		t.Errorf("GetAllCookies() after ClearCookies = %+v, %v; want none", all, err)
	}
}

// TestSessionCookieCallsShareSafely drives every cookie reader against
// concurrent writers on one session. The library frees a reader's buffer when
// a cookie is written or the same query repeats, so an unsynchronized read
// copies freed memory; here every read must still decode to a value a writer
// actually set. The name/value readers predate the attribute readers and are
// exercised on every library.
func TestSessionCookieCallsShareSafely(t *testing.T) {
	session := newCookieSession(t)
	const url = "https://a.example.test/"
	if err := session.SetCookie(url, "k", "v-start"); err != nil {
		t.Fatalf("SetCookie() error = %v", err)
	}

	deadline := time.Now().Add(300 * time.Millisecond)
	errs := make(chan error, 16)
	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; time.Now().Before(deadline); i++ {
				if err := session.SetCookie(url, "k", fmt.Sprintf("v-%d-%d", w, i)); err != nil {
					errs <- err
					return
				}
			}
		}(w)
	}
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				if err := readCookiesOnce(session, url); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

// readCookiesOnce calls each cookie reader once and checks the value it saw.
func readCookiesOnce(session *Session, url string) error {
	valid := func(v string) bool { return strings.HasPrefix(v, "v-") }

	if v, err := session.GetCookie(url, "k"); err != nil || !valid(v) {
		return fmt.Errorf("GetCookie() = %q, %v", v, err)
	}
	if c, err := session.GetCookies(url); err != nil || len(c) != 1 || !valid(c[0].Value) {
		return fmt.Errorf("GetCookies() = %+v, %v", c, err)
	}
	if !cookieAttrsSupported() {
		return nil
	}
	if c, err := session.GetCookiesWithAttrs(url); err != nil || len(c) != 1 || !valid(c[0].Value) {
		return fmt.Errorf("GetCookiesWithAttrs() = %+v, %v", c, err)
	}
	if c, err := session.GetAllCookies(); err != nil || len(c) != 1 || !valid(c[0].Value) {
		return fmt.Errorf("GetAllCookies() = %+v, %v", c, err)
	}
	return nil
}

// cookieKeys renders cookies as "name=value@path" in order, for comparison.
func cookieKeys(cookies []CookieInfo) string {
	keys := make([]string, len(cookies))
	for i, c := range cookies {
		keys[i] = c.Name + "=" + c.Value + "@" + c.Path
	}
	return strings.Join(keys, " ")
}
