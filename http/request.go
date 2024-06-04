package http

import (
	"net/url"
	"strings"

	"github.com/valyala/fasthttp"
)

// Request represents an HTTP request.
type Request struct {
	*fasthttp.Request
	client Client
}

// NewRequest creates a new HTTP request.
//
// The returned request must be released after use by calling ReleaseRequest.
func (c clientInstance) NewRequest() *Request {
	v := c.requestPool.Get()
	if v == nil {
		return &Request{
			Request: fasthttp.AcquireRequest(),
			client:  c,
		}
	}

	req, _ := v.(*Request)
	req.Request = fasthttp.AcquireRequest()
	req.client = c

	return req
}

// ReleaseRequest releases the HTTP request back to pool.
func (c clientInstance) ReleaseRequest(req *Request) {
	r := req.Request
	req.Request = nil
	req.client = nil

	fasthttp.ReleaseRequest(r)
	c.requestPool.Put(req)
}

// SetRequestURL sets the request URL.
func (r Request) SetRequestURL(u string) error {
	if baseURL := r.client.BaseURL(); baseURL != "" && !strings.Contains(u, "://") {
		var err error
		if u, err = url.JoinPath(baseURL, u); err != nil {
			return err
		}
	}

	r.Request.SetRequestURI(u)

	return nil
}

// BaseURL returns the base URL of the client.
func (r Request) BaseURL() string {
	return r.client.BaseURL()
}

// RequestOption is a functional option for configuring the HTTP request.
type RequestOption interface {
	apply(r *Request)
}

func (r *Request) apply(opts []RequestOption) {
	for _, opt := range opts {
		opt.apply(r)
	}
}

type requestHeader struct {
	Key, Value string
}

func (h *requestHeader) apply(r *Request) {
	r.Header.Set(h.Key, h.Value)
}

// WithHeader sets the specified header key and value for the request.
func WithHeader(key, value string) RequestOption {
	return &requestHeader{
		Key:   key,
		Value: value,
	}
}

type requestQueryArg struct {
	Key      string
	Value    any
	Override bool
}

func (p *requestQueryArg) apply(r *Request) {
	args := r.URI().QueryArgs()

	val, ok := p.Value.(string)
	if !ok {
		return
	}

	if p.Override {
		args.Set(p.Key, val)
	} else {
		args.Add(p.Key, val)
	}
}

// WithQueryArg sets the specified query argument key and value for the request.
func WithQueryArg(key string, value any, override ...bool) RequestOption {
	var o bool
	if len(override) > 0 {
		o = override[0]
	}

	return &requestQueryArg{
		Key:      key,
		Value:    value,
		Override: o,
	}
}
