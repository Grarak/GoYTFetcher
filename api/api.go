package api

import (
	"../miniserver"
	"./v1"
	"strings"
)

type apiHandle func(path string, client *miniserver.Client) miniserver.Response

var v1Apis = map[string]apiHandle{
	"users":   v1.HandleUsersV1,
	"youtube": v1.HandleYoutubeV1,
}

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
