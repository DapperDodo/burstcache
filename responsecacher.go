package burstcache

import (
	"bytes"
	"fmt"
	"net/http"
)

/*
	Inspired by ResponseRecorder from the go standard library httptest
*/

// ResponseCacher is an implementation of http.ResponseWriter that
// caches responses as they are written
type ResponseCacher struct {
	Code int           // the HTTP response code from WriteHeader
	Head http.Header   // the HTTP response headers
	Body *bytes.Buffer // if non-nil, the bytes.Buffer to append written data to
	Done bool

	wroteHeader bool

	id    int  // unique identifier of this cache
	fresh bool // if fresh, serve it to clients. if not, keep serving but request a refresh
	regen bool // a refreshed response is being generated, until it arrives keep serving this
}

// NewResponseCacher returns an initialized ResponseCacher.
func NewResponseCacher(id int) *ResponseCacher {
	return &ResponseCacher{
		Head:  make(http.Header),
		Body:  new(bytes.Buffer),
		Code:  0,
		id:    id,
		fresh: true,
	}
}

// Serve the cached response (headers, statuscode and body) to a ResponseWriter
// optionally, if mark is true, it sets a header ("X-From-BurstCache")
// TODO: make this configurable
func (c *ResponseCacher) Serve(w http.ResponseWriter, mark bool) {
	for key, val := range c.Head {
		if len(val) > 0 {
			w.Header().Set(key, val[0])
		}
	}
	if mark {
		w.Header().Set("X-From-BurstCache", "1")
	}
	w.WriteHeader(c.Code)
	fmt.Fprintln(w, c.Body.String())
}

// Header returns the response headers.
func (c *ResponseCacher) Header() http.Header {
	m := c.Head
	if m == nil {
		m = make(http.Header)
		c.Head = m
	}
	return m
}

// Write always succeeds and writes to c.Body, if not nil.
func (c *ResponseCacher) Write(buf []byte) (int, error) {
	if !c.wroteHeader {
		c.WriteHeader(200)
	}
	if c.Body != nil {
		c.Body.Write(buf)
	}
	return len(buf), nil
}

// WriteHeader sets c.Code.
func (c *ResponseCacher) WriteHeader(code int) {
	if !c.wroteHeader {
		c.Code = code
	}
	c.wroteHeader = true
}

// Flush sets c.Done to true.
func (c *ResponseCacher) Flush() {
	if !c.wroteHeader {
		c.WriteHeader(200)
	}
	c.Done = true
}
