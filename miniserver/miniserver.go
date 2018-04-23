package miniserver

import (
	"io/ioutil"
	"net/http"
	"net"
	"strconv"

	"../utils"
	"strings"
	"fmt"
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
			if !utils.StringIsEmpty(res.contentType) {
				response.Header().Set("Content-Type", res.contentType)
			}
			response.Header().Set("Server", res.serverDescription)

			if utils.StringIsEmpty(res.file) {
				if utils.FileExists(res.file) {
					buf, err := ioutil.ReadFile(res.file)
					if err == nil {
						content = buf
					}
				}
			}

			rangeParser := func(headers http.Header, response []byte, ranges string) []byte {
				ranges = strings.Replace(ranges, "bytes=", "", 1)

				responseLength := len(response)
				middleIndex := strings.Index(ranges, "-")
				start, err := strconv.Atoi(ranges[:middleIndex])
				if err != nil {
					return response
				}
				end := responseLength - 1
				if middleIndex+1 < len(ranges) {
					end, err = strconv.Atoi(ranges[middleIndex+1:])
					if err != nil {
						return response
					}
					if end >= responseLength {
						end = responseLength - 1
					}
				}

				var finalResponse []byte
				finalResponse = append(finalResponse, response[start:end+1]...)

				headers.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, responseLength))
				return finalResponse
			}

			ranges := client.Header.Get("Range")

			statusCode := res.statusCode
			if statusCode == http.StatusOK &&
				strings.HasPrefix(ranges, "bytes=") &&
				strings.Contains(ranges, "-") {
				content = rangeParser(response.Header(), content, ranges)
				statusCode = http.StatusPartialContent
			}

			response.Header().Set("Accept-Ranges", "bytes")
			response.Header().Set("Content-Length", strconv.Itoa(len(content)))

			response.WriteHeader(statusCode)
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
