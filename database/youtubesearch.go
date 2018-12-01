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

var searchApiRegex = regexp.MustCompile("\"videoId\":\\s+\"([a-z_A-Z0-9\\-]{11})\"")

type YoutubeSearch struct {
	query   string
	results []YoutubeSearchResult

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

func (youtubeSearch *YoutubeSearch) search(youtubeDB *youtubeDBImpl) ([]YoutubeSearchResult, error) {
	youtubeSearch.rwLock.Lock()
	defer youtubeSearch.rwLock.Unlock()

	results, err := youtubeSearch.getSearchFromWebsite(youtubeDB)
	if err != nil && !utils.StringIsEmpty(youtubeDB.ytKey) {
		results, err = youtubeSearch.getSearchFromApi(youtubeDB)
	}
	if err != nil {
		results, err = youtubeSearch.getSearchFromYoutubeDL(youtubeDB)
	}
	if err != nil {
		return nil, err
	}
	youtubeSearch.results = results
	return results, err
}

func (youtubeSearch *YoutubeSearch) getSearchFromWebsite(youtubeDB *youtubeDBImpl) ([]YoutubeSearchResult, error) {
	infos, err := ytdl.GetVideosFromSearch(youtubeSearch.query)
	if err != nil {
		return nil, err
	}

	results := make([]YoutubeSearchResult, len(infos))
	for i, info := range infos {
		if utils.StringIsEmpty(info.Title) || info.Duration == 0 {
			result, err := youtubeDB.GetYoutubeInfo(info.ID)
			if err != nil {
				continue
			}
			results[i] = result
		} else {
			seconds := int(info.Duration.Seconds()) % 60
			minutes := int(info.Duration.Minutes())

			results[i] = YoutubeSearchResult{info.Title, info.ID,
				info.GetThumbnailURL(ytdl.ThumbnailQualityMedium).String(),
				utils.FormatMinutesSeconds(minutes, seconds)}

			if youtubeDB.idRanking.getSize() < 1000 {
				youtubeId := newYoutubeId(info.ID)
				youtubeId.result = results[i]
				youtubeDB.ids.LoadOrStore(info.ID, youtubeId)
			}
		}
	}

	return results, nil
}

func (youtubeSearch *YoutubeSearch) getSearchFromApi(youtubeDB *youtubeDBImpl) ([]YoutubeSearchResult, error) {
	searchUrl := "https://www.googleapis.com/youtube/v3/search?"
	query := url.Values{}
	query.Set("q", youtubeSearch.query)
	query.Set("type", "video")
	query.Set("maxResults", "10")
	query.Set("part", "snippet")
	query.Set("key", youtubeDB.ytKey)

	res, err := http.Get(searchUrl + query.Encode())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("couldn't get website")
	}

	ids := make([]string, 0)
	reader := bufio.NewReader(res.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		matches := searchApiRegex.FindAllStringSubmatch(line, 1)
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

	results := make([]YoutubeSearchResult, 0)
	for _, id := range ids {
		result, err := youtubeDB.GetYoutubeInfo(id)
		if err == nil {
			results = append(results, result)
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results found")
	}
	return results, nil
}

func (youtubeSearch *YoutubeSearch) getSearchFromYoutubeDL(youtubeDB *youtubeDBImpl) ([]YoutubeSearchResult, error) {
	cmd := exec.Command(youtubeDB.youtubeDL, "-e", "--get-id", "--get-thumbnail", "--get-duration",
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

	youtubeResults := make([]YoutubeSearchResult, 0)
	for _, result := range results {
		if youtubeResult, err := youtubeDB.GetYoutubeInfo(result); err == nil {
			youtubeResults = append(youtubeResults, youtubeResult)
		}
	}

	return youtubeResults, nil
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
		info.GetThumbnailURL(ytdl.ThumbnailQualityMedium).String(),
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

	if len(response.Items) == 0 {
		return YoutubeSearchResult{}, fmt.Errorf("no results")
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

func (youtubeSearch *YoutubeSearch) getResults() []YoutubeSearchResult {
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
