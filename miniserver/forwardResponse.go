package miniserver

import (
	"bytes"
	"io"
	"net/http"
)

type ForwardResponse struct {
	u string
}

func NewForwardResponse(u string) *ForwardResponse {
	return &ForwardResponse{u}
}

func (forwardResponse *ForwardResponse) write(writer http.ResponseWriter, client *Client) {
	errWriter := func() {
		writer.WriteHeader(http.StatusNotFound)
	}

	uClient := &http.Client{}
	uRequest, err := http.NewRequest(client.Method, forwardResponse.u,
		bytes.NewReader(client.Request))

	if err != nil {
		errWriter()
		return
	}

	for key := range client.Header {
		uRequest.Header.Set(key, client.Header.Get(key))
	}

	uResponse, err := uClient.Do(uRequest)
	if err != nil {
		errWriter()
		return
	}
	defer uResponse.Body.Close()

	for key := range uResponse.Header {
		writer.Header().Set(key, uResponse.Header.Get(key))
	}
	writer.WriteHeader(uResponse.StatusCode)

	io.Copy(writer, uResponse.Body)
}
