package database

import (
	"time"
	"sync"
	"../utils"
)

type YoutubeId struct {
	id     string
	result YoutubeSearchResult

	lastFetched     time.Time
	lastFetchedLock *sync.RWMutex

	rwLock sync.RWMutex
}

func newYoutubeId(id string) *YoutubeId {
	return &YoutubeId{
		id:              id,
		lastFetched:     time.Now(),
		lastFetchedLock: &sync.RWMutex{},
	}
}

func (youtubeId *YoutubeId) fetchId(youtubeDB *YoutubeDB) (YoutubeSearchResult, error) {
	youtubeId.rwLock.Lock()
	defer youtubeId.rwLock.Unlock()

	result, err := youtubeDB.getYoutubeVideoInfoFromYtdl(youtubeId.id)
	if err != nil && !utils.StringIsEmpty(youtubeDB.ytKey) {
		result, err = youtubeDB.getYoutubeVideoInfoFromApi(youtubeId.id)
	}
	if err != nil {
		return YoutubeSearchResult{}, err
	}
	youtubeId.result = result
	return result, err
}

func (youtubeId *YoutubeId) getResult() YoutubeSearchResult {
	youtubeId.rwLock.RLock()
	defer youtubeId.rwLock.RUnlock()
	return youtubeId.result
}

func (youtubeId *YoutubeId) setLastTimeFetched() {
	youtubeId.lastFetchedLock.Lock()
	defer youtubeId.lastFetchedLock.Unlock()
	youtubeId.lastFetched = time.Now()
}

func (youtubeId YoutubeId) GetUniqueId() string {
	return youtubeId.id
}

func (youtubeId YoutubeId) GetTime() time.Time {
	youtubeId.lastFetchedLock.RLock()
	defer youtubeId.lastFetchedLock.RUnlock()
	return youtubeId.lastFetched
}
