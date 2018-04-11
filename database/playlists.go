package database

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"../utils"
	"sync"
)

const TablePlaylists = "playlists"

type PlaylistLink struct {
	ApiKey string `json:"apikey"`
	Name   string `json:"name"`
	Id     string `json:"id"`
}

func NewPlaylist(data []byte) (PlaylistLink, error) {
	var name PlaylistLink
	err := json.Unmarshal(data, &name)
	return name, err
}

type PlaylistsDB struct {
	db     *sql.DB
	rwLock *sync.RWMutex
}

func newPlaylistsDB(db *sql.DB, rwLock *sync.RWMutex) (*PlaylistsDB, error) {
	err := createTable(db, TablePlaylists, ColumnApikey, ColumnName, ColumnId)
	if err != nil {
		return nil, err
	}

	return &PlaylistsDB{db, rwLock}, nil
}

func (playlistsDB *PlaylistsDB) ListPlaylistLinks(playlistName PlaylistName) ([]string, error) {
	playlistsDB.rwLock.RLock()
	defer playlistsDB.rwLock.RUnlock()

	row, err := playlistsDB.db.Query(fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s = '%s' AND %s = '%s",
		ColumnId.name, TablePlaylists, ColumnApikey.name, playlistName.ApiKey,
		ColumnName.name, playlistName.Name))
	if err != nil {
		return nil, err
	}
	defer row.Close()

	var links []string
	for row.Next() {
		var link string
		err := row.Scan(&link)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, nil
}

func (playlistsDB *PlaylistsDB) AddPlaylistLink(playlistLink PlaylistLink) error {
	playlistsDB.rwLock.Lock()
	defer playlistsDB.rwLock.Unlock()

	if utils.StringIsEmpty(playlistLink.Id) {
		return utils.Error("Id is empty")
	}

	row := playlistsDB.db.QueryRow(fmt.Sprintf(
		"SELECT 1 FROM %s WHERE %s = '%s' AND %s = '%s' AND %s = '%s'",
		TablePlaylists,
		ColumnApikey.name, playlistLink.ApiKey,
		ColumnName.name, playlistLink.Name,
		ColumnId.name, playlistLink.Id))

	var exists bool
	row.Scan(&exists)
	if exists {
		return utils.Error(playlistLink.Name + " already exists")
	}

	_, err := playlistsDB.db.Exec(fmt.Sprintf(
		"INSERT INTO %s (%s, %s, %s) VALUES (?, ?, ?)",
		TablePlaylists, ColumnApikey.name, ColumnName.name, ColumnId.name),
		playlistLink.ApiKey, playlistLink.Name, playlistLink.Id)
	return err
}

func (playlistsDB *PlaylistsDB) DeletePlaylistLink(playlistLink PlaylistLink) error {
	playlistsDB.rwLock.Lock()
	defer playlistsDB.rwLock.Unlock()

	_, err := playlistsDB.db.Exec(fmt.Sprintf(
		"DELETE FROM %s WHERE %s = '%s' AND %s = '%s' AND %s = '%s'",
		TablePlaylists,
		ColumnApikey.name, playlistLink.ApiKey,
		ColumnName.name, playlistLink.Name,
		ColumnId.name, playlistLink.Id))
	return err
}

func (playlistsDB *PlaylistsDB) DeletePlaylist(apiKey, name string) error {
	playlistsDB.rwLock.Lock()
	defer playlistsDB.rwLock.Unlock()

	_, err := playlistsDB.db.Exec(fmt.Sprintf(
		"DELETE FROM %s WHERE %s = '%s' AND %s = '%s'",
		TablePlaylists,
		ColumnApikey.name, apiKey,
		ColumnName.name, name))
	return err
}
