package v1

import (
	"../../miniserver"
	"../../database"
	"../../utils"
	"net/http"
)

func youtubeFetch(client *miniserver.Client) *miniserver.Response {
	request, err := database.NewYoutube(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	userDB := database.GetDatabase().UserDB
	if requester, err := userDB.FindUserByApiKey(request.ApiKey);
		err == nil && *requester.Verified {

		id, err := database.GetDatabase().YoutubeDB.FetchYoutubeSong(request.Id)
		if err != nil {
			client.CreateResponse(utils.StatusYoutubeFetchFailure)
		}

		database.GetDatabase().HistoriesDB.AddHistory(request.ApiKey, request.Id)
		return client.ResponseBody(id)
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func youtubeGet(client *miniserver.Client) *miniserver.Response {
	id := client.Queries.Get("id")
	if utils.StringIsEmpty(id) {
		return client.CreateResponse(utils.StatusInvalid)
	}

	data, err := database.GetDatabase().YoutubeDB.GetYoutubeSong(id)
	if err != nil {
		client.CreateResponse(utils.StatusYoutubeGetFailure)
	}

	response := client.ResponseBodyBytes(data)
	response.SetContentType(miniserver.ContentWebm)
	return response
}

func youtubeSearch(client *miniserver.Client) *miniserver.Response {
	request, err := database.NewYoutube(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	userDB := database.GetDatabase().UserDB
	if requester, err := userDB.FindUserByApiKey(request.ApiKey);
		err == nil && *requester.Verified {
		results, err := database.GetDatabase().YoutubeDB.GetYoutubeSearch(request.SearchQuery)
		if err != nil {
			client.CreateResponse(utils.StatusYoutubeSearchFailure)
		}
		return client.CreateJsonResponse(results)
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func youtubeGetInfo(client *miniserver.Client) *miniserver.Response {
	request, err := database.NewYoutube(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	userDB := database.GetDatabase().UserDB
	if requester, err := userDB.FindUserByApiKey(request.ApiKey);
		err == nil && *requester.Verified {
		info, err := database.GetDatabase().YoutubeDB.GetYoutubeInfo(request.Id)
		if err != nil {
			client.CreateResponse(utils.StatusYoutubeGetInfoFailure)
		}
		return client.CreateJsonResponse(info)
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func HandleYoutubeV1(path string, client *miniserver.Client) *miniserver.Response {
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
	}

	return nil
}