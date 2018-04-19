package database

import (
	"time"
	"sync"
	"io/ioutil"
	"../ytdl"
	"os"

	"../utils"
	"crypto/aes"
)

type YoutubeSong struct {
	id string

	googleUrl     string
	googleUrlLock sync.RWMutex

	lastFetched     time.Time
	lastFetchedLock *sync.RWMutex

	encryptedId     string
	encryptedIdLock sync.Mutex

	downloaded     bool
	downloadedLock sync.RWMutex

	downloading     bool
	downloadingLock sync.RWMutex

	rwLock sync.RWMutex
}

func newYoutubeSong(id string) *YoutubeSong {
	return &YoutubeSong{
		id:              id,
		lastFetched:     time.Now(),
		lastFetchedLock: &sync.RWMutex{},
	}
}

func (youtubeSong *YoutubeSong) isDownloaded() bool {
	youtubeSong.downloadedLock.RLock()
	defer youtubeSong.downloadedLock.RUnlock()
	return youtubeSong.downloaded
}

func (youtubeSong *YoutubeSong) setDownloaded(downloaded bool) {
	youtubeSong.downloadedLock.Lock()
	defer youtubeSong.downloadedLock.Unlock()
	youtubeSong.downloaded = downloaded
}

func (youtubeSong *YoutubeSong) isDownloading() bool {
	youtubeSong.downloadingLock.RLock()
	defer youtubeSong.downloadingLock.RUnlock()
	return youtubeSong.downloading
}

func (youtubeSong *YoutubeSong) setDownloading(downloading bool) {
	youtubeSong.downloadingLock.Lock()
	defer youtubeSong.downloadingLock.Unlock()
	youtubeSong.downloading = downloading
}

func (youtubeSong *YoutubeSong) read() ([]byte, error) {
	youtubeSong.rwLock.RLock()
	defer youtubeSong.rwLock.RUnlock()
	return ioutil.ReadFile(youtubeSong.getFilePath())
}

func (youtubeSong *YoutubeSong) getGoogleUrl() string {
	youtubeSong.googleUrlLock.RLock()
	defer youtubeSong.googleUrlLock.RUnlock()
	return youtubeSong.googleUrl
}

func (youtubeSong *YoutubeSong) download(youtubeDB *YoutubeDB) error {
	youtubeSong.setDownloading(true)
	defer youtubeSong.setDownloading(false)
	defer youtubeSong.setDownloaded(true)
	youtubeSong.rwLock.Lock()
	defer youtubeSong.rwLock.Unlock()

	info, err := youtubeDB.ytdl.GetVideoInfoFromID(youtubeSong.id)
	if err != nil {
		defer youtubeSong.googleUrlLock.Unlock()
		return err
	}

	formats := info.Formats.Worst(ytdl.FormatAudioEncodingKey)
	var downloadFormat ytdl.Format
	for _, format := range formats {
		if format.AudioEncoding == "opus" {
			downloadFormat = format
			break
		}
	}

	url, err := info.GetDownloadURL(downloadFormat)
	if err != nil {
		defer youtubeSong.googleUrlLock.Unlock()
		return err
	}
	youtubeSong.googleUrl = url.String()
	youtubeSong.googleUrlLock.Unlock()

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
