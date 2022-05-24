package storage

import (
	"database/sql"
	"errors"
	"fmt"
)

type DataBaseStorage struct {
	DataSourceName string
	conn           *sql.DB
}

func (db *DataBaseStorage) init() {

	if len(db.DataSourceName) < 1 {
		return
	}

	db.conn, _ = sql.Open("postgres", db.DataSourceName)
}

func (db DataBaseStorage) ReadAll() ([]Metrics, error) {

	var metrics []Metrics

	if db.conn == nil {
		return metrics, errors.New("connection with database is not established")
	}

	return metrics, nil
}

func (db DataBaseStorage) WriteAll(metrics []Metrics) error {

	if db.conn == nil {
		return errors.New("connection with database is not established")
	}

	for _, metric := range metrics {
		fmt.Printf("write to db: %s\n", metric.ShotString())
	}

	return nil
}

func (db DataBaseStorage) CheckHealth() bool {
	return db.conn != nil
}
