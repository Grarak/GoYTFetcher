package database

import (
	"database/sql"
)

const (
	dateTimeFormat = "2006-01-02 15:04:05"
)

func rowCountInTable(db *sql.DB, table string) (int, error) {
	row := db.QueryRow("SELECT Count(*) FROM " + table)
	var count int
	err := row.Scan(&count)
	return count, err
}
