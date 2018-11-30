package v1

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/Grarak/GoYTFetcher/database"
	"github.com/Grarak/GoYTFetcher/logger"
	"github.com/Grarak/GoYTFetcher/miniserver"
	"github.com/Grarak/GoYTFetcher/utils"
)

func youtubeFetch(client *miniserver.Client) miniserver.Response {
	request, err := database.NewYoutube(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	usersDB := database.GetDefaultDatabase().UsersDB
	if requester, err := usersDB.FindUserByApiKey(request.ApiKey); err == nil && *requester.Verified {
		logger.I(client.IPAddr + ": " + requester.Name + " fetching " + request.Id)
		youtubeDB := database.GetDefaultDatabase().YoutubeDB
		u, id, err := youtubeDB.FetchYoutubeSong(request.Id)
		if err != nil {
			logger.E(err)
			return client.CreateResponse(utils.StatusYoutubeFetchFailure)
		}

		if request.AddHistory {
			err := database.GetDefaultDatabase().HistoriesDB.AddHistory(request.ApiKey, request.Id)
			if err != nil {
				return client.CreateResponse(utils.StatusAddHistoryFailed)
			}
		}
		if !strings.HasPrefix(u, "http") {
			query := url.Values{}
			query.Set("id", u)

			if purl, err := url.Parse(u); err == nil {
				host := purl.Host
				if !strings.HasPrefix(host, "http") {
					host = "http://" + client.Host
				}
				u = host + strings.Replace(
					client.Url, "fetch", "get", 1) + "?" + query.Encode()
			}
		}

		response := client.ResponseBody(u)
		response.SetHeader("ytfetcher-id", id)
		return response
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func youtubeGet(client *miniserver.Client) miniserver.Response {
	id := client.Queries.Get("id")
	u := client.Queries.Get("url")

	if !utils.StringIsEmpty(id) {
		youtubeSong, err := database.GetDefaultDatabase().YoutubeDB.GetYoutubeSong(id)
		if err != nil {
			return client.CreateResponse(utils.StatusYoutubeGetFailure)
		}
		if strings.Contains(u, "googlevideo") {
			return miniserver.NewForwardResponse(u)
		}

		reader, err := youtubeSong.Reader()
		if err == nil {
			response := client.ResponseReader(reader)
			response.SetContentType(miniserver.ContentOgg)
			return response
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func youtubeSearch(client *miniserver.Client) miniserver.Response {
	request, err := database.NewYoutube(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	usersDB := database.GetDefaultDatabase().UsersDB
	if requester, err := usersDB.FindUserByApiKey(request.ApiKey); err == nil && *requester.Verified {

		logger.I(client.IPAddr + ": " + requester.Name + " searching " + request.SearchQuery)
		results, err := database.GetDefaultDatabase().YoutubeDB.GetYoutubeSearch(request.SearchQuery)
		if err != nil {
			return client.CreateResponse(utils.StatusYoutubeSearchFailure)
		}
		return client.CreateJsonResponse(results)
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func youtubeGetInfo(client *miniserver.Client) miniserver.Response {
	request, err := database.NewYoutube(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	usersDB := database.GetDefaultDatabase().UsersDB
	if requester, err := usersDB.FindUserByApiKey(request.ApiKey); err == nil && *requester.Verified {
		info, err := database.GetDefaultDatabase().YoutubeDB.GetYoutubeInfo(request.Id)
		if err != nil {
			return client.CreateResponse(utils.StatusYoutubeGetInfoFailure)
		}
		return client.CreateJsonResponse(info)
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func youtubeGetCharts(client *miniserver.Client) miniserver.Response {
	request, err := database.NewYoutube(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	usersDB := database.GetDefaultDatabase().UsersDB
	if requester, err := usersDB.FindUserByApiKey(request.ApiKey); err == nil && *requester.Verified {
		info, err := database.GetDefaultDatabase().YoutubeDB.GetYoutubeCharts()
		if err != nil {
			return client.CreateResponse(utils.StatusYoutubeGetChartsFailure)
		}
		return client.CreateJsonResponse(info)
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func HandleYoutubeV1(path string, client *miniserver.Client) miniserver.Response {
	switch path {
	case "fetch":
		if client.Method == http.MethodPost && client.IsContentJson() {
			return youtubeFetch(client)
		}
		break
	case "get":
		if client.Method == http.MethodGet {
			return youtubeGet(client)
		}
	case "search":
		if client.Method == http.MethodPost && client.IsContentJson() {
			return youtubeSearch(client)
		}
		break
	case "getinfo":
		if client.Method == http.MethodPost && client.IsContentJson() {
			return youtubeGetInfo(client)
		}
		break
	case "getcharts":
		if client.Method == http.MethodPost && client.IsContentJson() {
			return youtubeGetCharts(client)
		}
		break
	}

	return nil
}
