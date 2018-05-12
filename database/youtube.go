package database

import (
	"io/ioutil"
	"sync"
	"time"

	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Grarak/GoYTFetcher/utils"
)

type Youtube struct {
	ApiKey      string `json:"apikey"`
	SearchQuery string `json:"searchquery"`
	Id          string `json:"id"`
	AddHistory  bool   `json:"addhistory"`
}

func NewYoutube(data []byte) (Youtube, error) {
	var youtube Youtube
	err := json.Unmarshal(data, &youtube)
	return youtube, err
}

type YouTubeDB interface {
	GetYoutubeSong(id string) (*YoutubeSong, error)
	FetchYoutubeSong(id string) (string, string, error)
	GetYoutubeSearch(searchQuery string) ([]YoutubeSearchResult, error)
	GetYoutubeInfo(id string) (YoutubeSearchResult, error)
	GetYoutubeCharts() ([]YoutubeSearchResult, error)
}

type youtubeDBImpl struct {
	Host      string
	randomKey []byte

	ytKey     string
	youtubeDL string

	songsRanking *rankingTree
	songs        sync.Map

	searchesRanking *rankingTree
	searches        sync.Map

	idRanking *rankingTree
	ids       sync.Map

	deleteCacheLock sync.RWMutex

	charts            []YoutubeSearchResult
	chartsLock        sync.RWMutex
	chartsLastFetched time.Time
}

func newYoutubeDB(host string, key []byte, ytKey string) (YouTubeDB, error) {
	youtubeDL, err := exec.LookPath(utils.YOUTUBE_DL)
	if err != nil {
		return nil, err
	}

	youtubeDB := &youtubeDBImpl{
		youtubeDL:       youtubeDL,
		songsRanking:    new(rankingTree),
		searchesRanking: new(rankingTree),
		idRanking:       new(rankingTree),
		Host:            host,
		randomKey:       key,
		ytKey:           ytKey,
	}

	files, err := ioutil.ReadDir(utils.YOUTUBE_DIR)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() {
			id := file.Name()
			id = id[:strings.LastIndex(id, ".")]

			youtubeSong := newYoutubeSong(id)
			youtubeSong.setDownloaded(true)
			youtubeSong.filePath = utils.YOUTUBE_DIR + "/" + file.Name()
			youtubeDB.songsRanking.insert(*youtubeSong)
			youtubeDB.songs.Store(id, youtubeSong)
		}
	}

	return youtubeDB, nil
}

func (youtubeDB *youtubeDBImpl) GetYoutubeSong(id string) (*YoutubeSong, error) {
	decryptedId, err := utils.Decrypt(youtubeDB.randomKey, id)
	if err != nil {
		return nil, err
	}

	loadedSong, ok := youtubeDB.songs.Load(decryptedId[:11])
	if !ok {
		return nil, fmt.Errorf("%s does not exist", id)
	}
	youtubeSong := loadedSong.(*YoutubeSong)
	return youtubeSong, nil
}

func (youtubeDB *youtubeDBImpl) FetchYoutubeSong(id string) (string, string, error) {
	youtubeSong := newYoutubeSong(id)
	loadedSong, loaded := youtubeDB.songs.LoadOrStore(id, youtubeSong)
	if loaded {
		youtubeSong = loadedSong.(*YoutubeSong)
		youtubeSong.increaseCount()
	}

	encryptedId := youtubeSong.getEncryptedId(youtubeDB.randomKey)
	var url string
	if youtubeSong.isDownloaded() {
		url = encryptedId
	} else if youtubeSong.IsDownloading() {
		url = youtubeSong.getGoogleUrl()
	} else if !loaded {
		youtubeSong.googleUrlLock.Lock()
		go func() {
			youtubeDB.deleteCacheLock.RLock()
			defer youtubeDB.deleteCacheLock.RUnlock()
			youtubeSong.download(youtubeDB)
		}()
		url = youtubeSong.getGoogleUrl()
	}

	if utils.StringIsEmpty(url) {
		youtubeDB.songs.Delete(youtubeSong.id)
		return "", "", fmt.Errorf("failed to get url")
	}

	youtubeDB.songsRanking.delete(*youtubeSong)
	youtubeDB.songsRanking.insert(*youtubeSong)
	if youtubeDB.songsRanking.getSize() >= 1000 {
		lowestSong := youtubeDB.songsRanking.getLowest()
		youtubeDB.songsRanking.delete(lowestSong)

		loadedSong, loaded = youtubeDB.songs.Load(lowestSong.GetUniqueId())
		if loaded {
			youtubeSong := loadedSong.(*YoutubeSong)

			youtubeDB.songs.Delete(lowestSong.GetUniqueId())

			youtubeDB.deleteCacheLock.Lock()
			youtubeSong.delete()
			youtubeDB.deleteCacheLock.Unlock()
		}
	}
	return url, encryptedId, nil
}

func (youtubeDB *youtubeDBImpl) GetYoutubeSearch(searchQuery string) ([]YoutubeSearchResult, error) {
	if utils.StringIsEmpty(searchQuery) {
		return nil, fmt.Errorf("search query is empty")
	}

	youtubeSearch := newYoutubeSearch(searchQuery)
	loadedSearch, loaded := youtubeDB.searches.LoadOrStore(youtubeSearch.query, youtubeSearch)
	if loaded {
		youtubeSearch = loadedSearch.(*YoutubeSearch)
		youtubeSearch.increaseCount()
	}

	var results []string
	var err error
	if loaded {
		results = youtubeSearch.getResults()
	} else {
		results, err = youtubeSearch.search(youtubeDB)
	}

	if err == nil {
		youtubeDB.searchesRanking.delete(*youtubeSearch)
		youtubeDB.songsRanking.insert(*youtubeSearch)
		if youtubeDB.songsRanking.getSize() >= 1000 {
			lowestSearch := youtubeDB.songsRanking.getLowest()
			youtubeDB.songsRanking.delete(lowestSearch)
			youtubeDB.songs.Delete(lowestSearch.GetUniqueId())
		}
	} else {
		youtubeDB.searches.Delete(youtubeSearch.query)
	}

	youtubeSearchResults := make([]YoutubeSearchResult, 0)
	for _, id := range results {
		if result, e := youtubeDB.GetYoutubeInfo(id); e == nil {
			youtubeSearchResults = append(youtubeSearchResults, result)
		}
	}
	return youtubeSearchResults, err
}

func (youtubeDB *youtubeDBImpl) GetYoutubeInfo(id string) (YoutubeSearchResult, error) {
	if utils.StringIsEmpty(id) {
		return YoutubeSearchResult{}, fmt.Errorf("id is empty")
	}

	youtubeId := newYoutubeId(id)
	loadedId, loaded := youtubeDB.ids.LoadOrStore(youtubeId.id, youtubeId)
	if loaded {
		youtubeId = loadedId.(*YoutubeId)
		youtubeId.increaseCount()
	}

	var result YoutubeSearchResult
	var err error
	if loaded {
		result = youtubeId.getResult()
	} else {
		result, err = youtubeId.fetchId(youtubeDB)
	}

	if err == nil {
		youtubeDB.idRanking.delete(*youtubeId)
		youtubeDB.idRanking.insert(*youtubeId)
		if youtubeDB.idRanking.getSize() >= 1000 {
			lowestId := youtubeDB.idRanking.getLowest()
			youtubeDB.idRanking.delete(lowestId)
			youtubeDB.ids.Delete(lowestId.GetUniqueId())
		}
	} else {
		youtubeDB.ids.Delete(youtubeId.id)
	}
	return result, err
}

func (youtubeDB *youtubeDBImpl) GetYoutubeCharts() ([]YoutubeSearchResult, error) {
	youtubeDB.chartsLock.RLock()
	if len(youtubeDB.charts) == 0 || youtubeDB.chartsLastFetched.Day() != time.Now().Day() {
		youtubeDB.chartsLock.RUnlock()
		youtubeDB.chartsLock.Lock()
		defer youtubeDB.chartsLock.Unlock()

		charts, err := getYoutubeCharts(youtubeDB.ytKey)
		if err != nil {
			return nil, err
		}
		youtubeDB.chartsLastFetched = time.Now()
		youtubeDB.charts = charts
		return charts, nil
	}

	defer youtubeDB.chartsLock.RUnlock()
	return youtubeDB.charts, nil
}
