package database

import (
	"io/ioutil"
	"os"
	"sync"

	"crypto/aes"
	"net/url"

	"github.com/Grarak/GoYTFetcher/logger"
	"github.com/Grarak/GoYTFetcher/utils"
	"github.com/Grarak/GoYTFetcher/ytdl"
)

type YoutubeSong struct {
	id string

	googleUrl     string
	googleUrlLock sync.RWMutex

	count       int
	downloaded  bool
	downloading bool

	filePath string
	deleted  bool

	encryptedId string

	valuesLock sync.RWMutex
	rwLock     sync.RWMutex
}

func newYoutubeSong(id string) *YoutubeSong {
	return &YoutubeSong{id: id, count: 1}
}

func (youtubeSong *YoutubeSong) isDownloaded() bool {
	youtubeSong.valuesLock.RLock()
	defer youtubeSong.valuesLock.RUnlock()
	return youtubeSong.downloaded
}

func (youtubeSong *YoutubeSong) setDownloaded(downloaded bool) {
	youtubeSong.valuesLock.Lock()
	defer youtubeSong.valuesLock.Unlock()
	youtubeSong.downloaded = downloaded
}

func (youtubeSong *YoutubeSong) IsDownloading() bool {
	youtubeSong.valuesLock.RLock()
	defer youtubeSong.valuesLock.RUnlock()
	return youtubeSong.downloading
}

func (youtubeSong *YoutubeSong) setDownloading(downloading bool) {
	youtubeSong.valuesLock.Lock()
	defer youtubeSong.valuesLock.Unlock()
	youtubeSong.downloading = downloading
}

func (youtubeSong *YoutubeSong) Read() ([]byte, error) {
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

	info, err := ytdl.GetVideoDownloadInfo(youtubeSong.id)
	if err != nil {
		defer youtubeSong.setDownloading(false)
		defer youtubeSong.googleUrlLock.Unlock()
		return err
	}

	var link *url.URL
	if info.VideoInfo.Duration.Minutes() <= 20 {
		link, err = info.GetDownloadURL()
	} else {
		link, err = info.GetDownloadURLWorst()
	}
	if err != nil {
		defer youtubeSong.setDownloading(false)
		defer youtubeSong.googleUrlLock.Unlock()
		return err
	}
	youtubeSong.googleUrl = link.String()
	youtubeSong.googleUrlLock.Unlock()

	if info.VideoInfo.Duration.Minutes() <= 20 {
		logger.I("Downloading " + info.VideoInfo.Title)
		defer logger.I("Finished downloading " + info.VideoInfo.Title)

		defer youtubeSong.setDownloading(false)
		defer youtubeSong.setDownloaded(true)

		path, err := info.VideoInfo.Download(utils.YOUTUBE_DIR, youtubeDB.youtubeDL)
		if err != nil {
			return err
		}
		youtubeSong.filePath = path

		if youtubeSong.deleted {
			os.Remove(youtubeSong.getFilePath())
		}
		return nil
	}
	logger.I(info.VideoInfo.Title + " is too long, skipping download")
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
	youtubeSong.valuesLock.Lock()
	defer youtubeSong.valuesLock.Unlock()
	youtubeSong.count++
}

func (youtubeSong YoutubeSong) GetUniqueId() string {
	return youtubeSong.id
}

func (youtubeSong YoutubeSong) GetCount() int {
	return youtubeSong.count
}
