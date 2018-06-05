package database

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/Grarak/GoYTFetcher/logger"
	"github.com/Grarak/GoYTFetcher/utils"
	"github.com/Grarak/GoYTFetcher/ytdl"
)

var searchWebSiteRegex = regexp.MustCompile("href=\"/watch\\?v=([a-z_A-Z0-9\\-]{11})\"")
var searchApiRegex = regexp.MustCompile("\"videoId\":\\s+\"([a-z_A-Z0-9\\-]{11})\"")

type YoutubeSearch struct {
	query   string
	results []string

	count int

	valuesLock sync.RWMutex
	rwLock     sync.RWMutex
}

type querySort []string

func (query querySort) Len() int {
	return len(query)
}

func (query querySort) Less(i, j int) bool {
	return query[i] < query[j]
}

func (query querySort) Swap(i, j int) {
	query[i], query[j] = query[j], query[i]
}

func newYoutubeSearch(searchQuery string) *YoutubeSearch {
	searchQuery = strings.ToLower(searchQuery)
	searchQuery = regexp.MustCompile("\\s+").ReplaceAllString(searchQuery, " ")
	words := querySort(strings.Split(searchQuery, " "))
	sort.Sort(words)
	searchQuery = strings.Join(words, " ")

	return &YoutubeSearch{query: searchQuery, count: 1}
}

type YoutubeSearchResult struct {
	Title     string `json:"title"`
	Id        string `json:"id"`
	Thumbnail string `json:"thumbnail"`
	Duration  string `json:"duration"`
}

func (youtubeSearch *YoutubeSearch) search(youtubeDB *youtubeDBImpl) ([]string, error) {
	youtubeSearch.rwLock.Lock()
	defer youtubeSearch.rwLock.Unlock()

	results, err := youtubeSearch.getSearchFromWebsite(youtubeDB)
	if err != nil && !utils.StringIsEmpty(youtubeDB.ytKey) {
		results, err = youtubeSearch.getSearchFromApi(youtubeDB)
	}
	if err != nil {
		results, err = youtubeSearch.getSearchFromYoutubeDL(youtubeDB.youtubeDL)
	}
	if err != nil {
		return nil, err
	}
	youtubeSearch.results = results
	return results, err
}

func (youtubeSearch *YoutubeSearch) getSearchFromWebsite(youtubeDB *youtubeDBImpl) ([]string, error) {
	searchUrl := "https://www.youtube.com/results?"
	query := url.Values{}
	query.Set("search_query", youtubeSearch.query)

	return parseYoutubeSearchFromURL(searchUrl+query.Encode(), searchWebSiteRegex)
}

func (youtubeSearch *YoutubeSearch) getSearchFromApi(youtubeDB *youtubeDBImpl) ([]string, error) {
	searchUrl := "https://www.googleapis.com/youtube/v3/search?"
	query := url.Values{}
	query.Set("q", youtubeSearch.query)
	query.Set("type", "video")
	query.Set("maxResults", "10")
	query.Set("part", "snippet")
	query.Set("key", youtubeDB.ytKey)

	return parseYoutubeSearchFromURL(searchUrl+query.Encode(), searchApiRegex)
}

func (youtubeSearch *YoutubeSearch) getSearchFromYoutubeDL(youtubeDL string) ([]string, error) {
	cmd := exec.Command(youtubeDL, "-e", "--get-id", "--get-thumbnail", "--get-duration",
		fmt.Sprintf("ytsearch10:%s", youtubeSearch.query))

	reader, err := cmd.StdoutPipe()
	defer reader.Close()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	results := make([]string, 0)
	bufReader := bufio.NewReader(reader)
	for i := 0; ; i++ {
		line, err := bufReader.ReadString('\n')
		if err != nil {
			break
		}
		results = append(results, line)
	}
	return results, nil
}

func parseYoutubeSearchFromURL(searchUrl string, matcher *regexp.Regexp) ([]string, error) {
	res, err := http.Get(searchUrl)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failure")
	}

	ids := make([]string, 0)
	reader := bufio.NewReader(res.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		matches := matcher.FindAllStringSubmatch(line, 1)
		if len(matches) > 0 && len(matches[0]) > 1 {
			id := matches[0][1]
			if !utils.StringArrayContains(ids, id) {
				ids = append(ids, id)
				if len(ids) >= 10 {
					break
				}
			}
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no ids found")
	}
	return ids, nil
}

func (youtubeDB *youtubeDBImpl) getYoutubeVideoInfoFromYtdl(id string) (YoutubeSearchResult, error) {
	info, err := ytdl.GetVideoInfoFromID(id)
	if err != nil {
		logger.E(fmt.Sprintf("Couldn't get %s, %v", id, err))
		return YoutubeSearchResult{}, err
	}

	seconds := int(info.Duration.Seconds()) % 60
	minutes := int(info.Duration.Minutes())
	return YoutubeSearchResult{info.Title, id,
		info.GetThumbnailURL(ytdl.ThumbnailQualityDefault).String(),
		utils.FormatMinutesSeconds(minutes, seconds)}, nil
}

func (youtubeDB *youtubeDBImpl) getYoutubeVideoInfoFromApi(id string) (YoutubeSearchResult, error) {
	infoUrl := "https://www.googleapis.com/youtube/v3/videos?"
	query := url.Values{}
	query.Set("id", id)
	query.Set("part", "snippet,contentDetails")
	query.Set("key", youtubeDB.ytKey)

	response, err := getYoutubeApiResponseItems(infoUrl + query.Encode())
	if err != nil {
		return YoutubeSearchResult{}, err
	}

	item := response.Items[0]
	return YoutubeSearchResult{item.Snippet.Title, id,
		item.Snippet.Thumbnails.Default.Url,
		utils.FormatMinutesSeconds(
			parseYoutubeApiTime(item.ContentDetails.Duration))}, nil
}

func getYoutubeCharts(apiKey string) ([]YoutubeSearchResult, error) {
	categoriesUrl := "https://www.googleapis.com/youtube/v3/videoCategories?"
	query := url.Values{}
	query.Set("part", "snippet")
	query.Set("regionCode", "US")
	query.Set("key", apiKey)

	response, err := getYoutubeApiResponseItems(categoriesUrl + query.Encode())
	if err != nil {
		return nil, err
	}

	var musicCategoryId string
	for _, item := range response.Items {
		if item.Snippet.Title == "Music" {
			musicCategoryId = item.Id
			break
		}
	}

	if utils.StringIsEmpty(musicCategoryId) {
		return nil, fmt.Errorf("couldn't retrieve category id")
	}

	infoUrl := "https://www.googleapis.com/youtube/v3/videos?"
	query = url.Values{}
	query.Set("chart", "mostPopular")
	query.Set("part", "snippet,contentDetails")
	query.Set("maxResults", "30")
	query.Set("regionCode", "US")
	query.Set("key", apiKey)
	query.Set("videoCategoryId", musicCategoryId)

	response, err = getYoutubeApiResponseItems(infoUrl + query.Encode())
	if err != nil {
		return nil, err
	}

	var results []YoutubeSearchResult
	for _, item := range response.Items {
		result := YoutubeSearchResult{item.Snippet.Title, item.Id,
			item.Snippet.Thumbnails.Default.Url,
			utils.FormatMinutesSeconds(
				parseYoutubeApiTime(item.ContentDetails.Duration))}
		results = append(results, result)
	}
	return results, nil
}

func (youtubeSearch *YoutubeSearch) getResults() []string {
	youtubeSearch.rwLock.RLock()
	defer youtubeSearch.rwLock.RUnlock()
	return youtubeSearch.results
}

func (youtubeSearch *YoutubeSearch) increaseCount() {
	youtubeSearch.valuesLock.Lock()
	defer youtubeSearch.valuesLock.Unlock()
	youtubeSearch.count++
}

func (youtubeSearch YoutubeSearch) GetUniqueId() string {
	return youtubeSearch.query
}

func (youtubeSearch YoutubeSearch) GetCount() int {
	return youtubeSearch.count
}

func parseYoutubeApiTime(duration string) (int, int) {
	hours := 0
	minutes := 0
	seconds := 0

	var numbers []rune
	for _, c := range duration {
		if unicode.IsDigit(c) {
			numbers = append(numbers, c)
		}
		num, err := strconv.Atoi(string(numbers))
		if err != nil {
			num = 0
		}
		switch c {
		case 'H':
			hours = num
			numbers = numbers[:0]
			break
		case 'M':
			minutes = num
			numbers = numbers[:0]
			break
		case 'S':
			seconds = num
			numbers = numbers[:0]
			break
		}
	}
	minutes += hours * 60
	return minutes, seconds
}
