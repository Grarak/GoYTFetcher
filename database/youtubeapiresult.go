package database

import (
	"encoding/json"
	"net/http"
	"io/ioutil"

	"fmt"
)

type YoutubeThumbnail struct {
	Url string `json:"url"`
}

type YoutubeThumbnails struct {
	Default YoutubeThumbnail `json:"default"`
}

type YoutubeSnippet struct {
	Title      string            `json:"title"`
	Thumbnails YoutubeThumbnails `json:"thumbnails"`
}

type YoutubeContentDetails struct {
	Duration string `json:"duration"`
}

type YoutubeItem struct {
	Snippet        YoutubeSnippet        `json:"snippet"`
	ContentDetails YoutubeContentDetails `json:"contentDetails"`
	Id             string                `json:"id"`
}

type YoutubeResponse struct {
	Items []YoutubeItem `json:"items"`
}

func newYoutubeResponse(data []byte) (YoutubeResponse, error) {
	var response YoutubeResponse
	err := json.Unmarshal(data, &response)
	return response, err
}

func getYoutubeApiResponseItems(url string) (YoutubeResponse, error) {
	res, err := http.Get(url)
	if err != nil {
		return YoutubeResponse{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return YoutubeResponse{}, fmt.Errorf("failure")
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return YoutubeResponse{}, err
	}

	response, err := newYoutubeResponse(b)
	return response, err
}
