package http

import (
	"context"
	"fmt"
	"mime/multipart"
	"runtime/debug"
	"strings"
	"sync"

	"azugo.io/core/instrumenter"

	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
)

var (
	defaultUserAgent string

	strContentTypeJSON = []byte("application/json")
)

const (
	InstrumentationRequest = "http-client-request"
)

// Client is the interface that provides HTTP client.
type Client interface {
	// WithConfiguration returns a new client with specific named configuration.
	WithConfiguration(name string) (Client, error)
	// WithContext returns a new client with specified context.
	WithContext(ctx context.Context) Client
	// WithBaseURL returns a new client with specified base URL.
	WithBaseURL(url string) Client
	// WithOptions returns a new client with additional options applied.
	WithOptions(opt ...Option) Client

	// UserAgent returns client user agent.
	UserAgent() string
	// BaseURL returns client base URL.
	BaseURL() string

	// NewRequest creates a new HTTP request.
	//
	// The returned request must be released after use by calling ReleaseRequest.
	NewRequest() *Request
	// NewResponse creates a new HTTP response.
	//
	// The returned response must be released after use by calling ReleaseResponse.
	NewResponse() *Response
	// ReleaseRequest releases the HTTP request back to pool.
	ReleaseRequest(req *Request)
	// ReleaseResponse releases the HTTP response back to pool.
	ReleaseResponse(resp *Response)

	// Do sends an HTTP request and returns an HTTP response.
	Do(req *Request, resp *Response) error
	// Get sends an HTTP GET request and returns an HTTP response body.
	Get(url string, opt ...RequestOption) ([]byte, error)
	// GetJSON sends an HTTP GET request and unmarshals the response body into v.
	GetJSON(url string, v any, opt ...RequestOption) error
	// Head sends an HTTP POST request and returns an HTTP response body.
	Post(url string, body []byte, opt ...RequestOption) ([]byte, error)
	// PostJSON sends an HTTP POST request and unmarshals response body into v.
	PostJSON(url string, body, v any, opt ...RequestOption) error
	// PostForm sends an HTTP POST request with form data and returns an HTTP response body.
	PostForm(url string, form map[string][]string, opt ...RequestOption) ([]byte, error)
	// PostMultipartForm sends an HTTP POST request with multipart form data and returns an HTTP response body.
	PostMultipartForm(url string, form *multipart.Form, opt ...RequestOption) ([]byte, error)
	// Put sends an HTTP PUT request and returns an HTTP response body.
	Put(url string, body []byte, opt ...RequestOption) ([]byte, error)
	// PutJSON sends an HTTP PUT request and unmarshals response body into v.
	PutJSON(url string, body, v any, opt ...RequestOption) error
	// Patch sends an HTTP PATCH request and returns an HTTP response body.
	Patch(url string, body []byte, opt ...RequestOption) ([]byte, error)
	// PatchJSON sends an HTTP PATCH request and unmarshals response body into v.
	PatchJSON(url string, body, v any, opt ...RequestOption) error
	// Delete sends an HTTP DELETE request.
	Delete(url string, opt ...RequestOption) error
}

// ClientProvider is the interface that provides HTTP client.
type ClientProvider interface {
	// HTTPClient returns HTTP client instance.
	HTTPClient() Client
}

type clientOpts struct {
	requestPool  sync.Pool
	responsePool sync.Pool
	bufferPool   bytebufferpool.Pool
	reqMod       []RequestFunc
	respMod      []ResponseFunc
	config       *Configuration
	instrumenter instrumenter.Instrumenter
}

type client struct {
	*clientOpts
	c       *fasthttp.Client
	baseURL string
	ctx     context.Context
}

func NewClient(opt ...Option) Client {
	opts := &options{
		RequestModifiers:  make([]RequestFunc, 0),
		ResponseModifiers: make([]ResponseFunc, 0),
		Configuration: &Configuration{
			Clients: make(map[string]NamedClient),
		},
		Context:      context.Background(),
		Instrumenter: instrumenter.NullInstrumenter,
	}
	opts.apply(opt)

	if opts.UserAgent == "" {
		opts.UserAgent = defaultUserAgent
	}

	return &client{
		clientOpts: &clientOpts{
			reqMod:       opts.RequestModifiers,
			respMod:      opts.ResponseModifiers,
			config:       opts.Configuration,
			instrumenter: opts.Instrumenter,
		},
		c: &fasthttp.Client{
			Name:      opts.UserAgent,
			TLSConfig: opts.TLSConfig,
			Dial:      opts.Dial,
		},
		baseURL: opts.BaseURL,
		ctx:     opts.Context,
	}
}

func (c client) Do(req *Request, resp *Response) error {
	if c.ctx.Err() != nil {
		return c.ctx.Err()
	}

	for _, f := range c.reqMod {
		if err := f(c.ctx, req); err != nil {
			return err
		}

		if c.ctx.Err() != nil {
			return c.ctx.Err()
		}
	}

	finish := c.instrumenter.Observe(c.ctx, InstrumentationRequest, req, resp)

	err := c.c.Do(req.Request, resp.Response)

	if c.ctx.Err() != nil {
		finish(c.ctx.Err())

		return c.ctx.Err()
	}

	finish(err)

	for _, f := range c.respMod {
		if e := f(c.ctx, resp, err); e != nil {
			return e
		}

		if c.ctx.Err() != nil {
			return c.ctx.Err()
		}
	}

	return err
}

// UserAgent returns client default user agent.
func (c client) UserAgent() string {
	return c.c.Name
}

// BaseURL returns client base URL.
func (c client) BaseURL() string {
	return c.baseURL
}

// WithModifiers returns a new client with specified context.
func (c client) WithContext(ctx context.Context) Client {
	return &client{
		clientOpts: c.clientOpts,
		baseURL:    c.baseURL,
		c:          c.c,
		ctx:        ctx,
	}
}

// WithBaseURL returns a new client with specified base URL.
func (c client) WithBaseURL(url string) Client {
	return &client{
		clientOpts: c.clientOpts,
		baseURL:    url,
		c:          c.c,
		ctx:        c.ctx,
	}
}

// WithConfiguration returns a new client with specific named configuration.
func (c client) WithConfiguration(name string) (Client, error) {
	cl, ok := c.config.Clients[name]
	if !ok {
		return nil, fmt.Errorf("client %q not found", name)
	}

	return c.WithBaseURL(cl.BaseURL), nil
}

// WithOptions returns a new client with additional options applied.
func (c client) WithOptions(opt ...Option) Client {
	opts := &options{
		RequestModifiers:  c.reqMod,
		ResponseModifiers: c.respMod,
		Configuration:     c.config,
		Context:           c.ctx,
		Instrumenter:      c.instrumenter,
		TLSConfig:         c.c.TLSConfig,
		Dial:              c.c.Dial,
		UserAgent:         c.c.Name,
		BaseURL:           c.baseURL,
	}
	opts.apply(opt)

	if opts.UserAgent == "" {
		opts.UserAgent = defaultUserAgent
	}

	return &client{
		clientOpts: &clientOpts{
			reqMod:       opts.RequestModifiers,
			respMod:      opts.ResponseModifiers,
			config:       opts.Configuration,
			instrumenter: opts.Instrumenter,
		},
		c: &fasthttp.Client{
			Name:      opts.UserAgent,
			TLSConfig: opts.TLSConfig,
			Dial:      opts.Dial,
		},
		baseURL: opts.BaseURL,
		ctx:     opts.Context,
	}
}

// InstrRequest returns request and response if the operation is HTTP client request event.
func InstrRequest(op string, args ...any) (*Request, *Response, bool) {
	if op != InstrumentationRequest || len(args) != 2 {
		return nil, nil, false
	}

	req, ok1 := args[0].(*Request)
	resp, ok2 := args[1].(*Response)

	return req, resp, ok1 && ok2
}

func init() {
	if di, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range di.Deps {
			if dep.Path == "azugo.io/core" {
				ver := strings.TrimPrefix(dep.Version, "v")
				if ver == "" {
					ver = dep.Sum[:8]
				}

				defaultUserAgent = "Azugo/" + ver

				break
			}
		}
	}

	if defaultUserAgent == "" {
		defaultUserAgent = "Azugo/dev"
	}
}
