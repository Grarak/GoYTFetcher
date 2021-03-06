package database

import (
	"sync"

	"github.com/Grarak/GoYTFetcher/utils"
)

type YoutubeId struct {
	id     string
	result YoutubeSearchResult

	count int

	valuesLock sync.RWMutex
	rwLock     sync.RWMutex
}

func newYoutubeId(id string) *YoutubeId {
	return &YoutubeId{id: id, count: 1}
}

func (youtubeId *YoutubeId) fetchId(youtubeDB *youtubeDBImpl) (YoutubeSearchResult, error) {
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
	youtubeId.valuesLock.Lock()
	defer youtubeId.valuesLock.Unlock()
	youtubeId.count++
}

func (youtubeId YoutubeId) GetUniqueId() string {
	return youtubeId.id
}

func (youtubeId YoutubeId) GetCount() int {
	return youtubeId.count
}
