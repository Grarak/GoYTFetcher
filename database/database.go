package database

import (
	"database/sql"
	"sync"

	"github.com/Grarak/GoYTFetcher/utils"
	_ "github.com/mattn/go-sqlite3"
)

var singletonLock sync.Mutex
var databaseInstance *Database

type Database struct {
	db *sql.DB

	UsersDB     *UsersDB
	PlaylistsDB *PlaylistsDB
	HistoriesDB *HistoriesDB

	YoutubeDB YouTubeDB
}

func GetDefaultDatabase() *Database {
	return GetDatabase("", nil, "")
}

func GetDatabase(host string, key []byte, ytKey string) *Database {
	singletonLock.Lock()
	defer singletonLock.Unlock()

	if databaseInstance != nil {
		return databaseInstance
	}

	db, err := sql.Open("sqlite3", utils.DATADB)
	utils.Panic(err)

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	utils.Panic(err)

	rwLock := &sync.RWMutex{}

	usersDB, err := newUsersDB(db, rwLock)
	utils.Panic(err)

	playlistsDB, err := newPlaylistsDB(db, rwLock)
	utils.Panic(err)

	historiesDB, err := newHistoriesDB(db, rwLock)
	utils.Panic(err)

	youtubeDB, err := newYoutubeDB(host, key, ytKey)
	utils.Panic(err)

	databaseInstance = &Database{
		db,
		usersDB,
		playlistsDB,
		historiesDB,
		youtubeDB,
	}
	return databaseInstance
}

func (database *Database) Close() error {
	return database.db.Close()
}
