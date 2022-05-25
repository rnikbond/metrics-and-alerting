package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type DataBaseStorage struct {
	Driver *sql.DB
}

func (dbStore *DataBaseStorage) CreateTables() error {

	if dbStore.Driver == nil {
		return ErrorDatabaseDriver
	}

	_, err := dbStore.Driver.Exec(
		"CREATE TABLE IF NOT EXISTS metricsData " +
			"(id     CHARACTER VARYING(50) PRIMARY KEY," +
			" mtype  CHARACTER VARYING(50)," +
			" delta  INTEGER," +
			" val    DOUBLE PRECISION);")
	if err != nil {
		return err
	}

	return nil
}

func (dbStore DataBaseStorage) ReadAll() ([]Metrics, error) {

	if dbStore.Driver == nil {
		return nil, ErrorDatabaseDriver
	}

	rows, err := dbStore.Driver.Query("SELECT * FROM metricsData;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metrics := make([]Metrics, 0)

	for rows.Next() {
		var (
			idNS    sql.NullString
			mtypeNS sql.NullString
			deltaNS sql.NullInt64
			valueNS sql.NullFloat64
		)

		if err := rows.Scan(&idNS, &mtypeNS, &deltaNS, &valueNS); err != nil {
			log.Printf("error scan: %v\n", err)
			continue
		}

		if !idNS.Valid {
			log.Printf("error read 'id' - is not valid.\n")
			continue
		}

		if !mtypeNS.Valid {
			log.Printf("error read 'mtype' - is not valid.\n")
			continue
		}

		metric := Metrics{
			ID:    idNS.String,
			MType: mtypeNS.String,
		}

		switch metric.MType {
		case GaugeType:
			if valueNS.Valid {
				metric.Value = &valueNS.Float64
			}

		case CounterType:
			if deltaNS.Valid {
				metric.Delta = &deltaNS.Int64
			}

		default:
			log.Printf("error unknown 'mtype': %s\n", metric.MType)
			continue
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

func (dbStore DataBaseStorage) WriteAll(metrics []Metrics) error {

	if dbStore.Driver == nil {
		return ErrorDatabaseDriver
	}

	query := `INSERT INTO metricsData 
			  VALUES 
				($1,$2,$3,$4) 
              ON CONFLICT(id) DO UPDATE SET
              mtype=$2,delta=$3,val=$4`

	fmt.Println("write ...")

	for _, metric := range metrics {

		var (
			deltaNS sql.NullInt64
			valueNS sql.NullFloat64
		)

		switch metric.MType {
		case GaugeType:
			if metric.Value == nil {
				log.Printf("error write metric without value: %s\n", metric.StringValue())
				continue
			}

			valueNS.Valid = true
			valueNS.Float64 = *metric.Value

		case CounterType:
			if metric.Delta == nil {
				log.Printf("error write metric without delta: %s\n", metric.StringValue())
				continue
			}

			deltaNS.Valid = true
			deltaNS.Int64 = *metric.Delta

		default:
			log.Printf("error write metric with unknown type: %s\n", metric.StringValue())
			continue
		}

		if _, err := dbStore.Driver.Exec(query, metric.ID, metric.MType, deltaNS, valueNS); err != nil {
			log.Printf("error insert or update: %s\n", err.Error())
		} else {
			fmt.Printf("success write: %s/%s/%s\n", metric.ID, metric.MType, metric.StringValue())
		}

	}

	return nil
}

func (dbStore DataBaseStorage) Ping() bool {

	if dbStore.Driver == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := dbStore.Driver.PingContext(ctx)
	return err == nil
}
