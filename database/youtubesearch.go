package database

import (
	"time"
	"sync"
	"os/exec"
	"regexp"
	"strings"
	"sort"
	"bufio"
	"fmt"
	"net/http"

	"../utils"
	"net/url"
	"github.com/rylio/ytdl"
	"io/ioutil"
	"encoding/json"
	"unicode"
	"strconv"
)

type YoutubeSearch struct {
	query   string
	results []YoutubeSearchResult

	lastFetched     time.Time
	lastFetchedLock *sync.RWMutex

	rwLock sync.RWMutex
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
	searchQuery = regexp.MustCompile("\\s+").ReplaceAllString(searchQuery, " ")
	words := querySort(strings.Split(searchQuery, " "))
	sort.Sort(words)
	searchQuery = strings.Join(words, " ")

	return &YoutubeSearch{
		query:           searchQuery,
		lastFetched:     time.Now(),
		lastFetchedLock: &sync.RWMutex{},
	}
}

type YoutubeSearchResult struct {
	Title     string `json:"title"`
	Id        string `json:"id"`
	Thumbnail string `json:"thumbnail"`
	Duration  string `json:"duration"`
}

func (youtubeSearch *YoutubeSearch) search(youtubeDB *YoutubeDB) ([]YoutubeSearchResult, error) {
	youtubeSearch.rwLock.Lock()
	defer youtubeSearch.rwLock.Unlock()

	results, err := youtubeSearch.getSearchFromWebsite(youtubeDB.ytKey)
	if err != nil && !utils.StringIsEmpty(youtubeDB.ytKey) {
		results, err = youtubeSearch.getSearchFromApi(youtubeDB.ytKey)
	}
	if err != nil {
		results, err = youtubeSearch.getSearchFromApi(youtubeDB.ytKey)
	}
	youtubeSearch.results = results
	return results, err
}

func (youtubeSearch *YoutubeSearch) getSearchFromWebsite(apiKey string) ([]YoutubeSearchResult, error) {
	searchUrl := "https://www.youtube.com/results?"
	query := url.Values{}
	query.Set("search_query", youtubeSearch.query)

	matcher, err := regexp.Compile("href=\"/watch\\?v=([a-z_A-Z0-9\\-]{11})\"")
	if err != nil {
		return nil, err
	}
	ids, err := parseYoutubeSearchFromURL(searchUrl+query.Encode(), matcher)
	if err != nil {
		return nil, err
	}

	var results []YoutubeSearchResult
	for _, id := range ids {
		result, err := getYoutubeVideoInfo(id, apiKey)
		if err == nil {
			results = append(results, result)
		}
	}
	return results, nil
}

func (youtubeSearch *YoutubeSearch) getSearchFromApi(apiKey string) ([]YoutubeSearchResult, error) {
	searchUrl := "https://www.googleapis.com/youtube/v3/search?"
	query := url.Values{}
	query.Set("q", youtubeSearch.query)
	query.Set("type", "video")
	query.Set("maxResults", "5")
	query.Set("part", "snippet")
	query.Set("key", apiKey)

	matcher, err := regexp.Compile("\"videoId\":\\s+\"([a-z_A-Z0-9\\-]{11})\"")
	if err != nil {
		return nil, err
	}
	ids, err := parseYoutubeSearchFromURL(searchUrl+query.Encode(), matcher)
	if err != nil {
		return nil, err
	}

	var results []YoutubeSearchResult
	for _, id := range ids {
		result, err := getYoutubeVideoInfo(id, apiKey)
		if err == nil {
			results = append(results, result)
		}
	}
	return results, nil
}

func (youtubeSearch *YoutubeSearch) getSearchFromYoutubeDL(youtubeDL string) ([]YoutubeSearchResult, error) {
	cmd := exec.Command(youtubeDL, "-e", "--get-id", "--get-thumbnail", "--get-duration",
		fmt.Sprintf("ytsearch5:%s", youtubeSearch.query))

	reader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	var results []YoutubeSearchResult
	var result YoutubeSearchResult
	bufReader := bufio.NewReader(reader)
	for i := 0; ; i++ {
		line, err := bufReader.ReadString('\n')
		if err != nil {
			break
		}
		switch i {
		case 0:
			result = YoutubeSearchResult{}
			result.Title = line
			break
		case 1:
			result.Id = line
			break
		case 2:
			result.Thumbnail = line

			// check if medium quality exist
			thumbnailUrl := result.Thumbnail
			thumbnailUrl = thumbnailUrl[:strings.LastIndex(thumbnailUrl,
				"/")] + "/default.jpg"

			res, err := http.Get(thumbnailUrl)
			if err != nil || res.StatusCode != http.StatusOK {
				break
			}
			result.Thumbnail = thumbnailUrl
			break
		case 3:
			result.Duration = line
			results = append(results, result)
			i = -1
			break
		}
	}
	reader.Close()

	if len(results) == 0 {
		return nil, utils.Error("No videos found!")
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
		return nil, utils.Error("Failure!")
	}

	var ids [] string
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
				if len(ids) >= 5 {
					break
				}
			}
		}
	}
	if len(ids) == 0 {
		return nil, utils.Error("No ids found!")
	}
	return ids, nil
}

func getYoutubeVideoInfo(id, apiKey string) (YoutubeSearchResult, error) {
	var result YoutubeSearchResult
	var err error = utils.Error("No results!")
	if !utils.StringIsEmpty(apiKey) {
		result, err = getYoutubeVideoInfoFromApi(id, apiKey)
	}
	if err != nil {
		result, err = getYoutubeVideoInfoFromYtdl(id)
	}
	return result, err
}

func getYoutubeVideoInfoFromApi(id, apiKey string) (YoutubeSearchResult, error) {
	infoUrl := "https://www.googleapis.com/youtube/v3/videos?"
	query := url.Values{}
	query.Set("id", id)
	query.Set("part", "snippet,contentDetails")
	query.Set("key", apiKey)

	type Thumbnail struct {
		Url string `json:"url"`
	}

	type Thumbnails struct {
		Default Thumbnail `json:"default"`
	}

	type Snippet struct {
		Title      string     `json:"title"`
		Thumbnails Thumbnails `json:"thumbnails"`
	}

	type ContentDetails struct {
		Duration string `json:"duration"`
	}

	type YoutubeItem struct {
		Snippet        Snippet        `json:"snippet"`
		ContentDetails ContentDetails `json:"contentDetails"`
	}

	type Response struct {
		Items []YoutubeItem `json:"items"`
	}

	res, err := http.Get(infoUrl + query.Encode())
	if err != nil {
		return YoutubeSearchResult{}, err
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return YoutubeSearchResult{}, utils.Error("Failure!")
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return YoutubeSearchResult{}, err
	}

	var response Response
	err = json.Unmarshal(b, &response)
	if err != nil {
		return YoutubeSearchResult{}, err
	}

	hours := 0
	minutes := 0
	seconds := 0

	var numbers []rune
	item := response.Items[0]
	for _, c := range item.ContentDetails.Duration {
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

	return YoutubeSearchResult{item.Snippet.Title, id,
		item.Snippet.Thumbnails.Default.Url,
		utils.FormatMinutesSeconds(minutes, seconds)}, nil
}

func getYoutubeVideoInfoFromYtdl(id string) (YoutubeSearchResult, error) {
	info, err := ytdl.GetVideoInfoFromID(id)
	if err != nil {
		return YoutubeSearchResult{}, err
	}

	seconds := int(info.Duration.Seconds()) % 60
	minutes := int(info.Duration.Minutes())
	return YoutubeSearchResult{info.Title, id,
		info.GetThumbnailURL(ytdl.ThumbnailQualityDefault).String(),
		utils.FormatMinutesSeconds(minutes, seconds)}, nil
}

func (youtubeSearch *YoutubeSearch) getResults() []YoutubeSearchResult {
	youtubeSearch.rwLock.RLock()
	defer youtubeSearch.rwLock.RUnlock()
	return youtubeSearch.results
}

func (youtubeSearch *YoutubeSearch) setLastTimeFetched() {
	youtubeSearch.lastFetchedLock.Lock()
	defer youtubeSearch.lastFetchedLock.Unlock()
	youtubeSearch.lastFetched = time.Now()
}

func (youtubeSearch YoutubeSearch) GetUniqueId() string {
	return youtubeSearch.query
}

func (youtubeSearch YoutubeSearch) GetTime() time.Time {
	youtubeSearch.lastFetchedLock.RLock()
	defer youtubeSearch.lastFetchedLock.RUnlock()
	return youtubeSearch.lastFetched
}
