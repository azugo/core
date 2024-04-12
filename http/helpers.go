package http

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"mime/multipart"

	"github.com/goccy/go-json"
	"github.com/valyala/fasthttp"
)

func (c clientInstance) call(req *Request) ([]byte, error) {
	resp := c.NewResponse()
	defer c.ReleaseResponse(resp)

	err := c.Do(req, resp)
	c.ReleaseRequest(req)

	if err != nil {
		return nil, err
	}

	if err := resp.Error(); err != nil {
		return nil, err
	}

	body, err := resp.BodyUncompressed()
	if err != nil {
		return nil, err
	}

	return body, nil
}

// Get performs a GET request to the specified URI.
func (c clientInstance) Get(uri string, opt ...RequestOption) ([]byte, error) {
	req := c.NewRequest()
	if err := req.SetRequestURI(uri); err != nil {
		return nil, err
	}

	req.apply(opt)

	return c.call(req)
}

// GetJSON performs a GET request to the specified URI and unmarshals the response into v.
func (c clientInstance) GetJSON(uri string, v any, opt ...RequestOption) error {
	resp, err := c.Get(uri, opt...)
	if err != nil {
		return err
	}

	if len(resp) > 0 {
		return json.UnmarshalContext(c.ctx, resp, v)
	}

	return nil
}

// Post performs a POST request to the specified URI.
//
// From this point onward the body argument must not be changed.
func (c clientInstance) Post(uri string, body []byte, opt ...RequestOption) ([]byte, error) {
	req := c.NewRequest()
	if err := req.SetRequestURI(uri); err != nil {
		return nil, err
	}

	req.apply(opt)

	req.Header.SetMethod(fasthttp.MethodPost)
	req.SetBodyRaw(body)

	return c.call(req)
}

// PostJSON performs a POST request to the specified URI and unmarshals the response into v.
func (c clientInstance) PostJSON(uri string, body, v any, opt ...RequestOption) error {
	reqBody, err := json.MarshalContext(c.ctx, body)
	if err != nil {
		return err
	}

	opt = append([]RequestOption{WithHeader(fasthttp.HeaderContentType, "application/json")}, opt...)

	resp, err := c.Post(uri, reqBody, opt...)
	if err != nil {
		return err
	}

	if len(resp) > 0 {
		return json.UnmarshalContext(c.ctx, resp, v)
	}

	return nil
}

// PostForm performs a POST request to the specified URI with the specified form values encoded with URL encoding.
func (c clientInstance) PostForm(uri string, form map[string][]string, opt ...RequestOption) ([]byte, error) {
	args := fasthttp.AcquireArgs()
	defer fasthttp.ReleaseArgs(args)

	for k, v := range form {
		for _, vv := range v {
			args.Add(k, vv)
		}
	}

	opt = append([]RequestOption{WithHeader(fasthttp.HeaderContentType, "application/x-www-form-urlencoded")}, opt...)

	return c.Post(uri, args.QueryString(), opt...)
}

// PostMultipartForm performs a POST request to the specified URI with the specified multipart form values and files.
func (c clientInstance) PostMultipartForm(uri string, form *multipart.Form, opt ...RequestOption) ([]byte, error) {
	req := c.NewRequest()
	req.Header.SetMethod(fasthttp.MethodPost)

	if err := req.SetRequestURI(uri); err != nil {
		return nil, err
	}

	req.apply(opt)

	var bbuf [30]byte
	if _, err := io.ReadFull(rand.Reader, bbuf[:]); err != nil {
		return nil, err
	}

	boundary := hex.EncodeToString(bbuf[:])

	req.Header.SetMultipartFormBoundary(boundary)

	buf := c.bufferPool.Get()
	if err := fasthttp.WriteMultipartForm(buf, form, boundary); err != nil {
		c.bufferPool.Put(buf)

		return nil, err
	}

	req.SetBodyRaw(buf.Bytes())
	c.bufferPool.Put(buf)

	return c.call(req)
}

// Put performs a PUT request to the specified URI.
func (c clientInstance) Put(uri string, body []byte, opt ...RequestOption) ([]byte, error) {
	req := c.NewRequest()
	if err := req.SetRequestURI(uri); err != nil {
		return nil, err
	}

	req.apply(opt)

	req.Header.SetMethod(fasthttp.MethodPut)
	req.SetBodyRaw(body)

	return c.call(req)
}

// PutJSON performs a PUT request to the specified URI and unmarshals the response into v.
func (c clientInstance) PutJSON(uri string, body, v any, opt ...RequestOption) error {
	reqBody, err := json.MarshalContext(c.ctx, body)
	if err != nil {
		return err
	}

	opt = append([]RequestOption{WithHeader(fasthttp.HeaderContentType, "application/json")}, opt...)

	resp, err := c.Put(uri, reqBody, opt...)
	if err != nil {
		return err
	}

	if len(resp) > 0 {
		return json.UnmarshalContext(c.ctx, resp, v)
	}

	return nil
}

// Patch performs a PATCH request to the specified URI.
func (c clientInstance) Patch(uri string, body []byte, opt ...RequestOption) ([]byte, error) {
	req := c.NewRequest()
	if err := req.SetRequestURI(uri); err != nil {
		return nil, err
	}

	req.apply(opt)

	req.Header.SetMethod(fasthttp.MethodPatch)
	req.SetBodyRaw(body)

	return c.call(req)
}

// PatchJSON performs a PATCH request to the specified URI and unmarshals the response into v.
func (c clientInstance) PatchJSON(uri string, body, v any, opt ...RequestOption) error {
	reqBody, err := json.MarshalContext(c.ctx, body)
	if err != nil {
		return err
	}

	opt = append([]RequestOption{WithHeader(fasthttp.HeaderContentType, "application/json")}, opt...)

	resp, err := c.Patch(uri, reqBody, opt...)
	if err != nil {
		return err
	}

	if len(resp) > 0 {
		return json.UnmarshalContext(c.ctx, resp, v)
	}

	return nil
}

// Delete performs a DELETE request to the specified URI.
func (c clientInstance) Delete(uri string, opt ...RequestOption) error {
	req := c.NewRequest()
	if err := req.SetRequestURI(uri); err != nil {
		return err
	}

	req.apply(opt)

	req.Header.SetMethod(fasthttp.MethodDelete)

	resp := c.NewResponse()
	defer c.ReleaseResponse(resp)

	err := c.Do(req, resp)
	c.ReleaseRequest(req)

	if err != nil {
		return err
	}

	return resp.Error()
}
