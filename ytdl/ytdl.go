package ytdl

import (
	"time"
	"net/url"
	"net/http"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"strconv"
	"encoding/json"
	"bytes"
	"github.com/PuerkitoBio/goquery"
	"../utils"
	"../logger"
	"os"
)

const youtubeBaseURL = "https://www.youtube.com/watch"

type Ytdl struct {
	jsonRegex *regexp.Regexp
}

func NewYtdl() Ytdl {
	return Ytdl{
		regexp.MustCompile("ytplayer.config = (.*?);ytplayer.load"),
	}
}

// VideoInfo contains the info a youtube video
type VideoInfo struct {
	// The video ID
	ID string `json:"id"`
	// The video title
	Title string `json:"title"`
	// Duration of the video
	Duration time.Duration
}

func (ytdl Ytdl) GetVideoInfoFromID(id string) (*VideoInfo, error) {
	u, _ := url.ParseRequestURI(youtubeBaseURL)
	values := u.Query()
	values.Set("v", id)
	u.RawQuery = values.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return ytdl.getVideoInfoFromHTML(id, body)
}

func (ytdl Ytdl) getVideoInfoFromHTML(id string, html []byte) (*VideoInfo, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, err
	}

	info := &VideoInfo{}

	// extract description and title
	info.Title = strings.TrimSpace(doc.Find("#eow-title").Text())
	info.ID = id

	// match json in javascript
	matches := ytdl.jsonRegex.FindSubmatch(html)
	var jsonConfig map[string]interface{}
	if len(matches) > 1 {
		err = json.Unmarshal(matches[1], &jsonConfig)
		if err != nil {
			return nil, err
		}
	}

	inf, ok := jsonConfig["args"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error no args in json %s", id)
	}
	if status, ok := inf["status"].(string); ok && status == "fail" {
		return nil, fmt.Errorf("error %d:%s", inf["errorcode"], inf["reason"])
	}

	if length, ok := inf["length_seconds"].(string); ok {
		if duration, err := strconv.ParseInt(length, 10, 64); err == nil {
			info.Duration = time.Second * time.Duration(duration)
		} else {
			logger.I(fmt.Sprintf("Unable to parse duration string: %s", length))
		}
	} else {
		logger.E("Unable to extract duration")
	}
	return info, nil
}

func (info *VideoInfo) GetThumbnailURL(quality ThumbnailQuality) *url.URL {
	u, _ := url.Parse(fmt.Sprintf("http://img.youtube.com/vi/%s/%s.jpg",
		info.ID, quality))
	return u
}

func (info *VideoInfo) GetDownloadURL(youtubeDL string) (string, error) {
	return utils.ExecuteCmd(youtubeDL, "--get-url", "--extract-audio",
		"--audio-format", "vorbis", info.ID)
}

func (info *VideoInfo) GetDownloadURLWorst(youtubeDL string) (string, error) {
	return utils.ExecuteCmd(youtubeDL, "--get-url", "-f", "worstaudio", info.ID)
}

func (info *VideoInfo) Download(path, youtubeDL, ffmpeg string) (string, error) {
	destination := path + "/" + info.ID
	destinationTmp := destination + "-tmp.%(ext)s"

	output, err := utils.ExecuteCmd(youtubeDL, "--extract-audio",
		"--audio-format", "vorbis", "--output", destinationTmp, info.ID)
	if err != nil {
		return "", err
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "[ffmpeg] Destination:") {
			destinationTmp = line[strings.Index(line, path):]
			break
		}
	}

	destination = strings.Replace(destinationTmp, info.ID+"-tmp", info.ID, 1)
	_, err = utils.ExecuteCmd(ffmpeg, "-y", "-i", destinationTmp, "-ab", "96k", destination)
	os.Remove(destinationTmp)
	if err != nil {
		return "", err
	}
	if !utils.FileExists(destination) {
		return "", fmt.Errorf(destination + " does not exists")
	}

	return destination, nil
}
