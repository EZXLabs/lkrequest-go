package lkrequest

// HTTP verb helpers. These are pure Go sugar over the single FFI request entry
// point (lk_request_new / lk_session_pool_guard_request_new), which takes the
// method as a string; there is no per-verb symbol to bind.
//
// They return *Request rather than (*Request, error) so a call stays chainable
// with the setters that follow it. A construction failure is parked in the
// request's firstErr and surfaces from Send and the other send variants, which
// already check it before touching the handle. NewRequest keeps its
// (*Request, error) shape for callers that want the error immediately.

// errorRequest builds a Request that owns no handle and only carries err.
// Every setter short-circuits on it via checkMutable, and every send variant
// returns err.
func errorRequest(session *Session, method, url string, err error) *Request {
	return &Request{
		firstErr: err,
		session:  session,
		method:   method,
		url:      url,
	}
}

// Request starts a request with an arbitrary method on this session. It backs
// the verb helpers below and is the entry point for methods they do not cover,
// such as PROPFIND or a service-specific verb.
func (s *Session) Request(method, url string) *Request {
	req, err := NewRequest(s, method, url)
	if err != nil {
		return errorRequest(s, method, url, err)
	}
	return req
}

// Get starts a GET request on this session.
func (s *Session) Get(url string) *Request { return s.Request("GET", url) }

// Post starts a POST request on this session.
func (s *Session) Post(url string) *Request { return s.Request("POST", url) }

// Put starts a PUT request on this session.
func (s *Session) Put(url string) *Request { return s.Request("PUT", url) }

// Delete starts a DELETE request on this session.
func (s *Session) Delete(url string) *Request { return s.Request("DELETE", url) }

// Head starts a HEAD request on this session.
func (s *Session) Head(url string) *Request { return s.Request("HEAD", url) }

// Patch starts a PATCH request on this session.
func (s *Session) Patch(url string) *Request { return s.Request("PATCH", url) }

// Options starts an OPTIONS request on this session.
func (s *Session) Options(url string) *Request { return s.Request("OPTIONS", url) }

// Request starts a request with an arbitrary method on the session held by this
// guard. The returned Request has its own lifetime and stays valid after the
// guard is released.
func (g *SessionPoolGuard) Request(method, url string) *Request {
	req, err := g.NewRequest(method, url)
	if err != nil {
		return errorRequest(nil, method, url, err)
	}
	return req
}

// Get starts a GET request on the guarded session.
func (g *SessionPoolGuard) Get(url string) *Request { return g.Request("GET", url) }

// Post starts a POST request on the guarded session.
func (g *SessionPoolGuard) Post(url string) *Request { return g.Request("POST", url) }

// Put starts a PUT request on the guarded session.
func (g *SessionPoolGuard) Put(url string) *Request { return g.Request("PUT", url) }

// Delete starts a DELETE request on the guarded session.
func (g *SessionPoolGuard) Delete(url string) *Request { return g.Request("DELETE", url) }

// Head starts a HEAD request on the guarded session.
func (g *SessionPoolGuard) Head(url string) *Request { return g.Request("HEAD", url) }

// Patch starts a PATCH request on the guarded session.
func (g *SessionPoolGuard) Patch(url string) *Request { return g.Request("PATCH", url) }

// Options starts an OPTIONS request on the guarded session.
func (g *SessionPoolGuard) Options(url string) *Request { return g.Request("OPTIONS", url) }
