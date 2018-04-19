package database

import (
	"sync"
	"time"
	"io/ioutil"

	"../utils"
	"strings"
	"../ytdl"
	"encoding/json"
	"os/exec"
	"fmt"
	"strconv"
)

type Youtube struct {
	ApiKey      string `json:"apikey"`
	SearchQuery string `json:"searchquery"`
	Id          string `json:"id"`
}

func NewYoutube(data []byte) (Youtube, error) {
	var youtube Youtube
	err := json.Unmarshal(data, &youtube)
	return youtube, err
}

type YoutubeDB struct {
	Host string

	randomKey []byte

	ytKey     string
	ytdl      ytdl.Ytdl
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

func newYoutubeDB() (*YoutubeDB, error) {
	youtubeDL, err := exec.LookPath(utils.YOUTUBE_DL)
	if err != nil {
		return nil, err
	}

	youtubeDB := &YoutubeDB{
		ytdl:            ytdl.NewYtdl(),
		youtubeDL:       youtubeDL,
		songsRanking:    new(rankingTree),
		searchesRanking: new(rankingTree),
		idRanking:       new(rankingTree),
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
			youtubeDB.songsRanking.insert(*youtubeSong)
			youtubeDB.songs.Store(id, youtubeSong)
		}
	}

	return youtubeDB, nil
}

func (youtubeDB *YoutubeDB) GetYoutubeSong(id string) ([]byte, error) {
	decryptedId, err := utils.Decrypt(youtubeDB.randomKey, id)
	if err != nil {
		return nil, err
	}

	loadedSong, ok := youtubeDB.songs.Load(decryptedId[:11])
	if !ok {
		return nil, fmt.Errorf("%s does not exist", id)
	}
	youtubeSong := loadedSong.(*YoutubeSong)
	return youtubeSong.read()
}

func (youtubeDB *YoutubeDB) FetchYoutubeSong(id string) (string, error) {
	info, err := youtubeDB.GetYoutubeInfo(id)
	if err != nil {
		return "", nil
	}
	minutes, err := strconv.Atoi(info.Duration[:strings.Index(info.Duration, ":")])
	if err != nil {
		return "", nil
	}
	if minutes > 20 {
		return "", fmt.Errorf("video too long")
	}

	youtubeSong := newYoutubeSong(id)
	loadedSong, loaded := youtubeDB.songs.LoadOrStore(id, youtubeSong)
	if loaded {
		youtubeSong = loadedSong.(*YoutubeSong)
		youtubeSong.setLastTimeFetched()
	}

	var url string
	if youtubeSong.isDownloaded() {
		url = youtubeSong.getEncryptedId(youtubeDB.randomKey)
	} else if youtubeSong.isDownloading() {
		url = youtubeSong.getGoogleUrl()
	} else {
		youtubeSong.googleUrlLock.Lock()
		go func() {
			youtubeDB.deleteCacheLock.RLock()
			youtubeSong.download(youtubeDB)
			youtubeDB.deleteCacheLock.RUnlock()
		}()
		url = youtubeSong.getGoogleUrl()
	}

	if utils.StringIsEmpty(url) {
		youtubeDB.songs.Delete(id)
		return youtubeDB.FetchYoutubeSong(id)
	}

	if err == nil {
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
	} else {
		youtubeDB.songs.Delete(youtubeSong)
	}
	return url, nil
}

func (youtubeDB *YoutubeDB) GetYoutubeSearch(searchQuery string) ([]YoutubeSearchResult, error) {
	if utils.StringIsEmpty(searchQuery) {
		return nil, fmt.Errorf("search query is empty")
	}

	youtubeSearch := newYoutubeSearch(searchQuery)
	loadedSearch, loaded := youtubeDB.searches.LoadOrStore(youtubeSearch.query, youtubeSearch)
	if loaded {
		youtubeSearch = loadedSearch.(*YoutubeSearch)
		youtubeSearch.setLastTimeFetched()
	}

	var results []YoutubeSearchResult
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
		youtubeDB.searches.Delete(youtubeSearch)
	}
	return results, err
}

func (youtubeDB *YoutubeDB) GetYoutubeInfo(id string) (YoutubeSearchResult, error) {
	if utils.StringIsEmpty(id) {
		return YoutubeSearchResult{}, fmt.Errorf("id is empty")
	}

	youtubeId := newYoutubeId(id)
	loadedId, loaded := youtubeDB.ids.LoadOrStore(youtubeId.id, youtubeId)
	if loaded {
		youtubeId = loadedId.(*YoutubeId)
		youtubeId.setLastTimeFetched()
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
		youtubeDB.ids.Delete(youtubeId)
	}
	return result, err
}

func (youtubeDB *YoutubeDB) GetYoutubeCharts() ([]YoutubeSearchResult, error) {
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
