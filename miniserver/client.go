package miniserver

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/Grarak/GoYTFetcher/utils"
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
	ipAddr := request.RemoteAddr[:strings.LastIndex(request.RemoteAddr, ":")]
	if cfConnectionIP := request.Header.Get("Cf-Connecting-Ip"); !utils.StringIsEmpty(cfConnectionIP) {
		ipAddr = cfConnectionIP
	}

	return &Client{
		request.URL.Path,
		request.Method,
		ipAddr,
		body,
		request.Header,
		request.Form,
	}
}

func (client *Client) IsContentJson() bool {
	return strings.HasPrefix(client.Header.Get("Content-Type"), ContentJson)
}
