package database

import (
	"database/sql"
	"fmt"
	"time"
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

func insertColumns(db *sql.DB, table string, columns ...column) error {
	for _, column := range columns {
		if tableContainsColumn(db, table, column.name) {
			continue
		}
		_, err := db.Exec(fmt.Sprintf(
			"ALTER TABLE %s ADD %s %s",
			table, column.name, column.dataType))
		if err != nil {
			panic(err)
		}
	}
	return nil
}

func tableContainsColumn(db *sql.DB, table, column string) bool {
	row, err := db.Query("PRAGMA table_info(" + table + ")")
	defer row.Close()
	if err != nil {
		return false
	}

	c, err := row.Columns()
	for row.Next() {
		if err != nil {
			return false
		}

		values := make([]interface{}, len(c))
		for i := range values {
			values[i] = new(MetalScanner)
		}

		err = row.Scan(values...)
		if err != nil {
			return false
		}

		if string(values[1].(*MetalScanner).value.([]byte)) == column {
			return true
		}
	}

	return false
}

type MetalScanner struct {
	valid bool
	value interface{}
}

func (scanner *MetalScanner) getBytes(src interface{}) []byte {
	if a, ok := src.([]uint8); ok {
		return a
	}
	return nil
}

func (scanner *MetalScanner) Scan(src interface{}) error {
	switch src.(type) {
	case int64:
		if value, ok := src.(int64); ok {
			scanner.value = value
			scanner.valid = true
		}
	case float64:
		if value, ok := src.(float64); ok {
			scanner.value = value
			scanner.valid = true
		}
	case bool:
		if value, ok := src.(bool); ok {
			scanner.value = value
			scanner.valid = true
		}
	case string:
		value := scanner.getBytes(src)
		scanner.value = string(value)
		scanner.valid = true
	case []byte:
		value := scanner.getBytes(src)
		scanner.value = value
		scanner.valid = true
	case time.Time:
		if value, ok := src.(time.Time); ok {
			scanner.value = value
			scanner.valid = true
		}
	case nil:
		scanner.value = nil
		scanner.valid = true
	}
	return nil
}
