package http

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"mime/multipart"

	"github.com/goccy/go-json"
	"github.com/valyala/fasthttp"
)

func (c client) call(req *Request) ([]byte, error) {
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

// Get performs a GET request to the specified URL.
func (c client) Get(url string, opt ...RequestOption) ([]byte, error) {
	req := c.NewRequest()
	if err := req.SetRequestURL(url); err != nil {
		return nil, err
	}

	req.apply(opt)

	return c.call(req)
}

// GetJSON performs a GET request to the specified URL and unmarshals the response into v.
func (c client) GetJSON(url string, v any, opt ...RequestOption) error {
	resp, err := c.Get(url, opt...)
	if err != nil {
		return err
	}

	if len(resp) > 0 {
		return json.Unmarshal(resp, v)
	}

	return nil
}

// Query performs a QUERY request to the specified URL.
//
// Request content must have a media type set using the WithHeader Content-Type option (RFC 10008, Section 2.1).
// From this point onward the body argument must not be changed.
func (c client) Query(url string, body []byte, opt ...RequestOption) ([]byte, error) {
	req := c.NewRequest()
	if err := req.SetRequestURL(url); err != nil {
		return nil, err
	}

	req.apply(opt)

	req.Header.SetMethod(MethodQuery.String())
	req.SetBodyRaw(body)

	return c.call(req)
}

// QueryJSON performs a QUERY request to the specified URL and unmarshals the response into v.
func (c client) QueryJSON(url string, body, v any, opt ...RequestOption) error {
	reqBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	opt = append([]RequestOption{WithHeader(HeaderContentType, ContentTypeJSON)}, opt...)

	resp, err := c.Query(url, reqBody, opt...)
	if err != nil {
		return err
	}

	if len(resp) > 0 {
		return json.Unmarshal(resp, v)
	}

	return nil
}

// QueryForm performs a QUERY request to the specified URL with the specified form values encoded with URL encoding.
func (c client) QueryForm(url string, form map[string][]string, opt ...RequestOption) ([]byte, error) {
	args := fasthttp.AcquireArgs()
	defer fasthttp.ReleaseArgs(args)

	for k, v := range form {
		for _, vv := range v {
			args.Add(k, vv)
		}
	}

	opt = append([]RequestOption{WithHeader(HeaderContentType, ContentTypeFormURLEncoded)}, opt...)

	return c.Query(url, args.QueryString(), opt...)
}

// Post performs a POST request to the specified URL.
//
// From this point onward the body argument must not be changed.
func (c client) Post(url string, body []byte, opt ...RequestOption) ([]byte, error) {
	req := c.NewRequest()
	if err := req.SetRequestURL(url); err != nil {
		return nil, err
	}

	req.apply(opt)

	req.Header.SetMethod(MethodPost.String())
	req.SetBodyRaw(body)

	return c.call(req)
}

// PostJSON performs a POST request to the specified URL and unmarshals the response into v.
func (c client) PostJSON(url string, body, v any, opt ...RequestOption) error {
	reqBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	opt = append([]RequestOption{WithHeader(HeaderContentType, ContentTypeJSON)}, opt...)

	resp, err := c.Post(url, reqBody, opt...)
	if err != nil {
		return err
	}

	if len(resp) > 0 {
		return json.Unmarshal(resp, v)
	}

	return nil
}

// PostForm performs a POST request to the specified URL with the specified form values encoded with URL encoding.
func (c client) PostForm(url string, form map[string][]string, opt ...RequestOption) ([]byte, error) {
	args := fasthttp.AcquireArgs()
	defer fasthttp.ReleaseArgs(args)

	for k, v := range form {
		for _, vv := range v {
			args.Add(k, vv)
		}
	}

	opt = append([]RequestOption{WithHeader(HeaderContentType, ContentTypeFormURLEncoded)}, opt...)

	return c.Post(url, args.QueryString(), opt...)
}

// PostMultipartForm performs a POST request to the specified URL with the specified multipart form values and files.
func (c client) PostMultipartForm(url string, form *multipart.Form, opt ...RequestOption) ([]byte, error) {
	req := c.NewRequest()
	req.Header.SetMethod(MethodPost.String())

	if err := req.SetRequestURL(url); err != nil {
		return nil, err
	}

	req.apply(opt)

	var bbuf [30]byte
	if _, err := io.ReadFull(rand.Reader, bbuf[:]); err != nil {
		return nil, err
	}

	boundary := hex.EncodeToString(bbuf[:])

	req.Header.SetMultipartFormBoundary(boundary)

	buf := c.BufferPool.Get()
	if err := fasthttp.WriteMultipartForm(buf, form, boundary); err != nil {
		c.BufferPool.Put(buf)

		return nil, err
	}

	req.SetBodyRaw(buf.Bytes())
	c.BufferPool.Put(buf)

	return c.call(req)
}

// Put performs a PUT request to the specified URL.
func (c client) Put(url string, body []byte, opt ...RequestOption) ([]byte, error) {
	req := c.NewRequest()
	if err := req.SetRequestURL(url); err != nil {
		return nil, err
	}

	req.apply(opt)

	req.Header.SetMethod(MethodPut.String())
	req.SetBodyRaw(body)

	return c.call(req)
}

// PutJSON performs a PUT request to the specified URL and unmarshals the response into v.
func (c client) PutJSON(url string, body, v any, opt ...RequestOption) error {
	reqBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	opt = append([]RequestOption{WithHeader(HeaderContentType, ContentTypeJSON)}, opt...)

	resp, err := c.Put(url, reqBody, opt...)
	if err != nil {
		return err
	}

	if len(resp) > 0 {
		return json.Unmarshal(resp, v)
	}

	return nil
}

// Patch performs a PATCH request to the specified URL.
func (c client) Patch(url string, body []byte, opt ...RequestOption) ([]byte, error) {
	req := c.NewRequest()
	if err := req.SetRequestURL(url); err != nil {
		return nil, err
	}

	req.apply(opt)

	req.Header.SetMethod(MethodPatch.String())
	req.SetBodyRaw(body)

	return c.call(req)
}

// PatchJSON performs a PATCH request to the specified URL and unmarshals the response into v.
func (c client) PatchJSON(url string, body, v any, opt ...RequestOption) error {
	reqBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	opt = append([]RequestOption{WithHeader(HeaderContentType, ContentTypeJSON)}, opt...)

	resp, err := c.Patch(url, reqBody, opt...)
	if err != nil {
		return err
	}

	if len(resp) > 0 {
		return json.Unmarshal(resp, v)
	}

	return nil
}

// Delete performs a DELETE request to the specified URL.
func (c client) Delete(url string, opt ...RequestOption) ([]byte, error) {
	req := c.NewRequest()
	if err := req.SetRequestURL(url); err != nil {
		return nil, err
	}

	req.apply(opt)

	req.Header.SetMethod(MethodDelete.String())

	return c.call(req)
}

// DeleteJSON performs a DELETE request to the specified URL and unmarshals the response into v.
func (c client) DeleteJSON(url string, body, v any, opt ...RequestOption) error {
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}

		if len(buf) > 0 {
			opt = append([]RequestOption{WithHeader(HeaderContentType, ContentTypeJSON), WithBody(buf)}, opt...)
		}
	}

	resp, err := c.Delete(url, opt...)
	if err != nil {
		return err
	}

	if len(resp) > 0 {
		return json.Unmarshal(resp, v)
	}

	return nil
}
