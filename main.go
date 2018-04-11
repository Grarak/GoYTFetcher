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
	logger.I(client.IPAddr + ": requesting " + client.Url +
		" with queries: " + fmt.Sprintf("%v", client.Queries) +
		" headers: " + fmt.Sprintf("%v", client.Header))

	args := strings.Split(client.Url, "/")[1:]
	if args[0] == "api" {
		return api.GetResponse(args[1], args[2], args[3:], client)
	}

	response := client.ResponseBody("Not found")
	response.SetStatusCode(http.StatusNotFound)
	return response
}

func main() {
	if _, err := exec.LookPath(utils.YOUTUBE_DL); err != nil {
		logger.E(utils.YOUTUBE_DL + " is not installed!")
		return
	}

	var port int
	var ytKey string
	flag.IntVar(&port, "p", 6713, "Which port to use")
	flag.StringVar(&ytKey, "yt", "", "Youtube Api key")
	flag.Parse()

	utils.MkDir(utils.DATABASE)
	utils.MkDir(utils.YOUTUBE_DIR)

	databaseInstance := database.GetDatabase()
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
