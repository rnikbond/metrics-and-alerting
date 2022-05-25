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
			"(id     CHARACTER VARYING(50) PRIMARY KEY," +
			" mtype  CHARACTER VARYING(50)," +
			" delta  INTEGER," +
			" value  DOUBLE PRECISION);")
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

	rows, err := conn.Query("SELECT * FROM metricsData;")
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {

		metric := Metrics{}

		var delta sql.NullInt64
		var value sql.NullFloat64

		if err := rows.Scan(&metric.ID, &metric.MType, &delta, &value); err != nil {
			log.Printf("error scan: %v\n", err)
			continue
		}

		switch metric.MType {
		case GaugeType:
			if value.Valid {
				v, err := value.Value()
				if err != nil {
					continue
				}
				vv, ok := v.(float64)
				if ok {
					metric.Value = &vv
				}
			}

		case CounterType:
			if delta.Valid {
				v, err := delta.Value()
				if err != nil {
					continue
				}
				vv, ok := v.(int64)
				if ok {
					metric.Delta = &vv
				}
			}
		}

		metrics = append(metrics, metric)
		fmt.Printf("read: %s\n", metric.ShotString())
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

		switch metric.MType {
		case GaugeType:
			_, err := conn.Exec(
				"INSERT INTO metricsData (id, mtype, value) VALUES ($1,$2,$3) "+
					"ON CONFLICT(id) DO UPDATE SET "+
					"mtype=$2,value=$3",
				metric.ID, metric.MType, *metric.Value)
			if err != nil {
				log.Printf("error insert or update: %s\n", err.Error())
			} else {
				fmt.Printf("success write: %s/%s/%s\n", metric.ID, metric.MType, metric.StringValue())
			}
		case CounterType:
			_, err := conn.Exec(
				"INSERT INTO metricsData (id, mtype, delta) VALUES ($1,$2,$3) "+
					"ON CONFLICT(id) DO UPDATE SET "+
					"mtype=$2,delta=$3",
				metric.ID, metric.MType, *metric.Delta)
			if err != nil {
				log.Printf("error insert or update: %s\n", err.Error())
			} else {
				fmt.Printf("success write: %s/%s/%s\n", metric.ID, metric.MType, metric.StringValue())
			}
		}
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
