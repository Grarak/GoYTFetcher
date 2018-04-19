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

	"../logger"
	"encoding/xml"
	"io"
)

const youtubeBaseURL = "https://www.youtube.com/watch"

type Ytdl struct {
	jsonRegex, sigRegex, sigSubRegex *regexp.Regexp
}

func NewYtdl() Ytdl {
	return Ytdl{
		regexp.MustCompile("ytplayer.config = (.*?);ytplayer.load"),
		regexp.MustCompile("\\/s\\/([a-fA-F0-9\\.]+)"),
		regexp.MustCompile("([a-fA-F0-9\\.]+)"),
	}
}

// VideoInfo contains the info a youtube video
type VideoInfo struct {
	// The video ID
	ID string `json:"id"`
	// The video title
	Title string `json:"title"`
	// Formats the video is available in
	Formats FormatList `json:"formats"`
	// Duration of the video
	Duration time.Duration

	htmlPlayerFile string
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

	inf := jsonConfig["args"].(map[string]interface{})
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
				logger.I(fmt.Sprintf("No metadata found for itag: %d, skipping...", itag))
			}
		} else {
			logger.I(fmt.Sprintf("Unable to format string %s", err.Error()))
		}
	}

	if dashManifestURL, ok := inf["dashmpd"].(string); ok {
		tokens, err := getSigTokens(info.htmlPlayerFile)
		if err != nil {
			return nil, fmt.Errorf("unable to extract signature tokens: %s", err.Error())
		}
		dashManifestURL = ytdl.sigRegex.ReplaceAllStringFunc(dashManifestURL, func(str string) string {
			return "/signature/" + decipherTokens(tokens, ytdl.sigSubRegex.FindString(str))
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

func (info *VideoInfo) GetDownloadURL(format Format) (*url.URL, error) {
	return getDownloadURL(format, info.htmlPlayerFile)
}

func (info *VideoInfo) GetThumbnailURL(quality ThumbnailQuality) *url.URL {
	u, _ := url.Parse(fmt.Sprintf("http://img.youtube.com/vi/%s/%s.jpg",
		info.ID, quality))
	return u
}

func (info *VideoInfo) Download(format Format, dest io.Writer) error {
	u, err := info.GetDownloadURL(format)
	if err != nil {
		return err
	}
	resp, err := http.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}
	_, err = io.Copy(dest, resp.Body)
	return err
}
