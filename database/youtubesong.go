package database

import (
	"time"
	"sync"
	"io/ioutil"
	"github.com/rylio/ytdl"
	"os"

	"../utils"
	"crypto/aes"
)

type YoutubeSong struct {
	id string

	lastFetched     time.Time
	lastFetchedLock *sync.RWMutex

	encryptedId     string
	encryptedIdLock sync.Mutex

	downloadWait sync.WaitGroup
	rwLock       sync.RWMutex
}

func newYoutubeSong(id string) *YoutubeSong {
	return &YoutubeSong{
		id:              id,
		lastFetched:     time.Now(),
		lastFetchedLock: &sync.RWMutex{},
	}
}

func (youtubeSong *YoutubeSong) read() ([]byte, error) {
	youtubeSong.rwLock.RLock()
	defer youtubeSong.rwLock.RUnlock()
	return ioutil.ReadFile(youtubeSong.getFilePath())
}

func (youtubeSong *YoutubeSong) download(info *ytdl.VideoInfo) error {
	youtubeSong.rwLock.Lock()
	defer youtubeSong.rwLock.Unlock()

	formats := info.Formats.Worst(ytdl.FormatAudioEncodingKey)
	var downloadFormat ytdl.Format
	for _, format := range formats {
		if format.AudioEncoding == "opus" {
			downloadFormat = format
			break
		}
	}

	file, err := os.Create(youtubeSong.getFilePath())
	if err != nil {
		panic(err)
	}
	defer file.Close()
	return info.Download(downloadFormat, file)
}

func (youtubeSong *YoutubeSong) delete() error {
	youtubeSong.rwLock.Lock()
	defer youtubeSong.rwLock.Unlock()
	return os.Remove(youtubeSong.getFilePath())
}

func (youtubeSong *YoutubeSong) setLastTimeFetched() {
	youtubeSong.lastFetchedLock.Lock()
	defer youtubeSong.lastFetchedLock.Unlock()
	youtubeSong.lastFetched = time.Now()
}

func (youtubeSong *YoutubeSong) getFilePath() string {
	return utils.YOUTUBE_DIR + "/" + youtubeSong.id + ".webm"
}

func (youtubeSong *YoutubeSong) getEncryptedId(key []byte) string {
	youtubeSong.encryptedIdLock.Lock()
	defer youtubeSong.encryptedIdLock.Unlock()
	if utils.StringIsEmpty(youtubeSong.encryptedId) {
		id := youtubeSong.id
		for i := len(id); i < aes.BlockSize; i++ {
			id += " "
		}
		youtubeSong.encryptedId = utils.Encrypt(key, id)
	}
	return youtubeSong.encryptedId
}

func (youtubeSong YoutubeSong) GetUniqueId() string {
	return youtubeSong.id
}

func (youtubeSong YoutubeSong) GetTime() time.Time {
	youtubeSong.lastFetchedLock.RLock()
	defer youtubeSong.lastFetchedLock.RUnlock()
	return youtubeSong.lastFetched
}
