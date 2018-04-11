package miniserver

import (
	"net/http"
)

type Response struct {
	file, contentType, serverDescription string
	body                                 []byte
	statusCode                           int
}

func newResponseBody(body string) *Response {
	response := newResponse()
	response.body = []byte(body)
	return response
}

func newResponseBodyBytes(body []byte) *Response {
	response := newResponse()
	response.body = body
	return response
}

func newResponseFile(file string) *Response {
	response := newResponse()
	response.file = file
	return response
}

func newResponse() *Response {
	return &Response{
		contentType:       ContentText,
		serverDescription: "Go MiniServer",
		statusCode:        http.StatusOK,
	}
}

func (response *Response) SetContentType(contentType string) {
	response.contentType = contentType
}

func (response *Response) SetStatusCode(code int) {
	response.statusCode = code
}

func (response *Response) SetServerDescription(description string) {
	response.serverDescription = description
}
