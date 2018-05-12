package miniserver

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/Grarak/GoYTFetcher/utils"
)

type SimpleResponse struct {
	file, contentType, serverDescription string
	body                                 []byte
	headers                              http.Header
	statusCode                           int
}

func newResponse() *SimpleResponse {
	return &SimpleResponse{
		contentType:       ContentText,
		serverDescription: "Go MiniServer",
		headers:           make(map[string][]string),
		statusCode:        http.StatusOK,
	}
}

func (client *Client) ResponseBody(body string) *SimpleResponse {
	response := newResponse()
	response.body = []byte(body)
	return response
}

func (client *Client) ResponseBodyBytes(body []byte) *SimpleResponse {
	response := newResponse()
	response.body = body
	return response
}

func (client *Client) ResponseFile(file string) *SimpleResponse {
	response := newResponse()
	response.file = file
	return response
}

func (client *Client) CreateJsonResponse(data interface{}) *SimpleResponse {
	b, err := json.Marshal(data)
	utils.Panic(err)

	response := client.ResponseBody(string(b))
	response.SetContentType(ContentJson)
	return response
}

func (client *Client) CreateResponse(statusCode int) *SimpleResponse {
	type ResponseStruct struct {
		StatusCode int    `json:"statuscode"`
		Path       string `json:"path"`
	}
	b, err := json.Marshal(ResponseStruct{statusCode,
		client.Url})
	utils.Panic(err)

	response := client.ResponseBody(string(b))
	if statusCode == utils.StatusNoError {
		response.SetStatusCode(http.StatusOK)
	} else {
		response.SetStatusCode(http.StatusNotFound)
	}
	response.SetContentType(ContentJson)

	return response
}

func (response *SimpleResponse) SetContentType(contentType string) {
	response.contentType = contentType
}

func (response *SimpleResponse) SetStatusCode(code int) {
	response.statusCode = code
}

func (response *SimpleResponse) SetHeader(key, value string) {
	response.headers.Set(key, value)
}

func (response *SimpleResponse) write(writer http.ResponseWriter, client *Client) {
	content := response.body
	if !utils.StringIsEmpty(response.contentType) {
		writer.Header().Set("Content-Type", response.contentType)
	}
	writer.Header().Set("Server", response.serverDescription)
	for key := range response.headers {
		writer.Header().Set(key, response.headers.Get(key))
	}

	if utils.StringIsEmpty(response.file) {
		if utils.FileExists(response.file) {
			buf, err := ioutil.ReadFile(response.file)
			if err == nil {
				content = buf
			}
		}
	}

	ranges := client.Header.Get("Range")
	statusCode := response.statusCode
	if statusCode == http.StatusOK &&
		strings.HasPrefix(ranges, "bytes=") &&
		strings.Contains(ranges, "-") {
		partContent, contentRange := rangeParser(content, ranges)
		content = partContent
		writer.Header().Set("Content-Range", contentRange)
		statusCode = http.StatusPartialContent
	}

	writer.Header().Set("Accept-Ranges", "bytes")
	writer.Header().Set("Content-Length", strconv.Itoa(len(content)))

	writer.WriteHeader(statusCode)
	writer.Write(content)
}
