package database

import (
	"sync"
	"time"
	"io/ioutil"

	"../utils"
	"strings"
	"github.com/rylio/ytdl"
	"encoding/json"
	"os/exec"
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

func newYoutubeDB() (*YoutubeDB, error) {
	youtubeDL, err := exec.LookPath(utils.YOUTUBE_DL)
	if err != nil {
		return nil, err
	}

	youtubeDB := &YoutubeDB{
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
		return nil, utils.Error(id + " does not exist")
	}
	youtubeSong := loadedSong.(*YoutubeSong)
	return youtubeSong.read()
}

func (youtubeDB *YoutubeDB) FetchYoutubeSong(id string) (string, error) {
	videoInfo, err := ytdl.GetVideoInfoFromID(id)
	if err != nil {
		return "", err
	}
	if videoInfo.Duration.Minutes() > 20 {
		return "", utils.Error("Video too long!")
	}

	youtubeSong := newYoutubeSong(videoInfo.ID)
	loadedSong, loaded := youtubeDB.songs.LoadOrStore(youtubeSong.id, youtubeSong)
	if loaded {
		youtubeSong = loadedSong.(*YoutubeSong)
		youtubeSong.setLastTimeFetched()
	}

	if loaded {
		youtubeSong.downloadWait.Wait()
	} else {
		youtubeSong.downloadWait.Add(1)
		defer youtubeSong.downloadWait.Done()
		youtubeDB.deleteCacheLock.RLock()
		err = youtubeSong.download(videoInfo)
		youtubeDB.deleteCacheLock.RUnlock()
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
	return youtubeSong.getEncryptedId(youtubeDB.randomKey), nil
}

func (youtubeDB *YoutubeDB) GetYoutubeSearch(searchQuery string) ([]YoutubeSearchResult, error) {
	if utils.StringIsEmpty(searchQuery) {
		return nil, utils.Error("Search query is empty!")
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
		return YoutubeSearchResult{}, utils.Error("Id is empty!")
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
