package database

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/Grarak/GoYTFetcher/utils"
)

type YoutubeChartThumbnailDetails struct {
	Url string `json:"url"`
}

type YoutubeChartThumbnail struct {
	Details []YoutubeChartThumbnailDetails `json:"thumbnails"`
}

type YoutubeChartVideoView struct {
	Id        string                `json:"id"`
	Title     string                `json:"title"`
	Thumbnail YoutubeChartThumbnail `json:"thumbnail"`
	Duration  int                   `json:"videoDuration"`
}

type YoutubeChartVideo struct {
	ListType   string                  `json:"listType"`
	VideoViews []YoutubeChartVideoView `json:"videoViews"`
}

type YoutubeChartMusicAnalyticsSectionRendererContent struct {
	Videos []YoutubeChartVideo `json:"videos"`
}

type YoutubeChartMusicAnalyticsSectionRenderer struct {
	Content YoutubeChartMusicAnalyticsSectionRendererContent `json:"content"`
}

type YoutubeChartSectionListRendererContent struct {
	MusicAnalyticsSectionRenderer YoutubeChartMusicAnalyticsSectionRenderer `json:"musicAnalyticsSectionRenderer"`
}

type YoutubeChartSectionListRenderer struct {
	Contents []YoutubeChartSectionListRendererContent `json:"contents"`
}

type YoutubeChartContents struct {
	SectionListRenderer YoutubeChartSectionListRenderer `json:"sectionListRenderer"`
}

type YoutubeChart struct {
	Contents YoutubeChartContents `json:"contents"`
}

func getYoutubeChartsFromApi(apiKey string) ([]YoutubeSearchResult, error) {
	chartsUrl := "https://charts.youtube.com/youtubei/v1/browse?"
	query := url.Values{}
	query.Add("alt", "json")
	query.Add("maxResults", "30")
	query.Add("key", apiKey)

	payload := `{
					"context": {
						"client": {
							"clientName": "WEB_MUSIC_ANALYTICS",
							"clientVersion": "0.2",
      						"hl": "en",
      						"gl": "US",
      						"experimentIds": null,
      						"theme": "MUSIC"
    					},
    					"capabilities": {},
    					"request": {
							"internalExperimentFlags": []
    					}
  					},
  					"query": "chart_params_type=WEEK&perspective=CHART&flags=viral_video_chart&selected_chart=TRACKS&chart_params_id=weekly%3A0%3A0%3Aus",
  					"browseId": "FEmusic_analytics"
				}`

	req, err := http.NewRequest("POST", chartsUrl+query.Encode(),
		bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return nil, err
	}
	defer req.Body.Close()

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Referer", "https://charts.youtube.com/charts/TrendingVideos/us")

	res, err := http.DefaultClient.Do(req)
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var chart YoutubeChart
	err = json.Unmarshal(b, &chart)
	if err != nil {
		return nil, err
	}

	videoTypes := chart.Contents.SectionListRenderer.Contents[0].MusicAnalyticsSectionRenderer.Content.Videos
	var videoChart *YoutubeChartVideo
	for _, videoType := range videoTypes {
		if videoType.ListType == "TRENDING_CHART" {
			videoChart = &videoType
		}
	}

	if videoChart == nil {
		videoChart = &videoTypes[0]
	}

	results := make([]YoutubeSearchResult, 0)
	for _, video := range videoChart.VideoViews {
		minutes := video.Duration / 60
		seconds := video.Duration % 60

		results = append(results, YoutubeSearchResult{video.Title, video.Id,
			video.Thumbnail.Details[1].Url,
			utils.FormatMinutesSeconds(minutes, seconds)})
	}
	return results, nil
}
