package http

import (
	"bytes"
	"fmt"
	"io"

	"github.com/goccy/go-json"
	"github.com/valyala/fasthttp"
)

// Response represents HTTP response.
//
// It is forbidden copying Response instances. Create new instances
// and use CopyTo instead.
//
// Response instance MUST NOT be used from concurrently running goroutines.
type Response struct {
	*fasthttp.Response
}

// Success returns true if the response status code is 2xx.
func (r Response) Success() bool {
	return r.StatusCode()/100 == 2
}

// Error if the response status code is not 2xx.
func (r Response) Error() error {
	if r.Success() {
		return nil
	}

	switch r.StatusCode() {
	case StatusForbidden:
		return ForbiddenError{}
	case StatusNotFound:
		return NotFoundError{}
	case StatusUnauthorized:
		return UnauthorizedError{}
	default:
		body, _ := r.BodyUncompressed()
		if len(body) > 0 && bytes.Equal(r.Header.ContentType(), strContentTypeJSON) {
			e := ErrorResponse{}
			if err := json.Unmarshal(body, &e); err == nil && len(e.Errors) > 0 {
				return e
			}
		}

		bodyLen := len(body)
		if bodyLen > 0 {
			if bodyLen > 100 {
				bodyLen = 100
			}

			body = append([]byte(": "), body[:bodyLen]...)
		}

		return fmt.Errorf("unexpected response status %d%s", r.StatusCode(), body)
	}
}

// ReadBody returns at most limit bytes of the response body and closes the body stream.
//
// Returns ErrBodyTooLarge with the truncated body if the body exceeds limit.
func (r Response) ReadBody(limit int) ([]byte, error) {
	stream := r.BodyStream()
	if stream == nil {
		body := r.Body()
		if len(body) > limit {
			return body[:limit], fasthttp.ErrBodyTooLarge
		}

		return body, nil
	}

	closer, ok := stream.(fasthttp.ReadCloserWithError)
	if !ok {
		_ = r.CloseBodyStream()

		return nil, fmt.Errorf("unexpected body stream type %T", stream)
	}

	body, err := io.ReadAll(io.LimitReader(stream, int64(limit)+1))
	if err != nil {
		_ = closer.CloseWithError(err)

		return body, err
	}

	if len(body) > limit {
		_ = closer.CloseWithError(fasthttp.ErrBodyTooLarge)

		return body[:limit], fasthttp.ErrBodyTooLarge
	}

	return body, r.CloseBodyStream()
}

// NewResponse returns a new response instance.
//
// The returned response must be released after use by calling ReleaseResponse.
func (c client) NewResponse() *Response {
	v := c.ResponsePool.Get()
	if v == nil {
		return &Response{
			Response: fasthttp.AcquireResponse(),
		}
	}

	res, _ := v.(*Response)
	res.Response = fasthttp.AcquireResponse()

	return res
}

// ReleaseResponse releases the response instance.
func (c client) ReleaseResponse(res *Response) {
	r := res.Response
	res.Response = nil

	fasthttp.ReleaseResponse(r)
	c.ResponsePool.Put(res)
}
