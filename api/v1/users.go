package v1

import (
	"../../miniserver"
	"net/http"

	"../../database"
	"../../utils"
	"strconv"
	"../../logger"
	"fmt"
)

func usersSignUp(client *miniserver.Client) *miniserver.Response {
	request, err := database.NewUser(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	userDB := database.GetDatabase().UserDB
	user, code := userDB.AddUser(request)
	if code == utils.StatusNoError {
		logger.I(fmt.Sprintf("Created new user %s", user.Name))
		return client.CreateJsonResponse(user)
	}

	return client.CreateResponse(code)
}

func usersLogin(client *miniserver.Client) *miniserver.Response {
	request, err := database.NewUser(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	userDB := database.GetDatabase().UserDB
	user, code := userDB.GetUserWithPassword(request.Name, request.Password)
	if code == utils.StatusNoError {
		return client.CreateJsonResponse(user)
	}

	return client.CreateResponse(code)
}

func usersList(client *miniserver.Client) *miniserver.Response {
	request, err := database.NewUser(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	userDB := database.GetDatabase().UserDB
	if user, err := userDB.FindUserByApiKey(request.ApiKey);
		err == nil && *user.Verified {
		page, err := strconv.Atoi(client.Queries.Get("page"))
		if err != nil {
			page = 1
		}
		users, err := userDB.ListUsers(page)
		if err == nil {
			return client.CreateJsonResponse(users)
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func usersSetVerification(client *miniserver.Client) *miniserver.Response {
	request, err := database.NewUser(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	userDB := database.GetDatabase().UserDB
	if requester, err := userDB.FindUserByApiKey(request.ApiKey);
		err == nil && *requester.Admin {
		err = userDB.SetVerificationUser(request)
		if err == nil {
			return client.CreateResponse(utils.StatusNoError)
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func usersDelete(client *miniserver.Client) *miniserver.Response {
	request, err := database.NewUser(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	userDB := database.GetDatabase().UserDB
	if requester, err := userDB.FindUserByApiKey(request.ApiKey);
		err == nil && *requester.Admin {
		err = userDB.DeleteUser(request)
		if err != nil {
			return client.CreateResponse(utils.StatusNoError)
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func usersDeleteAll(client *miniserver.Client) *miniserver.Response {
	request, err := database.NewUser(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	userDB := database.GetDatabase().UserDB
	if requester, err := userDB.FindUserByApiKey(request.ApiKey);
		err == nil && *requester.Admin {
		err = userDB.DeleteAllNonVerifiedUsers(request)
		if err == nil {
			return client.CreateResponse(utils.StatusNoError)
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func usersResetPassword(client *miniserver.Client) *miniserver.Response {
	request, err := database.NewUser(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	userDB := database.GetDatabase().UserDB
	if requester, err := userDB.FindUserByApiKey(request.ApiKey);
		err == nil && *requester.Admin {
		err = userDB.ResetPasswordUser(request)
		if err == nil {
			return client.CreateResponse(utils.StatusNoError)
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func playlistList(client *miniserver.Client) *miniserver.Response {
	request, err := database.NewPlayListName(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	playlistNamesDB := database.GetDatabase().PlaylistNamesDB
	list, err := playlistNamesDB.ListPlaylistNames(request.ApiKey, false)
	if err == nil {
		return client.CreateJsonResponse(list)
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func playlistListPublic(client *miniserver.Client) *miniserver.Response {
	request, err := database.NewPlayListName(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	userDB := database.GetDatabase().UserDB
	requester, err := userDB.FindUserByApiKey(request.ApiKey)
	if err == nil && *requester.Verified {
		user, err := userDB.FindUserByName(request.Name)
		playlistNamesDB := database.GetDatabase().PlaylistNamesDB
		list, err := playlistNamesDB.ListPlaylistNames(user.ApiKey, true)
		if err == nil {
			return client.CreateJsonResponse(list)
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func playlistCreate(client *miniserver.Client) *miniserver.Response {
	request, err := database.NewPlayListName(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	userDB := database.GetDatabase().UserDB
	user, err := userDB.FindUserByApiKey(request.ApiKey)
	if err == nil && *user.Verified {
		playlistNamesDB := database.GetDatabase().PlaylistNamesDB
		err := playlistNamesDB.CreatePlaylistName(request)
		if err != nil {
			return client.CreateResponse(utils.StatusPlaylistAlreadyExists)
		}
		return client.CreateResponse(utils.StatusNoError)
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func playlistDelete(client *miniserver.Client) *miniserver.Response {
	request, err := database.NewPlayListName(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	userDB := database.GetDatabase().UserDB
	user, err := userDB.FindUserByApiKey(request.ApiKey)
	if err == nil && *user.Verified {
		playlistNamesDB := database.GetDatabase().PlaylistNamesDB
		err := playlistNamesDB.DeletePlaylistName(request)
		if err == nil {
			return client.CreateResponse(utils.StatusNoError)
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func playlistSetPublic(client *miniserver.Client) *miniserver.Response {
	request, err := database.NewPlayListName(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	playlistNamesDB := database.GetDatabase().PlaylistNamesDB
	err = playlistNamesDB.SetPlaylistNamePublic(request)
	if err == nil {
		return client.CreateResponse(utils.StatusNoError)
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func playlistListLinks(client *miniserver.Client) *miniserver.Response {
	request, err := database.NewPlayListName(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	playlistsDB := database.GetDatabase().PlaylistsDB
	list, err := playlistsDB.ListPlaylistLinks(request)
	if err == nil {
		return client.CreateJsonResponse(list)
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func playlistAddLink(client *miniserver.Client) *miniserver.Response {
	request, err := database.NewPlaylist(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	userDB := database.GetDatabase().UserDB
	user, err := userDB.FindUserByApiKey(request.ApiKey)
	if err == nil && *user.Verified {
		playlistDB := database.GetDatabase().PlaylistsDB
		err := playlistDB.AddPlaylistLink(request)
		if err == nil {
			return client.CreateResponse(utils.StatusNoError)
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func playlistDeleteLink(client *miniserver.Client) *miniserver.Response {
	request, err := database.NewPlaylist(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	userDB := database.GetDatabase().UserDB
	user, err := userDB.FindUserByApiKey(request.ApiKey)
	if err == nil && *user.Verified {
		playlistDB := database.GetDatabase().PlaylistsDB
		err := playlistDB.DeletePlaylistLink(request)
		if err == nil {
			return client.CreateResponse(utils.StatusNoError)
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func historyList(client *miniserver.Client) *miniserver.Response {
	request, err := database.NewHistory(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	historiesDB := database.GetDatabase().HistoriesDB
	histories, err := historiesDB.GetHistory(request.ApiKey)
	if err == nil {
		return client.CreateJsonResponse(histories)
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func HandleUsersV1(path string, client *miniserver.Client) *miniserver.Response {
	if client.Method != http.MethodPost || !client.IsContentJson() {
		return nil
	}

	switch path {

	// user database
	case "signup":
		return usersSignUp(client)
	case "login":
		return usersLogin(client)
	case "list":
		return usersList(client)
	case "setverification":
		return usersSetVerification(client)
	case "delete":
		return usersDelete(client)
	case "deleteall":
		return usersDeleteAll(client)
	case "resetpassword":
		return usersResetPassword(client)

		// playlist database
	case "playlist/list":
		return playlistList(client)
	case "playlist/listpublic":
		return playlistListPublic(client)
	case "playlist/create":
		return playlistCreate(client)
	case "playlist/delete":
		return playlistDelete(client)
	case "playlist/setpublic":
		return playlistSetPublic(client)
	case "playlist/listlinks":
		return playlistListLinks(client)
	case "playlist/addlink":
		return playlistAddLink(client)
	case "playlist/deletelink":
		return playlistDeleteLink(client)

		// history database
	case "history/list":
		return historyList(client)
	}

	return nil
}
