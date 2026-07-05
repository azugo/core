package http

// Method represents an HTTP request method.
type Method string

// HTTP request methods.
const (
	MethodGet     Method = "GET"     // RFC 9110, 9.3.1
	MethodHead    Method = "HEAD"    // RFC 9110, 9.3.2
	MethodPost    Method = "POST"    // RFC 9110, 9.3.3
	MethodPut     Method = "PUT"     // RFC 9110, 9.3.4
	MethodDelete  Method = "DELETE"  // RFC 9110, 9.3.5
	MethodConnect Method = "CONNECT" // RFC 9110, 9.3.6
	MethodOptions Method = "OPTIONS" // RFC 9110, 9.3.7
	MethodTrace   Method = "TRACE"   // RFC 9110, 9.3.8
	MethodPatch   Method = "PATCH"   // RFC 5789
	MethodQuery   Method = "QUERY"   // RFC 10008
)

// String returns the HTTP method as a string.
func (m Method) String() string {
	return string(m)
}

// IsGet returns true if the method is GET.
func (m Method) IsGet() bool {
	return m == MethodGet
}

// IsHead returns true if the method is HEAD.
func (m Method) IsHead() bool {
	return m == MethodHead
}

// IsPost returns true if the method is POST.
func (m Method) IsPost() bool {
	return m == MethodPost
}

// IsPut returns true if the method is PUT.
func (m Method) IsPut() bool {
	return m == MethodPut
}

// IsDelete returns true if the method is DELETE.
func (m Method) IsDelete() bool {
	return m == MethodDelete
}

// IsConnect returns true if the method is CONNECT.
func (m Method) IsConnect() bool {
	return m == MethodConnect
}

// IsOptions returns true if the method is OPTIONS.
func (m Method) IsOptions() bool {
	return m == MethodOptions
}

// IsTrace returns true if the method is TRACE.
func (m Method) IsTrace() bool {
	return m == MethodTrace
}

// IsPatch returns true if the method is PATCH.
func (m Method) IsPatch() bool {
	return m == MethodPatch
}

// IsQuery returns true if the method is QUERY.
func (m Method) IsQuery() bool {
	return m == MethodQuery
}
