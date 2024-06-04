package http

import (
	"bytes"
	"context"
	"mime/multipart"
	"strings"
	"testing"
	"time"

	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestUserAgent(t *testing.T) {
	c := NewClient()
	qt.Check(t, qt.Equals(c.UserAgent(), "Azugo/dev"))
}

func TestGetRequest(t *testing.T) {
	s := newTestHttpServer()
	s.Handler = func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Request.Header.Method()) != fasthttp.MethodGet {
			ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			return
		}

		if string(ctx.QueryArgs().Peek("name")) != "John Doe" {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}

		ctx.SetBodyString("Hello World")
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
	s.Start()
	defer s.Stop()

	c := NewClient(s.DialContext(), BaseURL("http://localhost:8080"))
	body, err := c.Get("/", WithQueryArg("name", "Test"), WithQueryArg("name", "John Doe", true))
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(string(body), "Hello World"))
}

func TestGetJSONRequest(t *testing.T) {
	s := newTestHttpServer()
	s.Handler = func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Request.Header.Method()) != fasthttp.MethodGet {
			ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			return
		}

		ctx.SetContentTypeBytes(strContentTypeJSON)
		ctx.SetBodyString(`{"message":"Hello World"}`)
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
	s.Start()
	defer s.Stop()

	c := NewClient(s.DialContext())

	ret := struct {
		Message string `json:"message"`
	}{}

	err := c.GetJSON("http://localhost:8080", &ret)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(ret.Message, "Hello World"))
}

func TestPostRequest(t *testing.T) {
	s := newTestHttpServer()
	s.Handler = func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Request.Header.Method()) != fasthttp.MethodPost {
			ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			return
		}

		if !bytes.Equal(ctx.Request.Header.ContentType(), strContentTypeJSON) {
			ctx.SetStatusCode(fasthttp.StatusUnprocessableEntity)
			return
		}

		if string(ctx.Request.Body()) != `{"message":"Hello World"}` {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}

		ctx.SetBodyString("OK")
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
	s.Start()
	defer s.Stop()

	c := NewClient(s.DialContext())
	resp, err := c.Post("http://localhost:8080", []byte(`{"message":"Hello World"}`), WithHeader(fasthttp.HeaderContentType, "application/json"))
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(string(resp), "OK"))
}

func TestPostJSONRequest(t *testing.T) {
	s := newTestHttpServer()
	s.Handler = func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Request.Header.Method()) != fasthttp.MethodPost {
			ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			return
		}

		if !bytes.Equal(ctx.Request.Header.ContentType(), strContentTypeJSON) {
			ctx.SetStatusCode(fasthttp.StatusUnprocessableEntity)
			return
		}

		if string(ctx.Request.Body()) != `{"message":"Hello World"}` {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}

		ctx.SetContentTypeBytes(strContentTypeJSON)
		ctx.SetBodyString(`{"status":true}`)
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
	s.Start()
	defer s.Stop()

	c := NewClient(s.DialContext())
	req := struct {
		Message string `json:"message"`
	}{
		Message: "Hello World",
	}

	resp := struct {
		Status bool `json:"status"`
	}{}

	err := c.PostJSON("http://localhost:8080", req, &resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(resp.Status))
}

func TestPostFormRequest(t *testing.T) {
	s := newTestHttpServer()
	s.Handler = func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Request.Header.Method()) != fasthttp.MethodPost {
			ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			return
		}

		if string(ctx.Request.Header.ContentType()) != "application/x-www-form-urlencoded" {
			ctx.SetStatusCode(fasthttp.StatusUnprocessableEntity)
			return
		}

		if string(ctx.Request.PostArgs().Peek("message")) != "Hello World" {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}

		ctx.SetBodyString("OK")
		ctx.SetStatusCode(fasthttp.StatusOK)
	}

	s.Start()
	defer s.Stop()

	c := NewClient(s.DialContext())
	resp, err := c.PostForm("http://localhost:8080", map[string][]string{"message": {"Hello World"}})
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(string(resp), "OK"))
}

func TestPostMultipartFormRequest(t *testing.T) {
	s := newTestHttpServer()
	s.Handler = func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Request.Header.Method()) != fasthttp.MethodPost {
			ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			return
		}

		if !strings.HasPrefix(string(ctx.Request.Header.ContentType()), "multipart/form-data") {
			ctx.SetStatusCode(fasthttp.StatusUnprocessableEntity)
			return
		}

		form, err := ctx.Request.MultipartForm()
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			return
		}
		if form.Value["message"][0] != "Hello World" {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}

		ctx.SetBodyString("OK")
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
	s.Start()
	defer s.Stop()

	c := NewClient(s.DialContext())
	form := new(multipart.Form)
	form.Value = map[string][]string{
		"message": {"Hello World"},
	}

	resp, err := c.PostMultipartForm("http://localhost:8080", form)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(string(resp), "OK"))
}

func TestPutRequest(t *testing.T) {
	s := newTestHttpServer()
	s.Handler = func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Request.Header.Method()) != fasthttp.MethodPut {
			ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			return
		}

		if !bytes.Equal(ctx.Request.Header.ContentType(), strContentTypeJSON) {
			ctx.SetStatusCode(fasthttp.StatusUnprocessableEntity)
			return
		}

		if string(ctx.Request.Body()) != `{"message":"Hello World"}` {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}

		ctx.SetBodyString("OK")
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
	s.Start()
	defer s.Stop()

	c := NewClient(s.DialContext())
	resp, err := c.Put("http://localhost:8080", []byte(`{"message":"Hello World"}`), WithHeader(fasthttp.HeaderContentType, "application/json"))
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(string(resp), "OK"))
}

func TestPutJSONRequest(t *testing.T) {
	s := newTestHttpServer()
	s.Handler = func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Request.Header.Method()) != fasthttp.MethodPut {
			ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			return
		}

		if !bytes.Equal(ctx.Request.Header.ContentType(), strContentTypeJSON) {
			ctx.SetStatusCode(fasthttp.StatusUnprocessableEntity)
			return
		}

		if string(ctx.Request.Body()) != `{"message":"Hello World"}` {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}

		ctx.SetContentTypeBytes(strContentTypeJSON)
		ctx.SetBodyString(`{"status":true}`)
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
	s.Start()
	defer s.Stop()

	c := NewClient(s.DialContext())
	req := struct {
		Message string `json:"message"`
	}{
		Message: "Hello World",
	}

	resp := struct {
		Status bool `json:"status"`
	}{}

	err := c.PutJSON("http://localhost:8080", req, &resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(resp.Status))
}

func TestPatchRequest(t *testing.T) {
	s := newTestHttpServer()
	s.Handler = func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Request.Header.Method()) != fasthttp.MethodPatch {
			ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			return
		}

		if !bytes.Equal(ctx.Request.Header.ContentType(), strContentTypeJSON) {
			ctx.SetStatusCode(fasthttp.StatusUnprocessableEntity)
			return
		}

		if string(ctx.Request.Body()) != `{"message":"Hello World"}` {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}

		ctx.SetBodyString("OK")
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
	s.Start()
	defer s.Stop()

	c := NewClient(s.DialContext())
	resp, err := c.Patch("http://localhost:8080", []byte(`{"message":"Hello World"}`), WithHeader(fasthttp.HeaderContentType, "application/json"))
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(string(resp), "OK"))
}

func TestPatchJSONRequest(t *testing.T) {
	s := newTestHttpServer()
	s.Handler = func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Request.Header.Method()) != fasthttp.MethodPatch {
			ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			return
		}

		if !bytes.Equal(ctx.Request.Header.ContentType(), strContentTypeJSON) {
			ctx.SetStatusCode(fasthttp.StatusUnprocessableEntity)
			return
		}

		if string(ctx.Request.Body()) != `{"message":"Hello World"}` {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}

		ctx.SetContentTypeBytes(strContentTypeJSON)
		ctx.SetBodyString(`{"status":true}`)
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
	s.Start()
	defer s.Stop()

	c := NewClient(s.DialContext())
	req := struct {
		Message string `json:"message"`
	}{
		Message: "Hello World",
	}

	resp := struct {
		Status bool `json:"status"`
	}{}

	err := c.PatchJSON("http://localhost:8080", req, &resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(resp.Status))
}

func TestDeleteRequest(t *testing.T) {
	s := newTestHttpServer()
	s.Handler = func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Request.Header.Method()) != fasthttp.MethodDelete {
			ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			return
		}

		ctx.SetStatusCode(fasthttp.StatusOK)
	}
	s.Start()
	defer s.Stop()

	c := NewClient(s.DialContext())
	err := c.Delete("http://localhost:8080")
	qt.Assert(t, qt.IsNil(err))
}

func TestWithAuthorizationHeader(t *testing.T) {
	s := newTestHttpServer()
	s.Handler = func(ctx *fasthttp.RequestCtx) {
		if auth := ctx.Request.Header.Peek("Authorization"); auth == nil || string(auth) != "Bearer 123456" {
			ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			return
		}
		ctx.SetBodyString("Hello World")
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
	s.Start()
	defer s.Stop()

	c := NewClient(s.DialContext())
	body, err := c.Get("http://localhost:8080", WithHeader("Authorization", "Bearer 123456"))
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(string(body), "Hello World"))
}

func TestInstrumentation(t *testing.T) {
	delay := 250 * time.Millisecond

	s := newTestHttpServer()
	s.Handler = func(ctx *fasthttp.RequestCtx) {
		time.Sleep(delay)
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
	s.Start()
	defer s.Stop()

	c := NewClient(s.DialContext(), Instrumenter(func(ctx context.Context, op string, args ...any) func(err error) {
		qt.Check(t, qt.Equals(op, InstrumentationRequest))

		s := time.Now()

		return func(err error) {
			qt.Check(t, qt.IsNil(err))
			qt.Check(t, qt.IsTrue(time.Since(s) > delay))
		}
	}))

	_, _ = c.Get("http://localhost:8080")
}

func TestClientRequestReuse(t *testing.T) {
	s := newTestHttpServer()
	s.Handler = func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
	s.Start()
	defer s.Stop()

	c := NewClient(s.DialContext(), BaseURL("http://localhost:8080"))
	body, err := c.Get("/")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(string(body), ""))

	body, err = c.Get("/")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(string(body), ""))
}

func TestClientWithConfiguration(t *testing.T) {
	s := newTestHttpServer()
	s.Handler = func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
	s.Start()
	defer s.Stop()

	cfg := &Configuration{
		Clients: map[string]NamedClient{
			"test": {
				BaseURL: "http://localhost:8080",
			},
		},
	}

	c, err := NewClient(s.DialContext(), cfg).WithConfiguration("test")
	qt.Assert(t, qt.IsNil(err))

	body, err := c.Get("/")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(string(body), ""))
}
