package miniserver

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type Response interface {
	write(writer http.ResponseWriter, client *Client)
}

func rangeParser(response []byte, ranges string) ([]byte, string) {
	ranges = strings.Replace(ranges, "bytes=", "", 1)

	responseLength := len(response)
	middleIndex := strings.Index(ranges, "-")
	start, err := strconv.Atoi(ranges[:middleIndex])
	if err != nil {
		return response, ""
	}
	if start >= responseLength {
		start = responseLength - 1
	}

	end := responseLength - 1
	if middleIndex+1 < len(ranges) {
		end, err = strconv.Atoi(ranges[middleIndex+1:])
		if err != nil {
			return response, ""
		}
		if end >= responseLength {
			end = responseLength - 1
		}
	}

	var finalResponse []byte
	finalResponse = append(finalResponse, response[start:end+1]...)

	return finalResponse, fmt.Sprintf("bytes %d-%d/%d", start, end, responseLength)
}
