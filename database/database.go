package database

import (
	"sync"
	"database/sql"

	"../utils"
)

var singletonLock sync.Mutex
var databaseInstance *Database

type Database struct {
	db *sql.DB

	UserDB          *UserDB
	PlaylistNamesDB *PlaylistNamesDB
	PlaylistsDB     *PlaylistsDB
	HistoriesDB     *HistoriesDB

	YoutubeDB *YoutubeDB
}

func GetDatabase() *Database {
	singletonLock.Lock()
	defer singletonLock.Unlock()

	if databaseInstance != nil {
		return databaseInstance
	}

	db, err := sql.Open("sqlite3", utils.DATADB)
	utils.Panic(err)

	rwLock := &sync.RWMutex{}

	userDB, err := newUserDB(db, rwLock)
	utils.Panic(err)

	playlistNamesDB, err := newPlaylistNamesDB(db, rwLock)
	utils.Panic(err)

	playlistsDB, err := newPlaylistsDB(db, rwLock)
	utils.Panic(err)

	historiesDB, err := newHistoriesDB(db, rwLock)
	utils.Panic(err)

	youtubeDB, err := newYoutubeDB()
	utils.Panic(err)

	databaseInstance = &Database{
		db,
		userDB,
		playlistNamesDB,
		playlistsDB,
		historiesDB,
		youtubeDB,
	}
	return databaseInstance
}

func (database *Database) SetRandomKey(key []byte) {
	database.YoutubeDB.randomKey = key
}

func (database *Database) SetYTApiKey(key string) {
	database.YoutubeDB.ytKey = key
}

func (database *Database) Close() error {
	return database.db.Close()
}
