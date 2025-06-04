package http

import (
	"context"
	"crypto/tls"
	"net"

	"azugo.io/core/instrumenter"

	"github.com/valyala/fasthttp"
)

type options struct {
	TLSConfig         *tls.Config
	Dial              fasthttp.DialFunc
	RetryIf           fasthttp.RetryIfErrFunc
	Transport         fasthttp.RoundTripper
	Context           context.Context
	Instrumenter      instrumenter.Instrumenter
	UserAgent         string
	BaseURL           string
	RequestModifiers  []RequestFunc
	ResponseModifiers []ResponseFunc
	Configuration     *Configuration
	StreamResponse    bool
}

func (o *options) apply(opts []Option) {
	for _, opt := range opts {
		if opt == nil {
			continue
		}

		opt.apply(o)
	}
}

// Option is a functional option for configuring the HTTP client.
type Option interface {
	apply(opt *options)
}

// TLSConfig configures the TLS settings that client will use.
type TLSConfig tls.Config

func (c *TLSConfig) apply(o *options) {
	o.TLSConfig = (*tls.Config)(c)
}

type contextOption struct {
	Context context.Context
}

func (c *contextOption) apply(o *options) {
	o.Context = c.Context
}

// Context sets the custom context for the HTTP client.
func Context(ctx context.Context) Option {
	return &contextOption{ctx}
}

type roundTripperOption struct {
	Transport fasthttp.RoundTripper
}

func (r roundTripperOption) apply(o *options) {
	o.Transport = r.Transport
}

// Transport option for the HTTP client.
func Transport(t fasthttp.RoundTripper) Option {
	return roundTripperOption{Transport: t}
}

// RetryIf is a function that determines if the request should be retried.
type RetryIf fasthttp.RetryIfErrFunc

func (r RetryIf) apply(o *options) {
	o.RetryIf = fasthttp.RetryIfErrFunc(r)
}

// DialContextFunc is a function that dials a network address.
type DialContextFunc func(ctx context.Context, network, addr string) (net.Conn, error)

func (f DialContextFunc) apply(o *options) {
	o.Dial = func(addr string) (net.Conn, error) {
		ctx := o.Context
		if ctx == nil {
			ctx = context.Background()
		}

		return f(ctx, "tcp", addr)
	}
}

// BaseURL sets the base URL for the HTTP client requests.
type BaseURL string

func (u BaseURL) apply(o *options) {
	o.BaseURL = string(u)
}

// UserAgent sets the user agent for the HTTP client requests.
type UserAgent string

func (u UserAgent) apply(o *options) {
	o.UserAgent = string(u)
}

// RequestFunc is a function that modifies the HTTP request.
type RequestFunc func(context.Context, *Request) error

func (f RequestFunc) apply(o *options) {
	o.RequestModifiers = append(o.RequestModifiers, f)
}

// ResponseFunc is a function that modifies the HTTP response.
type ResponseFunc func(context.Context, *Response, error) error

func (f ResponseFunc) apply(o *options) {
	o.ResponseModifiers = append(o.ResponseModifiers, f)
}

// Instrumenter is a function that instruments HTTP client operations.
type Instrumenter instrumenter.Instrumenter

func (i Instrumenter) apply(c *options) {
	c.Instrumenter = instrumenter.Instrumenter(i)
}

// StreamResponse enables receiving response as a stream for the HTTP client.
type StreamResponse bool

func (s StreamResponse) apply(o *options) {
	o.StreamResponse = bool(s)
}
