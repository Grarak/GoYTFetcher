package database

import (
	"crypto/aes"
	"io/ioutil"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/Grarak/GoYTFetcher/logger"
	"github.com/Grarak/GoYTFetcher/utils"
	"github.com/Grarak/GoYTFetcher/ytdl"
)

type YoutubeSong struct {
	id string

	downloadUrl     string
	downloadUrlTime time.Time

	count       int
	downloaded  bool
	downloading bool

	filePath string
	deleted  bool

	encryptedId string

	songLock  sync.Mutex
	stateLock sync.RWMutex
	readLock  sync.RWMutex
}

func newYoutubeSong(id string) *YoutubeSong {
	return &YoutubeSong{id: id, count: 1}
}

func (youtubeSong *YoutubeSong) isDownloaded() bool {
	youtubeSong.stateLock.RLock()
	defer youtubeSong.stateLock.RUnlock()
	return youtubeSong.downloaded
}

func (youtubeSong *YoutubeSong) setDownloaded(downloaded bool) {
	youtubeSong.stateLock.Lock()
	defer youtubeSong.stateLock.Unlock()
	youtubeSong.downloaded = downloaded
}

func (youtubeSong *YoutubeSong) IsDownloading() bool {
	youtubeSong.stateLock.RLock()
	defer youtubeSong.stateLock.RUnlock()
	return youtubeSong.downloading
}

func (youtubeSong *YoutubeSong) setDownloading(downloading bool) {
	youtubeSong.stateLock.Lock()
	defer youtubeSong.stateLock.Unlock()
	youtubeSong.downloading = downloading
}

func (youtubeSong *YoutubeSong) Read() ([]byte, error) {
	youtubeSong.readLock.RLock()
	defer youtubeSong.readLock.RUnlock()
	return ioutil.ReadFile(youtubeSong.filePath)
}

func (youtubeSong *YoutubeSong) getDownloadUrl() (string, error) {
	currentTime := time.Now()
	if currentTime.Sub(youtubeSong.downloadUrlTime).Hours() < 1 &&
		!utils.StringIsEmpty(youtubeSong.downloadUrl) {
		return youtubeSong.downloadUrl, nil
	}

	info, err := ytdl.GetVideoDownloadInfo(youtubeSong.id)
	if err != nil {
		defer youtubeSong.setDownloading(false)
		return "", err
	}

	var link *url.URL
	if info.VideoInfo.Duration.Minutes() <= 20 {
		link, err = info.GetDownloadURL()
	} else {
		link, err = info.GetDownloadURLWorst()
	}
	if err != nil {
		return "", err
	}

	youtubeSong.downloadUrl = link.String()
	youtubeSong.downloadUrlTime = currentTime

	return youtubeSong.downloadUrl, nil
}

func (youtubeSong *YoutubeSong) download(youtubeDB *youtubeDBImpl) error {
	youtubeSong.setDownloading(true)

	info, err := ytdl.GetVideoDownloadInfo(youtubeSong.id)
	if err != nil {
		youtubeSong.setDownloading(false)
		return err
	}

	if info.VideoInfo.Duration.Minutes() <= 20 {
		logger.I("Downloading " + info.VideoInfo.Title)
		defer logger.I("Finished downloading " + info.VideoInfo.Title)

		defer youtubeSong.setDownloading(false)

		path, err := info.VideoInfo.Download(utils.YOUTUBE_DIR, youtubeDB.youtubeDL)
		if err != nil {
			return err
		}
		youtubeSong.filePath = path

		if youtubeSong.deleted {
			os.Remove(youtubeSong.filePath)
		} else {
			defer youtubeSong.setDownloaded(true)
		}
		return nil
	}
	logger.I(info.VideoInfo.Title + " is too long, skipping download")
	return nil
}

func (youtubeSong *YoutubeSong) delete() {
	youtubeSong.readLock.Lock()
	defer youtubeSong.readLock.Unlock()
	youtubeSong.deleted = true

	if youtubeSong.isDownloaded() {
		os.Remove(youtubeSong.filePath)
	}
}

func (youtubeSong *YoutubeSong) getEncryptedId(key []byte) string {
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
	youtubeSong.stateLock.Lock()
	defer youtubeSong.stateLock.Unlock()
	youtubeSong.count++
}

func (youtubeSong YoutubeSong) GetUniqueId() string {
	return youtubeSong.id
}

func (youtubeSong YoutubeSong) GetCount() int {
	return youtubeSong.count
}
