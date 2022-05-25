package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type DataBaseStorage struct {
	DataSourceName string
	conn           *sql.DB
}

func (db *DataBaseStorage) CreateTables() error {

	if db.conn == nil {
		return errors.New("not connection to database")
	}

	_, err := db.conn.Exec(
		"CREATE TABLE IF NOT EXISTS metricsData " +
			"(ID CHARACTER VARYING(50) PRIMARY KEY," +
			"MTYPE CHARACTER VARYING(50)," +
			"MEAN CHARACTER VARYING(50)" +
			");")
	if err != nil {
		return err
	}

	return nil
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
	} else {
		log.Println("success connect to database")
	}

	db.conn = conn

	if err := db.CreateTables(); err != nil {
		log.Printf("error create table: %v", err)
	}

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

	conn, err := db.Connect()
	if err != nil {
		return metrics, err
	}
	defer conn.Close()

	rows, err := conn.Query("SELECT * FROM data;")
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var (
			id    string
			mtype string
			value string
		)

		if err := rows.Scan(&id, &mtype, &value); err != nil {
			log.Printf("error scan: %v\n", err)
			continue
		}

		m := NewMetric(mtype, id, value)
		metrics = append(metrics, m)
		fmt.Printf("read: %s\n", m.ShotString())
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return metrics, nil
}

func (db DataBaseStorage) WriteAll(metrics []Metrics) error {

	conn, err := db.Connect()
	if err != nil {
		return err
	}

	defer conn.Close()

	for _, metric := range metrics {
		query := `INSERT INTO data
				  VALUES 
                      ($1,$2,$3)
                  ON CONFLICT(ID)
                  DO UPDATE SET 
                         MTYPE=$2,MEAN=$3`

		_, err := conn.Exec(query, metric.ID, metric.MType, metric.StringValue())
		if err != nil {
			log.Printf("error insert: %s\n", err.Error())
		}

		fmt.Printf("write: %s\n", metric.ShotString())

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
