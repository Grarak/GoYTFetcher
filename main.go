package main

import (
	"flag"
	"os/signal"
	"os"
	"fmt"

	"./logger"
	"./miniserver"
	"strconv"
	"strings"
	"./api"
	"./utils"
	"./database"
	"net/http"
	"os/exec"
)

func clientHandler(client *miniserver.Client) *miniserver.Response {
	log := client.IPAddr + ": " + client.Method + " " + client.Url
	if ranges := client.Header.Get("Range"); !utils.StringIsEmpty(ranges) {
		log += " " + ranges
	}

	logger.I(log)

	args := strings.Split(client.Url, "/")[1:]
	if args[0] == "api" {
		return api.GetResponse(args[1], args[2], args[3:], client)
	}

	response := client.ResponseBody("Not found")
	response.SetStatusCode(http.StatusNotFound)
	return response
}

func main() {
	logger.Init()

	if _, err := exec.LookPath(utils.YOUTUBE_DL); err != nil {
		logger.E(utils.YOUTUBE_DL + " is not installed!")
		return
	}

	ffmpeg, err := exec.LookPath(utils.FFMPEG)
	if err != nil {
		logger.E(utils.FFMPEG + " is not installed!")
		return
	}

	codecs, err := utils.ExecuteCmd(ffmpeg, "-codecs")
	if err != nil || !strings.Contains(codecs, "libvorbis") {
		logger.E(utils.FFMPEG + " vorbis is not enabled")
		return
	}

	var port int
	var host string
	var ytKey string
	flag.IntVar(&port, "p", 6713, "Which port to use")
	flag.StringVar(&host, "host", "", "Hostname (default: local IP)")
	flag.StringVar(&ytKey, "yt", "", "Youtube Api key")
	flag.Parse()

	if utils.StringIsEmpty(host) {
		host = fmt.Sprintf("%s:%d", utils.GetOutboundIP(), port)
	}

	utils.MkDir(utils.DATABASE)
	utils.MkDir(utils.YOUTUBE_DIR)

	databaseInstance := database.GetDatabase()
	databaseInstance.SetHost(host)
	databaseInstance.SetRandomKey(utils.GenerateRandom(16))
	databaseInstance.SetYTApiKey(ytKey)

	server := miniserver.NewServer(port)

	c := make(chan os.Signal, 1)
	cleanup := make(chan bool)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			logger.I(fmt.Sprintf("Captured %s, killing...", sig))
			server.StopListening()

			databaseInstance.Close()

			cleanup <- true
		}
	}()

	logger.I("Starting server on port " + strconv.Itoa(port))
	go server.StartListening(clientHandler)

	<-cleanup
}
