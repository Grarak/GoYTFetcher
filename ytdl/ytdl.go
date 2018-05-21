package ytdl

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Grarak/GoYTFetcher/logger"
	"github.com/Grarak/GoYTFetcher/utils"
	"github.com/PuerkitoBio/goquery"
)

const youtubeBaseURL = "https://www.youtube.com/watch"
const youtubeInfoURL = "https://www.youtube.com/get_video_info"

var jsonRegex = regexp.MustCompile("ytplayer.config = (.*?);ytplayer.load")
var sigRegex = regexp.MustCompile("\\/s\\/([a-fA-F0-9\\.]+)")
var sigSubRegex = regexp.MustCompile("([a-fA-F0-9\\.]+)")

// VideoInfo contains the info a youtube video
type VideoInfo struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Duration time.Duration
}

type VideoDownloadInfo struct {
	VideoInfo      *VideoInfo
	Formats        FormatList `json:"formats"`
	htmlPlayerFile string
}

func GetVideoInfoFromID(id string) (*VideoInfo, error) {
	u, _ := url.ParseRequestURI(youtubeInfoURL)
	values := u.Query()
	values.Set("video_id", id)
	u.RawQuery = values.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return getVideoInfoFromHTML(id)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	values, err = url.ParseQuery(string(body))
	if err != nil {
		return getVideoInfoFromHTML(id)
	}
	if status := values.Get("status"); utils.StringIsEmpty(status) || status != "ok" {
		return getVideoInfoFromHTML(id)
	}

	title := values.Get("title")
	length := values.Get("length_seconds")
	if utils.StringIsEmpty(title) || utils.StringIsEmpty(length) {
		return getVideoInfoFromHTML(id)
	}

	duration, err := time.ParseDuration(length + "s")
	if err != nil {
		return getVideoInfoFromHTML(id)
	}
	return &VideoInfo{ID: id, Title: title, Duration: duration}, nil
}

func getVideoInfoFromHTML(id string) (*VideoInfo, error) {
	downloadInfo, err := GetVideoDownloadInfo(id)
	if err != nil {
		return nil, err
	}
	return downloadInfo.VideoInfo, nil
}

func GetVideoDownloadInfo(id string) (*VideoDownloadInfo, error) {
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
	return parseVideoInfoFromHTML(id, body)
}

func parseVideoInfoFromHTML(id string, html []byte) (*VideoDownloadInfo, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, err
	}

	info := &VideoDownloadInfo{VideoInfo: &VideoInfo{}}

	// extract description and title
	info.VideoInfo.Title = strings.TrimSpace(doc.Find("#eow-title").Text())
	info.VideoInfo.ID = id

	// match json in javascript
	matches := jsonRegex.FindSubmatch(html)
	var jsonConfig map[string]interface{}
	if len(matches) > 1 {
		err = json.Unmarshal(matches[1], &jsonConfig)
		if err != nil {
			return nil, err
		}
	}

	inf, ok := jsonConfig["args"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%s: error no args in json", id)
	}
	if status, ok := inf["status"].(string); ok && status == "fail" {
		return nil, fmt.Errorf("%s: error %d:%s", id, inf["errorcode"], inf["reason"])
	}

	if length, ok := inf["length_seconds"].(string); ok {
		if duration, err := strconv.ParseInt(length, 10, 64); err == nil {
			info.VideoInfo.Duration = time.Second * time.Duration(duration)
		} else {
			logger.I(fmt.Sprintf(id+": Unable to parse duration string: %s", length))
		}
	} else {
		logger.E(id + ": Unable to extract duration")
	}

	info.htmlPlayerFile = jsonConfig["assets"].(map[string]interface{})["js"].(string)

	var formatStrings []string
	if fmtStreamMap, ok := inf["url_encoded_fmt_stream_map"].(string); ok {
		formatStrings = append(formatStrings, strings.Split(fmtStreamMap, ",")...)
	}

	if adaptiveFormats, ok := inf["adaptive_fmts"].(string); ok {
		formatStrings = append(formatStrings, strings.Split(adaptiveFormats, ",")...)
	}
	var formats FormatList
	for _, v := range formatStrings {
		query, err := url.ParseQuery(v)
		if err == nil {
			itag, _ := strconv.Atoi(query.Get("itag"))
			if format, ok := newFormat(itag); ok {
				if strings.HasPrefix(query.Get("conn"), "rtmp") {
					format.meta["rtmp"] = true
				}
				for k, v := range query {
					if len(v) == 1 {
						format.meta[k] = v[0]
					} else {
						format.meta[k] = v
					}
				}
				formats = append(formats, format)
			} else {
				logger.I(fmt.Sprintf(id+": No metadata found for itag: %d, skipping...", itag))
			}
		} else {
			logger.I(fmt.Sprintf(id+": Unable to format string %s", err.Error()))
		}
	}

	if dashManifestURL, ok := inf["dashmpd"].(string); ok {
		tokens, err := getSigTokens(info.htmlPlayerFile)
		if err != nil {
			return nil, fmt.Errorf("unable to extract signature tokens: %s", err.Error())
		}
		dashManifestURL = sigRegex.ReplaceAllStringFunc(dashManifestURL, func(str string) string {
			return "/signature/" + decipherTokens(tokens, sigSubRegex.FindString(str))
		})
		dashFormats, err := getDashManifest(dashManifestURL)
		if err != nil {
			return nil, fmt.Errorf("unable to extract dash manifest: %s", err.Error())
		}

		for _, dashFormat := range dashFormats {
			added := false
			for j, format := range formats {
				if dashFormat.Itag == format.Itag {
					formats[j] = dashFormat
					added = true
					break
				}
			}
			if !added {
				formats = append(formats, dashFormat)
			}
		}
	}
	info.Formats = formats
	return info, nil
}

type representation struct {
	Itag   int    `xml:"id,attr"`
	Height int    `xml:"height,attr"`
	URL    string `xml:"BaseURL"`
}

func getDashManifest(urlString string) (formats []Format, err error) {
	resp, err := http.Get(urlString)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("invalid status code %d", resp.StatusCode)
	}
	dec := xml.NewDecoder(resp.Body)
	var token xml.Token
	for ; err == nil; token, err = dec.Token() {
		if el, ok := token.(xml.StartElement); ok && el.Name.Local == "Representation" {
			var rep representation
			err = dec.DecodeElement(&rep, &el)
			if err != nil {
				break
			}
			if format, ok := newFormat(rep.Itag); ok {
				format.meta["url"] = rep.URL
				if rep.Height != 0 {
					format.Resolution = strconv.Itoa(rep.Height) + "p"
				} else {
					format.Resolution = ""
				}
				formats = append(formats, format)
			} else {
				logger.I(fmt.Sprintf("No metadata found for itag: %d, skipping...", rep.Itag))
			}
		}
	}
	if err != io.EOF {
		return nil, err
	}
	return formats, nil
}

func (info *VideoDownloadInfo) GetDownloadURL() (*url.URL, error) {
	var downloadFormat Format
	for _, format := range info.Formats.Best(FormatAudioEncodingKey) {
		if format.AudioEncoding == "vorbis" && format.Resolution == "" {
			downloadFormat = format
			break
		}
	}

	if downloadFormat.AudioBitrate == 0 {
		for _, format := range info.Formats.Best(FormatAudioEncodingKey) {
			if format.Resolution == "" {
				downloadFormat = format
				break
			}
		}
	}
	return getDownloadURL(downloadFormat, info.htmlPlayerFile)
}

func (info *VideoDownloadInfo) GetDownloadURLWorst() (*url.URL, error) {
	var downloadFormat Format
	for _, format := range info.Formats.Worst(FormatAudioEncodingKey) {
		if format.AudioEncoding == "opus" && format.Resolution == "" {
			downloadFormat = format
			break
		}
	}

	if downloadFormat.AudioBitrate == 0 {
		for _, format := range info.Formats.Worst(FormatAudioEncodingKey) {
			if format.Resolution == "" {
				downloadFormat = format
				break
			}
		}
	}
	return getDownloadURL(downloadFormat, info.htmlPlayerFile)
}

func (info *VideoInfo) GetThumbnailURL(quality ThumbnailQuality) *url.URL {
	u, _ := url.Parse(fmt.Sprintf("http://img.youtube.com/vi/%s/%s.jpg",
		info.ID, quality))

	resp, err := http.Get(u.String())
	defer resp.Body.Close()
	if err != nil || resp.StatusCode != http.StatusOK {
		u, _ = url.Parse(fmt.Sprintf("https://i.ytimg.com/vi/%s/%s.jpg",
			info.ID, quality))
	}
	return u
}

func (info *VideoInfo) Download(path, youtubeDL string) (string, error) {
	destination := path + "/" + info.ID + ".%(ext)s"

	output, err := utils.ExecuteCmd(youtubeDL, "--extract-audio",
		"--audio-quality", "96k", "--audio-format",
		"vorbis", "--output", destination, info.ID)
	if err != nil {
		return "", err
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "[ffmpeg] Destination:") {
			destination = line[strings.Index(line, path):]
			break
		}
	}

	if !utils.FileExists(destination) {
		return "", fmt.Errorf(destination + " does not exists")
	}
	return destination, nil
}
