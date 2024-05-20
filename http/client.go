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
	WithConfiguration(name string) (Client, error)
	WithContext(ctx context.Context) Client
	WithBaseURI(uri string) Client

	UserAgent() string
	BaseURI() string

	NewRequest() *Request
	NewResponse() *Response

	Do(req *Request, resp *Response) error
	Get(uri string, opt ...RequestOption) ([]byte, error)
	GetJSON(uri string, v any, opt ...RequestOption) error
	Post(uri string, body []byte, opt ...RequestOption) ([]byte, error)
	PostJSON(uri string, body, v any, opt ...RequestOption) error
	PostForm(uri string, form map[string][]string, opt ...RequestOption) ([]byte, error)
	PostMultipartForm(uri string, form *multipart.Form, opt ...RequestOption) ([]byte, error)
	Put(uri string, body []byte, opt ...RequestOption) ([]byte, error)
	PutJSON(uri string, body, v any, opt ...RequestOption) error
	Patch(uri string, body []byte, opt ...RequestOption) ([]byte, error)
	PatchJSON(uri string, body, v any, opt ...RequestOption) error
	Delete(uri string, opt ...RequestOption) error
}

type client struct {
	requestPool  sync.Pool
	responsePool sync.Pool
	bufferPool   bytebufferpool.Pool
	c            *fasthttp.Client
	reqMod       []RequestFunc
	respMod      []ResponseFunc
	config       *Configuration
	instrumenter instrumenter.Instrumenter
}

type clientInstance struct {
	*client
	baseURI string
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

	return &clientInstance{
		client: &client{
			c: &fasthttp.Client{
				Name:      opts.UserAgent,
				TLSConfig: opts.TLSConfig,
				Dial:      opts.Dial,
			},
			reqMod:       opts.RequestModifiers,
			respMod:      opts.ResponseModifiers,
			config:       opts.Configuration,
			instrumenter: opts.Instrumenter,
		},
		baseURI: opts.BaseURI,
		ctx:     opts.Context,
	}
}

func (c clientInstance) Do(req *Request, resp *Response) error {
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
func (c clientInstance) UserAgent() string {
	return c.c.Name
}

// BaseURI returns client base URI.
func (c clientInstance) BaseURI() string {
	return c.baseURI
}

// WithModifiers returns a new client with specified context.
func (c clientInstance) WithContext(ctx context.Context) Client {
	return &clientInstance{
		client:  c.client,
		baseURI: c.baseURI,
		ctx:     ctx,
	}
}

// WithBaseURI returns a new client with specified base URI.
func (c clientInstance) WithBaseURI(uri string) Client {
	return &clientInstance{
		client:  c.client,
		baseURI: uri,
		ctx:     c.ctx,
	}
}

// WithConfiguration returns a new client with specific named configuration.
func (c clientInstance) WithConfiguration(name string) (Client, error) {
	cl, ok := c.config.Clients[name]
	if !ok {
		return nil, fmt.Errorf("client %q not found", name)
	}

	return c.WithBaseURI(cl.BaseURI), nil
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
