package miniserver

import (
	"net/http"
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
	ContentOgg        = "audio/vorbis"
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

func (miniserver *MiniServer) StartListening(callback func(client *Client) Response) {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()

		request.ParseForm()
		client := newClient(request)

		res := callback(client)
		if res == nil {
			writer.WriteHeader(http.StatusNotFound)
			writer.Write([]byte("Not found"))
		} else {
			res.write(writer, client)
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
