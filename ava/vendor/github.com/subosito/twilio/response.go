package twilio

import (
	"net/http"
)

// Wraps http.Response. So we can add more functionalities later.
type Response struct {
	*http.Response
	Pagination
}

func NewResponse(r *http.Response) *Response {
	response := &Response{Response: r}
	return response
}
