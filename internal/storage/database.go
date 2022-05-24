package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type DataBaseStorage struct {
	DataSourceName string
	conn           *sql.DB
}

func (db *DataBaseStorage) Connect() (*sql.DB, error) {

	if db.conn != nil {
		return db.conn, nil
	}

	if len(db.DataSourceName) < 1 {
		return nil, errors.New("invalid DSN")
	}

	conn, err := sql.Open("postgres", db.DataSourceName)
	if err != nil {
		return nil, err
	}

	db.conn = conn
	return db.conn, nil
}

func (db DataBaseStorage) Close() error {

	if db.conn == nil {
		return nil
	}

	return db.conn.Close()
}

func (db DataBaseStorage) ReadAll() ([]Metrics, error) {

	var metrics []Metrics

	_, err := db.Connect()
	if err != nil {
		return metrics, err
	}

	return metrics, nil
}

func (db DataBaseStorage) WriteAll(metrics []Metrics) error {

	_, err := db.Connect()
	if err != nil {
		return err
	}

	for _, metric := range metrics {
		fmt.Printf("write to db: %s\n", metric.ShotString())
	}

	return nil
}

func (db DataBaseStorage) CheckHealth() bool {

	_, err := db.Connect()
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := db.conn.PingContext(ctx); err != nil {
		return false
	}

	return true
}
