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

	deleteCacheLock sync.RWMutex
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
	youtubeSong = loadedSong.(*YoutubeSong)
	youtubeSong.setLastTimeFetched()

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
	}

	return youtubeSong.getEncryptedId(youtubeDB.randomKey), nil
}

func (youtubeDB *YoutubeDB) GetYoutubeSearch(searchQuery string) ([]YoutubeSearchResult, error) {
	if utils.StringIsEmpty(searchQuery) {
		return nil, utils.Error("Search query is empty!")
	}

	youtubeSearch := newYoutubeSearch(searchQuery)
	loadedSearch, loaded := youtubeDB.searches.LoadOrStore(youtubeSearch.query, youtubeSearch)
	youtubeSearch = loadedSearch.(*YoutubeSearch)
	youtubeSearch.setLastTimeFetched()

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
	}

	return results, err
}

func (youtubeDB *YoutubeDB) GetYoutubeInfo(id string) (YoutubeSearchResult, error) {
	return getYoutubeVideoInfo(id, youtubeDB.ytKey)
}

type rankingInterface interface {
	GetUniqueId() string
	GetTime() time.Time
}

type rankingTree struct {
	start *node
	size  int

	lock sync.RWMutex
}

func (tree *rankingTree) insert(rankingItem rankingInterface) {
	tree.lock.Lock()
	defer tree.lock.Unlock()

	tree.size++
	if tree.start == nil {
		tree.start = &node{rankingItem: rankingItem}
		return
	}
	tree.start.insert(rankingItem)
}

func (tree *rankingTree) delete(rankingItem rankingInterface) bool {
	tree.lock.Lock()
	defer tree.lock.Unlock()

	if tree.start == nil {
		return false
	}
	if tree.start.rankingItem.GetUniqueId() == rankingItem.GetUniqueId() {
		tree.size--
		tree.start = createReplaceNode(tree.start)
		return true
	}
	if tree.start.delete(rankingItem) {
		tree.size--
		return true
	}
	return false
}

func (tree *rankingTree) getLowest() rankingInterface {
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	if tree.start == nil {
		return nil
	}
	return tree.start.getLowest()
}

func (tree *rankingTree) getSize() int {
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	return tree.size
}

type node struct {
	rankingItem rankingInterface
	left, right *node
}

func (nodeLeaf *node) insert(rankingItem rankingInterface) {
	if rankingItem.GetTime().Before(nodeLeaf.rankingItem.GetTime()) {
		if nodeLeaf.left == nil {
			nodeLeaf.left = &node{rankingItem: rankingItem}
		} else {
			nodeLeaf.left.insert(rankingItem)
		}
	} else {
		if nodeLeaf.right == nil {
			nodeLeaf.right = &node{rankingItem: rankingItem}
		} else {
			nodeLeaf.right.insert(rankingItem)
		}
	}
}

func (nodeLeaf *node) delete(rankingItem rankingInterface) bool {
	if nodeLeaf.left != nil &&
		nodeLeaf.left.rankingItem.GetUniqueId() == rankingItem.GetUniqueId() {
		nodeLeaf.left = createReplaceNode(nodeLeaf.left)
		return true
	} else if nodeLeaf.right != nil &&
		nodeLeaf.right.rankingItem.GetUniqueId() == rankingItem.GetUniqueId() {
		nodeLeaf.right = createReplaceNode(nodeLeaf.right)
		return true
	}

	if rankingItem.GetTime().Before(nodeLeaf.rankingItem.GetTime()) {
		if nodeLeaf.left != nil {
			return nodeLeaf.left.delete(rankingItem)
		}
	} else if nodeLeaf.right != nil {
		return nodeLeaf.right.delete(rankingItem)
	}

	return false
}

func (nodeLeaf *node) getLowest() rankingInterface {
	if nodeLeaf.left == nil {
		return nodeLeaf.rankingItem
	}
	return nodeLeaf.left.getLowest()
}

func createReplaceNode(replacedNode *node) *node {
	newNode := replacedNode.right
	if newNode == nil {
		return replacedNode.left
	}
	if replacedNode.left == nil {
		return newNode
	}

	if newNode.left == nil {
		newNode.left = replacedNode.left
		return newNode
	}
	lastLeftNode := newNode.left
	for lastLeftNode.left != nil {
		lastLeftNode = lastLeftNode.left
	}
	lastLeftNode.left = replacedNode.left
	return newNode
}
