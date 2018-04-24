package database

import (
	"sync"
	"io/ioutil"
	"os"

	"../utils"
	"crypto/aes"
	"../logger"
)

type YoutubeSong struct {
	id string

	googleUrl     string
	googleUrlLock sync.RWMutex

	count     int
	countLock sync.RWMutex

	downloaded   bool
	downloading  bool
	downloadLock sync.RWMutex

	filePath string
	deleted  bool

	encryptedId string
	rwLock      sync.RWMutex
}

func newYoutubeSong(id string) *YoutubeSong {
	return &YoutubeSong{id: id, count: 1}
}

func (youtubeSong *YoutubeSong) isDownloaded() bool {
	youtubeSong.downloadLock.RLock()
	defer youtubeSong.downloadLock.RUnlock()
	return youtubeSong.downloaded
}

func (youtubeSong *YoutubeSong) setDownloaded(downloaded bool) {
	youtubeSong.downloadLock.Lock()
	defer youtubeSong.downloadLock.Unlock()
	youtubeSong.downloaded = downloaded
}

func (youtubeSong *YoutubeSong) isDownloading() bool {
	youtubeSong.downloadLock.RLock()
	defer youtubeSong.downloadLock.RUnlock()
	return youtubeSong.downloading
}

func (youtubeSong *YoutubeSong) setDownloading(downloading bool) {
	youtubeSong.downloadLock.Lock()
	defer youtubeSong.downloadLock.Unlock()
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
	youtubeSong.rwLock.Lock()
	defer youtubeSong.rwLock.Unlock()

	info, err := youtubeDB.ytdl.GetVideoInfoFromID(youtubeSong.id)
	if err != nil {
		defer youtubeSong.setDownloading(false)
		defer youtubeSong.googleUrlLock.Unlock()
		return err
	}

	var url string
	if info.Duration.Minutes() <= 20 {
		url, err = info.GetDownloadURL(youtubeDB.youtubeDL)
	} else {
		url, err = info.GetDownloadURLWorst(youtubeDB.youtubeDL)
	}
	if err != nil {
		defer youtubeSong.setDownloading(false)
		defer youtubeSong.googleUrlLock.Unlock()
		return err
	}
	youtubeSong.googleUrl = url
	youtubeSong.googleUrlLock.Unlock()

	if info.Duration.Minutes() <= 20 {
		logger.I("Downloading " + info.Title)
		defer logger.I("Finished downloading " + info.Title)

		defer youtubeSong.setDownloading(false)
		defer youtubeSong.setDownloaded(true)

		path, err := info.Download(utils.YOUTUBE_DIR, youtubeDB.youtubeDL, youtubeDB.ffmpeg)
		if err != nil {
			return err
		}
		youtubeSong.filePath = path

		if youtubeSong.deleted {
			os.Remove(youtubeSong.getFilePath())
		}
		return nil
	}
	logger.I(info.Title + " is too long, skipping download")
	return nil
}

func (youtubeSong *YoutubeSong) delete() error {
	youtubeSong.rwLock.Lock()
	defer youtubeSong.rwLock.Unlock()
	youtubeSong.deleted = true
	return os.Remove(youtubeSong.getFilePath())
}

func (youtubeSong *YoutubeSong) getFilePath() string {
	return youtubeSong.filePath
}

func (youtubeSong *YoutubeSong) getEncryptedId(key []byte) string {
	youtubeSong.rwLock.RLock()
	defer youtubeSong.rwLock.RUnlock()
	if utils.StringIsEmpty(youtubeSong.encryptedId) {
		id := youtubeSong.id
		for i := len(id); i < aes.BlockSize; i++ {
			id += " "
		}
		youtubeSong.encryptedId = utils.Encrypt(key, id)
	}
	return youtubeSong.encryptedId
}

func (youtubeSong *YoutubeSong) increaseCount() {
	youtubeSong.countLock.Lock()
	defer youtubeSong.countLock.Unlock()
	youtubeSong.count++
}

func (youtubeSong YoutubeSong) GetUniqueId() string {
	return youtubeSong.id
}

func (youtubeSong YoutubeSong) GetCount() int {
	youtubeSong.countLock.RLock()
	defer youtubeSong.countLock.RUnlock()
	return youtubeSong.count
}
