package v1

import "github.com/Grarak/GoYTFetcher/miniserver"

func HandleInfoV1(_ string, client *miniserver.Client) miniserver.Response {
	return client.ResponseBody("Welcome to V1 API!")
}
