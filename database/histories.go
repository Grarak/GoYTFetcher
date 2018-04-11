package database

import (
	"database/sql"
	"time"
	"fmt"
	"encoding/json"
	"sync"
)

const TableHistories = "histories"

type History struct {
	ApiKey string    `json:"apikey"`
	Id     string    `json:"id"`
	Date   time.Time `json:"-"`
}

func NewHistory(data []byte) (History, error) {
	var history History
	err := json.Unmarshal(data, &history)
	return history, err
}

type HistoriesDB struct {
	db     *sql.DB
	rwLock *sync.RWMutex
}

func newHistoriesDB(db *sql.DB, rwLock *sync.RWMutex) (*HistoriesDB, error) {
	err := createTable(db, TableHistories, ColumnApikey, ColumnId, ColumnDate)
	if err != nil {
		return nil, err
	}

	return &HistoriesDB{db, rwLock}, nil
}

func (historiesDB *HistoriesDB) AddHistory(apiKey, id string) error {
	historiesDB.rwLock.Lock()
	defer historiesDB.rwLock.Unlock()

	row := historiesDB.db.QueryRow(fmt.Sprintf(
		"SELECT 1 FROM %s WHERE %s = '%s' AND %s = '%s'",
		TableHistories,
		ColumnApikey.name, apiKey,
		ColumnId.name, id))

	var exists bool
	row.Scan(&exists)

	if exists {
		_, err := historiesDB.db.Exec(fmt.Sprintf(
			"UPDATE %s SET %s = '%s' WHERE %s = '%s' AND %s = '%s'",
			TableHistories,
			ColumnDate.name, time.Now().Format(dateTimeFormat),
			ColumnApikey.name, apiKey,
			ColumnId.name, id))
		return err
	} else {
		_, err := historiesDB.db.Exec(fmt.Sprintf(
			"INSERT INTO %s (%s, %s, %s,) VALUES (?, ?, ?)",
			TableHistories, ColumnApikey.name, ColumnId.name,
			ColumnDate.name),
			apiKey, id, time.Now().Format(dateTimeFormat))
		return err
	}
}

func (historiesDB *HistoriesDB) GetHistory(apiKey string, page int) ([]string, error) {
	historiesDB.rwLock.RLock()
	defer historiesDB.rwLock.RUnlock()

	if page < 1 {
		page = 1
	}
	row, err := historiesDB.db.Query(fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s = '%s' "+
			"ORDER BY %s DESC "+
			"LIMIT 50 OFFSET %d",
		ColumnId.name, TableHistories, ColumnApikey.name, apiKey,
		ColumnDate.name, 50*(page-1)))
	if err != nil {
		return nil, err
	}
	defer row.Close()

	var links []string
	for row.Next() {
		var link string
		err = row.Scan(&link)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	return links, nil
}
