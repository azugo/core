package http

import (
	"strings"

	"github.com/valyala/fasthttp"
)

// ErrorResponseError is an error response error details.
type ErrorResponseError struct {
	Type    string `json:"type" yaml:"type" xml:"Type"`
	Message string `json:"message" yaml:"message" xml:"Message"`
}

// ErrorResponse represents an error response.
//
//nolint:errname
type ErrorResponse struct {
	Errors []*ErrorResponseError `json:"errors" yaml:"errors" xml:"Errors>Error"`
}

// Error implements error interface.
func (e ErrorResponse) Error() string {
	if len(e.Errors) == 0 {
		return "unknown error"
	}

	s := strings.Builder{}

	for i, err := range e.Errors {
		if i > 0 {
			s.WriteString("; ")
		}

		s.WriteString(err.Type)
		s.WriteString(": ")
		s.WriteString(err.Message)
	}

	return s.String()
}

// ResponseStatusCode is an interface that error can implement to return
// status code that will be set for the response.
type ResponseStatusCode interface {
	// StatusCode to be set for the response.
	StatusCode() int
}

// UnauthorizedError is an error that occurs when user is not authorized.
type UnauthorizedError struct{}

func (UnauthorizedError) Error() string {
	return "unauthorized"
}

func (UnauthorizedError) StatusCode() int {
	return fasthttp.StatusUnauthorized
}

// ForbiddenError is an error that occurs when user access is denied.
type ForbiddenError struct{}

func (ForbiddenError) Error() string {
	return "access forbidden"
}

func (ForbiddenError) StatusCode() int {
	return fasthttp.StatusForbidden
}

// NotFoundError is an error that occurs when searched resource is not found.
type NotFoundError struct {
	Resource string
}

func (e NotFoundError) Error() string {
	if e.Resource == "" {
		return "resource not found"
	}

	return e.Resource + " not found"
}

func (NotFoundError) StatusCode() int {
	return fasthttp.StatusNotFound
}
