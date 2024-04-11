package http

import (
	"context"
	"net"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

// testHttpServer is a test HTTP server for testing HTTP client.
type testHttpServer struct {
	ln *fasthttputil.InmemoryListener

	Handler fasthttp.RequestHandler
}

func newTestHttpServer() *testHttpServer {
	return &testHttpServer{}
}

func (s *testHttpServer) Start() {
	server := &fasthttp.Server{
		NoDefaultServerHeader:        true,
		Handler:                      s.Handler,
		StreamRequestBody:            true,
		DisablePreParseMultipartForm: true,
	}
	ln := fasthttputil.NewInmemoryListener()
	go func() {
		if err := server.Serve(ln); err != nil {
			panic(err)
		}
	}()
	s.ln = ln
}

func (s *testHttpServer) Stop() {
	if s.ln != nil {
		s.ln.Close()
	}
}

func (s *testHttpServer) DialContext() DialContextFunc {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return s.ln.Dial()
	}
}
