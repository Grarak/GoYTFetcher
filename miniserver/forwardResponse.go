package miniserver

import (
	"net/http"
	"bytes"
	"io/ioutil"
	"fmt"
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
	for key := range client.Header {
		uRequest.Header.Set(key, client.Header.Get(key))
	}

	if err != nil {
		errWriter()
		return
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

	flusher, ok := writer.(http.Flusher)
	if ok {
		for {
			buf := make([]byte, 8192)
			read, err := uResponse.Body.Read(buf)
			if read == 0 {
				break
			}
			_, err = writer.Write(buf[:read])
			if err != nil {
				break
			}
			flusher.Flush()
		}
	} else if body, err := ioutil.ReadAll(uResponse.Body); err == nil {
		fmt.Println(string(body))
		writer.Write(body)
	}
}