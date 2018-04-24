package database

import (
	"sync"
	"../utils"
)

type YoutubeId struct {
	id     string
	result YoutubeSearchResult

	count     int
	countLock sync.RWMutex

	rwLock sync.RWMutex
}

func newYoutubeId(id string) *YoutubeId {
	return &YoutubeId{id: id, count: 1}
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

func (youtubeId *YoutubeId) increaseCount() {
	youtubeId.countLock.Lock()
	defer youtubeId.countLock.Unlock()
	youtubeId.count++
}

func (youtubeId YoutubeId) GetUniqueId() string {
	return youtubeId.id
}

func (youtubeId YoutubeId) GetCount() int {
	youtubeId.countLock.RLock()
	defer youtubeId.countLock.RUnlock()
	return youtubeId.count
}
