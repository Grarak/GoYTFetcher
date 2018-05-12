package logger

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"sync"

	"github.com/Grarak/GoYTFetcher/utils"
	"github.com/op/go-logging"
)

const logFileName = utils.LOG_DIR + utils.LOG_EXTENSION

var logFileRegex = regexp.MustCompile(utils.LOG_PREFIX + "(\\d+)\\" + utils.LOG_EXTENSION)

const LogFilesLimit = 20
const LogFileSize = 100 * 1024

var log = logging.MustGetLogger("example")
var format = logging.MustStringFormatter("%{color}%{time:Jan 2 15:04:05.000}: %{message}%{color:reset}")
var lock sync.Mutex
var logFile *os.File

func Init() {
	utils.MkDir(utils.LOG_DIR)
	if !utils.FileExists(utils.LOG_DIR + "/log.txt") {
		_, err := os.Create(utils.LOG_DIR + "/log.txt")
		utils.Panic(err)
	}

	file, err := os.OpenFile(utils.LOG_DIR+"/log.txt",
		os.O_APPEND|os.O_WRONLY, 0600)
	utils.Panic(err)
	logFile = file

	consoleBackend := logging.NewLogBackend(os.Stderr, "", 0)
	fileBackend := logging.NewLogBackend(logFile, "", 0)

	logging.SetBackend(logging.NewBackendFormatter(consoleBackend, format),
		logging.NewBackendFormatter(fileBackend, format))
}

func I(message interface{}) {
	lock.Lock()
	defer lock.Unlock()

	text := fmt.Sprintf("%v", message)
	log.Info(text)
	checkLogSize()
}

func E(message interface{}) {
	lock.Lock()
	defer lock.Unlock()

	text := fmt.Sprintf("%v", message)
	log.Error(text)
	checkLogSize()
}

func checkLogSize() {
	info, err := logFile.Stat()
	utils.Panic(err)

	if info.Size() >= LogFileSize {
		utils.Panic(os.Rename(utils.LOG_DIR+"/"+logFileName,
			utils.LOG_DIR+"/"+newLogFile(0)))

		files, err := ioutil.ReadDir(utils.LOG_DIR)
		utils.Panic(err)

		highestCount := 0
		for _, fileInfo := range files {
			if logFileRegex.MatchString(fileInfo.Name()) {
				count, err := strconv.Atoi(logFileRegex.
					FindAllStringSubmatch(fileInfo.Name(), 1)[0][1])
				utils.Panic(err)

				if count > highestCount {
					highestCount = count
				}
			}
		}

		for i := LogFilesLimit; i <= highestCount; i++ {
			filePath := utils.LOG_DIR + "/" + newLogFile(i)
			utils.Panic(os.Remove(filePath))
		}
		for ; highestCount >= 0; highestCount-- {
			filePath := utils.LOG_DIR + "/" + newLogFile(highestCount)
			if utils.FileExists(filePath) {
				newFilePath := utils.LOG_DIR + "/" + newLogFile(highestCount+1)
				utils.Panic(os.Rename(filePath, newFilePath))
			}
		}

		logFile.Close()
		Init()
	}
}

func newLogFile(count int) string {
	return fmt.Sprintf(utils.LOG_PREFIX+"%d"+utils.LOG_EXTENSION, count)
}
