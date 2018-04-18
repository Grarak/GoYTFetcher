package miniserver

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"net"
	"strconv"

	"../utils"
)

const (
	ContentText       = "text/plain"
	ContentHtml       = "text/html"
	ContentJson       = "application/json"
	ContentJavascript = "text/javascript"
	ContentCss        = "text/css"
	ContentXIcon      = "image/x-icon"
	ContentSVG        = "image/svg+xml"
	ContentWebm       = "audio/webm"
)

type MiniServer struct {
	port     int
	listener net.Listener
}

func NewServer(port int) *MiniServer {
	return &MiniServer{
		port: port,
	}
}

func (miniserver *MiniServer) StartListening(callback func(client *Client) *Response) {
	http.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()

		request.ParseForm()
		client := newClient(request)

		res := callback(client)
		if res == nil {
			response.WriteHeader(http.StatusNotFound)
			response.Write([]byte("Not found"))
		} else {
			content := res.body
			response.Header().Set("Content-Type", fmt.Sprintf("%s", res.contentType))
			response.Header().Set("Server", res.serverDescription)

			if len(res.file) > 0 {
				if _, err := os.Stat(res.file); err == nil {
					buf, err := ioutil.ReadFile(res.file)
					if err == nil {
						content = buf
					}
				}
			}

			response.Header().Set("Content-Length", strconv.Itoa(len(content)))
			response.WriteHeader(res.statusCode)
			response.Write(content)
		}
	})

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(miniserver.port))
	utils.Panic(err)
	miniserver.listener = listener
	http.Serve(listener, nil)
}

func (miniserver *MiniServer) StopListening() {
	if miniserver.listener != nil {
		miniserver.listener.Close()
	}
}
