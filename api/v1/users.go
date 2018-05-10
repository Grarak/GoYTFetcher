package v1

import (
	"../../miniserver"
	"net/http"

	"../../database"
	"../../utils"
	"strconv"
	"../../logger"
)

func usersSignUp(client *miniserver.Client) miniserver.Response {
	request, err := database.NewUser(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	usersDB := database.GetDatabase().UsersDB
	user, code := usersDB.AddUser(request)
	if code == utils.StatusNoError {
		logger.I(client.IPAddr + ": " + "Created new user " + user.Name)
		return client.CreateJsonResponse(user)
	}

	return client.CreateResponse(code)
}

func usersLogin(client *miniserver.Client) miniserver.Response {
	request, err := database.NewUser(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	usersDB := database.GetDatabase().UsersDB
	user, code := usersDB.GetUserWithPassword(request.Name, request.Password)
	if code == utils.StatusNoError {
		logger.I(client.IPAddr + ": " + user.Name + " logged in")
		return client.CreateJsonResponse(user)
	}

	return client.CreateResponse(code)
}

func usersList(client *miniserver.Client) miniserver.Response {
	request, err := database.NewUser(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	usersDB := database.GetDatabase().UsersDB
	if user, err := usersDB.FindUserByApiKey(request.ApiKey);
		err == nil && *user.Verified {
		page, err := strconv.Atoi(client.Queries.Get("page"))
		if err != nil {
			page = 1
		}
		users, err := usersDB.ListUsers(page)
		if err == nil {
			return client.CreateJsonResponse(users)
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func usersSetVerification(client *miniserver.Client) miniserver.Response {
	request, err := database.NewUser(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	usersDB := database.GetDatabase().UsersDB
	if requester, err := usersDB.FindUserByApiKey(request.ApiKey);
		err == nil && *requester.Admin {
		err = usersDB.SetVerificationUser(request)
		if err == nil {
			return client.CreateResponse(utils.StatusNoError)
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func usersDelete(client *miniserver.Client) miniserver.Response {
	request, err := database.NewUser(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	usersDB := database.GetDatabase().UsersDB
	if requester, err := usersDB.FindUserByApiKey(request.ApiKey);
		err == nil && *requester.Admin {
		err = usersDB.DeleteUser(request)
		if err != nil {
			return client.CreateResponse(utils.StatusNoError)
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func usersDeleteAll(client *miniserver.Client) miniserver.Response {
	request, err := database.NewUser(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	usersDB := database.GetDatabase().UsersDB
	if requester, err := usersDB.FindUserByApiKey(request.ApiKey);
		err == nil && *requester.Admin {
		err = usersDB.DeleteAllNonVerifiedUsers(request)
		if err == nil {
			return client.CreateResponse(utils.StatusNoError)
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func usersResetPassword(client *miniserver.Client) miniserver.Response {
	request, err := database.NewUser(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	usersDB := database.GetDatabase().UsersDB
	if requester, err := usersDB.FindUserByApiKey(request.ApiKey);
		err == nil && *requester.Admin {
		err = usersDB.ResetPasswordUser(request)
		if err == nil {
			return client.CreateResponse(utils.StatusNoError)
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func playlistList(client *miniserver.Client) miniserver.Response {
	request, err := database.NewPlaylist(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	playlistsDB := database.GetDatabase().PlaylistsDB
	playlists, err := playlistsDB.GetPlaylists(request.ApiKey, false)
	if err == nil {
		return client.CreateJsonResponse(playlists)
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func playlistListPublic(client *miniserver.Client) miniserver.Response {
	request, err := database.NewPlaylist(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	usersDB := database.GetDatabase().UsersDB
	playlistsDB := database.GetDatabase().PlaylistsDB
	if requester, err := usersDB.FindUserByApiKey(request.ApiKey);
		err == nil && *requester.Verified {

		user, err := usersDB.FindUserByName(request.Name)
		if err == nil {
			playlists, err := playlistsDB.GetPlaylists(user.ApiKey, true)
			if err == nil {
				return client.CreateJsonResponse(playlists)
			}
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func playlistCreate(client *miniserver.Client) miniserver.Response {
	request, err := database.NewPlaylist(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	usersDB := database.GetDatabase().UsersDB
	playlistsDB := database.GetDatabase().PlaylistsDB
	if requester, err := usersDB.FindUserByApiKey(request.ApiKey);
		err == nil && *requester.Verified {

		err := playlistsDB.CreatePlaylist(request)
		if err == nil {
			return client.CreateResponse(utils.StatusNoError)
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func playlistDelete(client *miniserver.Client) miniserver.Response {
	request, err := database.NewPlaylist(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	playlistsDB := database.GetDatabase().PlaylistsDB
	err = playlistsDB.DeletePlaylist(request)
	if err == nil {
		return client.CreateResponse(utils.StatusNoError)
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func playlistSetPublic(client *miniserver.Client) miniserver.Response {
	request, err := database.NewPlaylist(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	playlistsDB := database.GetDatabase().PlaylistsDB
	err = playlistsDB.SetPublic(request)
	if err == nil {
		return client.CreateResponse(utils.StatusNoError)
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func playlistListIds(client *miniserver.Client) miniserver.Response {
	request, err := database.NewPlaylist(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	playlistsDB := database.GetDatabase().PlaylistsDB
	ids, err := playlistsDB.GetPlaylistIds(request)
	if err == nil {
		return client.CreateJsonResponse(ids)
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func playlistListIdsPublic(client *miniserver.Client) miniserver.Response {
	request, err := database.NewPlaylistPublic(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	usersDB := database.GetDatabase().UsersDB
	playlistsDB := database.GetDatabase().PlaylistsDB
	if requester, err := usersDB.FindUserByApiKey(request.ApiKey);
		err == nil && *requester.Verified {
		user, err := usersDB.FindUserByName(request.Name)
		if err == nil {
			playlist := database.Playlist{ApiKey: user.ApiKey, Name: request.Playlist}
			if playlistsDB.IsPlaylistPublic(playlist) {
				ids, err := playlistsDB.GetPlaylistIds(playlist)
				if err == nil {
					return client.CreateJsonResponse(ids)
				}
			}
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func playlistAddId(client *miniserver.Client) miniserver.Response {
	request, err := database.NewPlaylistId(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	usersDB := database.GetDatabase().UsersDB
	playlistsDB := database.GetDatabase().PlaylistsDB
	if requester, err := usersDB.FindUserByApiKey(request.ApiKey);
		err == nil && *requester.Verified {
		err = playlistsDB.AddIdToPlaylist(request)
		if err != nil {
			return client.CreateResponse(utils.StatusPlaylistIdAlreadyExists)
		}

		logger.I(client.IPAddr + ": " + requester.Name + " adding " +
			request.Id + " to playlist " + request.Name)
		return client.CreateResponse(utils.StatusNoError)
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func playlistDeleteId(client *miniserver.Client) miniserver.Response {
	request, err := database.NewPlaylistId(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	usersDB := database.GetDatabase().UsersDB
	playlistsDB := database.GetDatabase().PlaylistsDB
	if requester, err := usersDB.FindUserByApiKey(request.ApiKey);
		err == nil && *requester.Verified {
		err := playlistsDB.DeleteIdFromPlaylist(request)
		if err == nil {
			return client.CreateResponse(utils.StatusNoError)
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func playlistSetIds(client *miniserver.Client) miniserver.Response {
	request, err := database.NewPlaylistIds(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	usersDB := database.GetDatabase().UsersDB
	playlistsDB := database.GetDatabase().PlaylistsDB
	if requester, err := usersDB.FindUserByApiKey(request.ApiKey);
		err == nil && *requester.Verified {
		err := playlistsDB.SetPlaylistIds(request)
		if err == nil {
			return client.CreateResponse(utils.StatusNoError)
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func historyAdd(client *miniserver.Client) miniserver.Response {
	request, err := database.NewHistory(client.Request)
	if err != nil {
		return client.CreateResponse(utils.StatusInvalid)
	}

	userDB := database.GetDatabase().UsersDB
	historiesDB := database.GetDatabase().HistoriesDB
	if requester, err := userDB.FindUserByApiKey(request.ApiKey);
		err == nil && *requester.Verified {
		err = historiesDB.AddHistory(request.ApiKey, request.Id)
		if err == nil {
			logger.I(client.IPAddr + ": " + requester.Name +
				" adding " + request.Id + " to history")
			return client.CreateResponse(utils.StatusNoError)
		}
	}

	return client.CreateResponse(utils.StatusInvalid)
}

func historyList(client *miniserver.Client) miniserver.Response {
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

func HandleUsersV1(path string, client *miniserver.Client) miniserver.Response {
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
	case "playlist/listids":
		return playlistListIds(client)
	case "playlist/listidspublic":
		return playlistListIdsPublic(client)
	case "playlist/addid":
		return playlistAddId(client)
	case "playlist/deleteid":
		return playlistDeleteId(client)
	case "playlist/setids":
		return playlistSetIds(client)

		// history database
	case "history/add":
		return historyAdd(client)
	case "history/list":
		return historyList(client)
	}

	return nil
}
