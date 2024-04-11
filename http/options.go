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
	Context           context.Context
	Instrumenter      instrumenter.Instrumenter
	UserAgent         string
	BaseURI           string
	RequestModifiers  []RequestFunc
	ResponseModifiers []ResponseFunc
	Configuration     *Configuration
}

func (o *options) apply(opts []Option) {
	for _, opt := range opts {
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

// BaseURI sets the base URI for the HTTP client requests.
type BaseURI string

func (u BaseURI) apply(o *options) {
	o.BaseURI = string(u)
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
