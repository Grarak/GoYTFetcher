package api

import (
	"strings"

	"github.com/Grarak/GoYTFetcher/api/v1"
	"github.com/Grarak/GoYTFetcher/miniserver"
)

type apiHandle func(path string, client *miniserver.Client) miniserver.Response

var v1Apis = map[string]apiHandle{
	"users":   v1.HandleUsersV1,
	"youtube": v1.HandleYoutubeV1,
}

// GetResponse makes the request and gets the response from the server
func GetResponse(version, api string, args []string, client *miniserver.Client) miniserver.Response {
	var response apiHandle
	switch version {
	case "v1":
		response = v1Apis[api]
		break
	}
	if response != nil {
		return response(strings.Join(args, "/"), client)
	}
	return nil
}
