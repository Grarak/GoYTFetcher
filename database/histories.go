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
	cmd := newTableBuilder(TableHistories).
		addForeignKey(ForeignKeyApikey).
		addPrimaryKey(ColumnId).
		addColumn(ColumnDate).build()

	_, err := db.Exec(cmd)
	if err != nil {
		return nil, err
	}

	return &HistoriesDB{db, rwLock}, nil
}

func (historiesDB *HistoriesDB) AddHistory(apiKey, id string) error {
	historiesDB.rwLock.Lock()
	defer historiesDB.rwLock.Unlock()

	history, err := historiesDB.getHistory(apiKey)
	if err != nil {
		return err
	}
	for i := 50; i < len(history); i++ {
		_, err := historiesDB.db.Exec(fmt.Sprintf(
			"DELETE FROM %s WHERE %s = ? AND %s = ?",
			TableHistories, ColumnApikey.name, ColumnId.name), apiKey, history[i])
		if err != nil {
			return err
		}
	}

	_, err = historiesDB.db.Exec(fmt.Sprintf(
		"INSERT OR REPLACE INTO %s (%s, %s, %s) VALUES (?, ?, ?)",
		TableHistories, ColumnApikey.name, ColumnId.name,
		ColumnDate.name),
		apiKey, id, time.Now().Format(dateTimeFormat))
	return err
}

func (historiesDB *HistoriesDB) GetHistory(apiKey string) ([]string, error) {
	historiesDB.rwLock.RLock()
	defer historiesDB.rwLock.RUnlock()
	return historiesDB.getHistory(apiKey)
}

func (historiesDB *HistoriesDB) getHistory(apiKey string) ([]string, error) {
	stmt, err := historiesDB.db.Prepare(fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s = ? "+
			"ORDER BY %s DESC",
		ColumnId.name, TableHistories, ColumnApikey.name,
		ColumnDate.name))
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(apiKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := make([]string, 0)
	for rows.Next() {
		var link string
		err = rows.Scan(&link)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, nil
}
