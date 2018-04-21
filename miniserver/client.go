package miniserver

import (
	"net/http"
	"io/ioutil"
	"net/url"
	"encoding/json"
	"../utils"
	"strings"
)

type Client struct {
	Url, Method, IPAddr string
	Request             []byte
	Header              http.Header
	Queries             url.Values
}

func newClient(request *http.Request) *Client {
	defer request.Body.Close()

	body, _ := ioutil.ReadAll(request.Body)

	return &Client{
		request.URL.Path,
		request.Method,
		request.RemoteAddr[:strings.LastIndex(request.RemoteAddr, ":")],
		body,
		request.Header,
		request.Form,
	}
}

func (client *Client) IsContentJson() bool {
	return strings.HasPrefix(client.Header.Get("Content-Type"), ContentJson)
}

func (client *Client) ResponseBody(body string) *Response {
	return newResponseBody(body)
}

func (client *Client) ResponseBodyBytes(body []byte) *Response {
	return newResponseBodyBytes(body)
}

func (client *Client) ResponseFile(file string) *Response {
	return newResponseFile(file)
}

func (client *Client) CreateJsonResponse(data interface{}) *Response {
	b, err := json.Marshal(data)
	utils.Panic(err)

	response := client.ResponseBody(string(b))
	response.SetContentType(ContentJson)
	return response
}

func (client *Client) CreateResponse(statusCode int) *Response {
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
