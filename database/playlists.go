package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"strings"
	"../utils"
)

const TablePlaylists = "playlists"

type Playlist struct {
	ApiKey string `json:"apikey,omitempty"`
	Name   string `json:"name"`
	Public bool   `json:"public"`
}

type PlaylistId struct {
	ApiKey string `json:"apikey,omitempty"`
	Name   string `json:"name"`
	Id     string `json:"id"`
}

type PlaylistIds struct {
	ApiKey string   `json:"apikey,omitempty"`
	Name   string   `json:"name"`
	Ids    []string `json:"ids"`
}

type PlaylistLinkPublic struct {
	ApiKey   string `json:"apikey,omitempty"`
	Name     string `json:"name"`
	Playlist string `json:"playlist"`
}

func NewPlaylist(data []byte) (Playlist, error) {
	var name Playlist
	err := json.Unmarshal(data, &name)
	return name, err
}

func NewPlaylistId(data []byte) (PlaylistId, error) {
	var name PlaylistId
	err := json.Unmarshal(data, &name)
	return name, err
}

func NewPlaylistIds(data []byte) (PlaylistIds, error) {
	var name PlaylistIds
	err := json.Unmarshal(data, &name)
	return name, err
}

func NewPlaylistPublic(data []byte) (PlaylistLinkPublic, error) {
	var name PlaylistLinkPublic
	err := json.Unmarshal(data, &name)
	return name, err
}

type PlaylistsDB struct {
	db     *sql.DB
	rwLock *sync.RWMutex
}

func newPlaylistsDB(db *sql.DB, rwLock *sync.RWMutex) (*PlaylistsDB, error) {
	cmd := newTableBuilder(TablePlaylists).
		addForeignKey(ForeignKeyApikey).
		addPrimaryKey(ColumnName).
		addColumn(ColumnPublic).
		addColumn(ColumnIds).build()

	_, err := db.Exec(cmd)
	if err != nil {
		return nil, err
	}

	return &PlaylistsDB{db, rwLock}, nil
}

func (playlistsDB *PlaylistsDB) GetPlaylists(apiKey string, publicOnly bool) ([]Playlist, error) {
	playlistsDB.rwLock.RLock()
	defer playlistsDB.rwLock.RUnlock()

	cmd := fmt.Sprintf(
		"SELECT %s,%s FROM %s WHERE %s = ?",
		ColumnName.name, ColumnPublic.name, TablePlaylists,
		ColumnApikey.name)
	if publicOnly {
		cmd += fmt.Sprintf(" AND %s = 1", ColumnPublic.name)
	}

	stmt, err := playlistsDB.db.Prepare(cmd)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(apiKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	playlists := make([]Playlist, 0)
	for rows.Next() {
		var name string
		var public bool
		err := rows.Scan(&name, &public)
		if err != nil {
			return nil, err
		}

		playlists = append(playlists, Playlist{Name: name, Public: public})
	}

	return playlists, nil
}

func (playlistsDB *PlaylistsDB) CreatePlaylist(playlist Playlist) error {
	playlistsDB.rwLock.Lock()
	defer playlistsDB.rwLock.Unlock()

	_, err := playlistsDB.db.Exec(fmt.Sprintf(
		"INSERT INTO %s (%s,%s,%s,%s) VALUES (?,?,?,?)",
		TablePlaylists,
		ColumnApikey.name, ColumnName.name, ColumnPublic.name, ColumnIds.name),
		playlist.ApiKey, playlist.Name, playlist.Public, "")
	return err
}

func (playlistsDB *PlaylistsDB) DeletePlaylist(playlist Playlist) error {
	playlistsDB.rwLock.Lock()
	defer playlistsDB.rwLock.Unlock()

	_, err := playlistsDB.db.Exec(fmt.Sprintf(
		"DELETE FROM %s WHERE %s = ? AND %s = ?",
		TablePlaylists, ColumnApikey.name, ColumnName.name),
		playlist.ApiKey, playlist.Name)
	return err
}

func (playlistsDB *PlaylistsDB) SetPublic(playlist Playlist) error {
	playlistsDB.rwLock.Lock()
	defer playlistsDB.rwLock.Unlock()

	_, err := playlistsDB.db.Exec(fmt.Sprintf(
		"UPDATE %s SET %s = ? WHERE %s = ? AND %s = ?",
		TablePlaylists, ColumnPublic.name, ColumnApikey.name, ColumnName.name),
		playlist.Public, playlist.ApiKey, playlist.Name)
	return err
}

func (playlistsDB *PlaylistsDB) GetPlaylistIds(playlist Playlist) ([]string, error) {
	playlistsDB.rwLock.RLock()
	defer playlistsDB.rwLock.RUnlock()
	return playlistsDB.getPlaylistIds(playlist)
}

func (playlistsDB *PlaylistsDB) getPlaylistIds(playlist Playlist) ([]string, error) {
	stmt, err := playlistsDB.db.Prepare(fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s = ? AND %s = ?",
		ColumnIds.name, TablePlaylists, ColumnApikey.name, ColumnName.name))
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(playlist.ApiKey, playlist.Name)
	var ids string
	err = row.Scan(&ids)
	if err != nil {
		return nil, err
	}
	list := strings.Split(ids, ",")
	if len(list) == 1 && utils.StringIsEmpty(list[0]) {
		list = make([]string, 0)
	}
	return list, nil
}

func (playlistsDB *PlaylistsDB) IsPlaylistPublic(playlist Playlist) bool {
	playlistsDB.rwLock.RLock()
	defer playlistsDB.rwLock.RUnlock()

	row := playlistsDB.db.QueryRow(fmt.Sprintf(
		"SELECT 1 FROM %s WHERE %s = ? AND %s = ? AND %s = ?",
		TablePlaylists, ColumnApikey.name, ColumnName.name, ColumnPublic.name),
		playlist.ApiKey, playlist.Name, true)

	var public bool
	err := row.Scan(&public)
	return err == nil && public
}

func (playlistsDB *PlaylistsDB) AddIdToPlaylist(playlistId PlaylistId) error {
	playlistsDB.rwLock.Lock()
	defer playlistsDB.rwLock.Unlock()

	ids, err := playlistsDB.getPlaylistIds(Playlist{
		ApiKey: playlistId.ApiKey, Name: playlistId.Name})
	if err != nil {
		return err
	}

	ids = append(ids, playlistId.Id)
	return playlistsDB.setPlaylistIds(PlaylistIds{
		playlistId.ApiKey, playlistId.Name, ids})
}

func (playlistsDB *PlaylistsDB) DeleteIdFromPlaylist(playlistId PlaylistId) error {
	playlistsDB.rwLock.Lock()
	defer playlistsDB.rwLock.Unlock()

	ids, err := playlistsDB.getPlaylistIds(Playlist{
		ApiKey: playlistId.ApiKey, Name: playlistId.Name})
	if err != nil {
		return err
	}

	index := -1
	for i, id := range ids {
		if id == playlistId.Id {
			index = i
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("id to delete not found")
	}

	newIds := ids[:index]
	newIds = append(newIds, ids[index+1:]...)
	return playlistsDB.setPlaylistIds(PlaylistIds{
		playlistId.ApiKey, playlistId.Name, newIds})
}

func (playlistsDB *PlaylistsDB) SetPlaylistIds(playlistIds PlaylistIds) error {
	playlistsDB.rwLock.Lock()
	defer playlistsDB.rwLock.Unlock()
	return playlistsDB.setPlaylistIds(playlistIds)
}

func (playlistsDB *PlaylistsDB) setPlaylistIds(playlistIds PlaylistIds) error {
	set := make(map[string]struct{})
	for _, id := range playlistIds.Ids {
		if _, ok := set[id]; ok {
			return fmt.Errorf("duplicate in ids")
		}
		set[id] = struct{}{}
	}

	_, err := playlistsDB.db.Exec(fmt.Sprintf(
		"UPDATE %s SET %s = ? WHERE %s = ? AND %s = ?",
		TablePlaylists, ColumnIds.name, ColumnApikey.name, ColumnName.name),
		strings.Join(playlistIds.Ids, ","), playlistIds.ApiKey, playlistIds.Name)
	return err
}
