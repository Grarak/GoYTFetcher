package database

import (
	"database/sql"
	"fmt"

	"encoding/json"

	"../utils"
	"sync"
)

const TablePlaylistNames = "playlistnames"

type PlaylistName struct {
	ApiKey string `json:"apikey,omitempty"`
	Name   string `json:"name"`
	Public bool   `json:"public"`
}

func NewPlayListName(data []byte) (PlaylistName, error) {
	var name PlaylistName
	err := json.Unmarshal(data, &name)
	return name, err
}

type PlaylistNamesDB struct {
	db     *sql.DB
	rwLock *sync.RWMutex
}

func newPlaylistNamesDB(db *sql.DB, rwLock *sync.RWMutex) (*PlaylistNamesDB, error) {
	cmd := newTableBuilder(TablePlaylistNames).
		addForeignKey(ForeignKeyApikey).
		addPrimaryKey(ColumnName).
		addColumn(ColumnPublic).build()

	_, err := db.Exec(cmd)
	if err != nil {
		return nil, err
	}

	return &PlaylistNamesDB{db, rwLock}, nil
}

func (playlistNamesDB *PlaylistNamesDB) ListPlaylistNames(apiKey string, publicOnly bool) ([]PlaylistName, error) {
	playlistNamesDB.rwLock.RLock()
	defer playlistNamesDB.rwLock.RUnlock()

	cmd := fmt.Sprintf(
		"SELECT %s,%s FROM %s WHERE %s = '%s'",
		ColumnName.name, ColumnPublic.name, TablePlaylistNames, ColumnApikey.name, apiKey)

	if publicOnly {
		cmd += fmt.Sprintf(" AND %s = 1", ColumnPublic.name)
	}

	row, err := playlistNamesDB.db.Query(cmd)
	if err != nil {
		return nil, err
	}
	defer row.Close()

	list := make([]PlaylistName, 0)
	for row.Next() {
		var name string
		var public bool
		err := row.Scan(&name, &public)
		if err != nil {
			return nil, err
		}
		list = append(list, PlaylistName{Name: name, Public: public})
	}

	return list, nil
}

func (playlistNamesDB *PlaylistNamesDB) CreatePlaylistName(playlistName PlaylistName) error {
	playlistNamesDB.rwLock.Lock()
	defer playlistNamesDB.rwLock.Unlock()

	if utils.StringIsEmpty(playlistName.Name) {
		return utils.Error("Name is empty")
	}

	row := playlistNamesDB.db.QueryRow(fmt.Sprintf(
		"SELECT 1 FROM %s WHERE %s = '%s' AND %s = '%s'",
		TablePlaylistNames, ColumnApikey.name, playlistName.ApiKey,
		ColumnName.name, playlistName.Name))

	var exists bool
	row.Scan(&exists)
	if exists {
		return utils.Error(playlistName.Name + " already exists")
	}

	_, err := playlistNamesDB.db.Exec(fmt.Sprintf(
		"INSERT INTO %s (%s, %s, %s) VALUES (?, ?, ?)",
		TablePlaylistNames, ColumnApikey.name, ColumnName.name, ColumnPublic.name),
		playlistName.ApiKey, playlistName.Name, playlistName.Public)
	return err
}

func (playlistNamesDB *PlaylistNamesDB) SetPlaylistNamePublic(playlistName PlaylistName) error {
	playlistNamesDB.rwLock.Lock()
	defer playlistNamesDB.rwLock.Unlock()

	var public int
	if playlistName.Public {
		public = 1
	}

	_, err := playlistNamesDB.db.Exec(fmt.Sprintf(
		"UPDATE %s SET %s = %d WHERE %s = '%s' AND %s = '%s'",
		TablePlaylistNames, ColumnPublic.name, public,
		ColumnApikey.name, playlistName.ApiKey, ColumnName.name, playlistName.Name))
	return err
}

func (playlistNamesDB *PlaylistNamesDB) DeletePlaylistName(playlistName PlaylistName) error {
	playlistNamesDB.rwLock.Lock()
	defer playlistNamesDB.rwLock.Unlock()

	_, err := playlistNamesDB.db.Exec(fmt.Sprintf(
		"DELETE FROM %s WHERE %s = '%s' AND %s = '%s'",
		TablePlaylistNames,
		ColumnApikey.name, playlistName.ApiKey,
		ColumnName.name, playlistName.Name))
	return err
}
