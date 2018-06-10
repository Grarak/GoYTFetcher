package miniserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Grarak/GoYTFetcher/utils"
)

type SimpleResponse struct {
	contentType, serverDescription string
	headers                        http.Header
	statusCode                     int
	readHolder                     rangeReadHolder
}

type rangeReadHolder interface {
	Size() int64
	Close() error
	io.ReaderAt
}

type rangeReadHolderBytes struct {
	bytesReader *bytes.Reader
}

type rangeReadHolderFile struct {
	file *os.File
}

type rangeReader struct {
	start, end, size int64
	holder           rangeReadHolder
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
	return client.ResponseBodyBytes([]byte(body))
}

func (client *Client) ResponseBodyBytes(body []byte) *SimpleResponse {
	return client.ResponseReader(&rangeReadHolderBytes{bytes.NewReader(body)})
}

func (client *Client) ResponseFile(file string) *SimpleResponse {
	reader, _ := os.Open(file)
	response := client.ResponseReader(&rangeReadHolderFile{reader})
	response.contentType = getContentTypeForFile(file)
	return response
}

func (client *Client) ResponseReader(readHolder rangeReadHolder) *SimpleResponse {
	response := newResponse()
	response.readHolder = readHolder
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
	if !utils.StringIsEmpty(response.contentType) {
		writer.Header().Set("Content-Type", response.contentType)
	}
	writer.Header().Set("Server", response.serverDescription)
	for key := range response.headers {
		writer.Header().Set(key, response.headers.Get(key))
	}

	readerSize := response.readHolder.Size()
	contentLength := readerSize
	start, end := int64(0), readerSize-1

	ranges := client.Header.Get("Range")
	statusCode := response.statusCode
	if statusCode == http.StatusOK &&
		strings.HasPrefix(ranges, "bytes=") &&
		strings.Contains(ranges, "-") {
		start, end = rangeParser(ranges)
		if end < 0 {
			end = readerSize - 1
		}

		if start >= readerSize-1 {
			start = readerSize - 1
		}
		if end >= readerSize-1 {
			end = readerSize - 1
		}
		if end < start {
			end = start
		}
		writer.Header().Set("Content-Range",
			fmt.Sprintf("bytes %d-%d/%d", start, end, readerSize))
		statusCode = http.StatusPartialContent

		contentLength = end - start + 1
	}

	reader := &rangeReader{
		start, end, readerSize,
		response.readHolder,
	}
	defer reader.Close()

	writer.Header().Set("Accept-Ranges", "bytes")
	writer.Header().Set("Content-Length", fmt.Sprint(contentLength))

	writer.WriteHeader(statusCode)
	io.Copy(writer, reader)
}

func rangeParser(ranges string) (int64, int64) {
	ranges = strings.Replace(ranges, "bytes=", "", 1)

	middleIndex := strings.Index(ranges, "-")
	start, err := strconv.ParseInt(ranges[:middleIndex], 10, 64)
	if err != nil {
		return 0, 0
	}

	end := int64(-1)
	if middleIndex < len(ranges)-1 {
		end, err = strconv.ParseInt(ranges[middleIndex+1:], 10, 64)
		if err != nil {
			return start, 0
		}
	}
	return start, end
}

func (rangeReadHolderBytes *rangeReadHolderBytes) Size() int64 {
	return rangeReadHolderBytes.bytesReader.Size()
}

func (rangeReadHolderBytes *rangeReadHolderBytes) ReadAt(p []byte, off int64) (n int, err error) {
	return rangeReadHolderBytes.bytesReader.ReadAt(p, off)
}

func (rangeReadHolderBytes *rangeReadHolderBytes) Close() error {
	return nil
}

func (rangeReadHolderFile *rangeReadHolderFile) Size() int64 {
	info, err := rangeReadHolderFile.file.Stat()
	if err != nil {
		return 0
	}
	return info.Size()
}

func (rangeReadHolderFile *rangeReadHolderFile) ReadAt(p []byte, off int64) (n int, err error) {
	return rangeReadHolderFile.file.ReadAt(p, off)
}

func (rangeReadHolderFile *rangeReadHolderFile) Close() error {
	return rangeReadHolderFile.file.Close()
}

func (rangeReader *rangeReader) Read(b []byte) (n int, err error) {
	if rangeReader.start >= rangeReader.size {
		return 0, io.EOF
	}

	read, _ := rangeReader.holder.ReadAt(b, rangeReader.start)
	newStart := rangeReader.start + int64(read)

	if newStart > rangeReader.end {
		read = int(rangeReader.end-rangeReader.start) + 1
		rangeReader.start = rangeReader.size
	}

	rangeReader.start = newStart
	return read, nil
}

func (rangeReader *rangeReader) Close() error {
	return rangeReader.holder.Close()
}
