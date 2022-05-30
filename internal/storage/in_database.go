package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"metrics-and-alerting/pkg/config"

	_ "github.com/lib/pq"
)

const (
	driverDB = "postgres"
)

const (
	queryChangeGauge = `INSERT INTO runtimeMetrics (rm_name,rm_type,rm_value) 
                        VALUES ($1,$2,$3) 
                        ON CONFLICT (rm_name) 
                        DO UPDATE
                        SET rm_name=$1,rm_type=$2,rm_value=$3;`
	queryChangeCounter = `INSERT INTO runtimeMetrics (rm_name,rm_type,rm_delta)
                          VALUES ($1,$2,$3)
                          ON CONFLICT(rm_name)
                          DO UPDATE
                          SET rm_delta=(SELECT rm_delta
                                     FROM runtimeMetrics
                                     WHERE rm_name=$1)+$3;`
)

type DataBaseStorage struct {
	dsn     string
	signKey []byte
}

func (dbStore DataBaseStorage) DB() (*sql.DB, error) {
	if dbStore.dsn == "" {
		return nil, fmt.Errorf("error create DB connection: %w", ErrorInvalidDSN)
	}

	return sql.Open(driverDB, dbStore.dsn)
}

func (dbStore DataBaseStorage) CreateTable() error {

	db, err := dbStore.DB()
	if err != nil {
		return fmt.Errorf("error create table: %w", ErrorFailedConnection)
	}
	defer db.Close()

	query := `CREATE TABLE IF NOT EXISTS runtimeMetrics (
              id     SERIAL,
		      rm_name   CHARACTER VARYING(50) PRIMARY KEY,
		      rm_type   CHARACTER VARYING(50),
		      rm_delta  BIGINT,
		      rm_value  DOUBLE PRECISION );`

	if _, err := db.Exec(query); err != nil {
		return err
	}
	return nil
}

// VerifySign - Проверка подписи метрики
func (dbStore DataBaseStorage) VerifySign(metric Metric) error {
	if len(dbStore.signKey) < 1 {
		return nil
	}

	hash, err := Sign(metric, dbStore.signKey)
	if err != nil {
		return err
	}

	if hash != metric.Hash {
		return ErrorSignFailed
	}
	return nil
}

func (dbStore *DataBaseStorage) Init(cfg config.Config) error {

	dbStore.dsn = cfg.DatabaseDSN
	dbStore.signKey = []byte(cfg.SecretKey)

	if err := dbStore.CreateTable(); err != nil {
		return fmt.Errorf("can not init DB: %w", err)
	}

	return nil
}

// Update Обновление значения метрики
func (dbStore DataBaseStorage) Update(metric Metric) error {

	if err := dbStore.VerifySign(metric); err != nil {
		return fmt.Errorf("error updating metric: %w", err)
	}

	db, err := dbStore.DB()
	if err != nil {
		return fmt.Errorf("error create table: %w", ErrorFailedConnection)
	}
	defer db.Close()

	var errExec error
	switch metric.MType {
	case GaugeType:
		if metric.Value == nil {
			return ErrorInvalidValue
		}

		_, errExec = db.Exec(queryChangeGauge, metric.ID, metric.MType, *metric.Value)

	case CounterType:
		if metric.Delta == nil {
			return ErrorInvalidValue
		}

		_, errExec = db.Exec(queryChangeCounter, metric.ID, metric.MType, *metric.Delta)
	}

	if errExec != nil {
		return fmt.Errorf("error updating metric: %w", errExec)
	}

	return nil
}

// UpdateData - Обновление всех метрик
func (dbStore DataBaseStorage) UpdateData(metrics []Metric) error {

	db, err := dbStore.DB()
	if err != nil {
		return ErrorFailedConnection
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error updating all metrics: %w", err)
	}
	defer tx.Rollback()

	stmtGauge, err := tx.Prepare(queryChangeGauge)
	if err != nil {
		return fmt.Errorf("error prepare statement 'gauge' : %w", err)
	}
	defer stmtGauge.Close()

	stmtCounter, err := tx.Prepare(queryChangeCounter)
	if err != nil {
		return fmt.Errorf("error prepare statement 'counter' type: %w", err)
	}
	defer stmtCounter.Close()

	for _, metric := range metrics {

		if metric.ID == "" {
			return fmt.Errorf("error updating metrics. Metric %s. %w", metric.ShotString(), ErrorInvalidID)
		}

		if err := dbStore.VerifySign(metric); err != nil {
			return fmt.Errorf("error updating metric: %w", err)
		}

		switch metric.MType {
		case GaugeType:
			if metric.Value == nil {
				return fmt.Errorf("error updating metrics. Metric %s. %w", metric.ShotString(), ErrorInvalidValue)
			}

			if _, err := stmtGauge.Exec(metric.ID, metric.MType, *metric.Value); err != nil {
				return fmt.Errorf("error updating metrics: %w", err)
			}

		case CounterType:
			if metric.Delta == nil {
				return fmt.Errorf("error updating metrics. Metric %s. %w", metric.ShotString(), ErrorInvalidValue)
			}

			if _, err := stmtCounter.Exec(metric.ID, metric.MType, *metric.Delta); err != nil {
				return fmt.Errorf("error updating metrics: %w", err)
			}

		default:
			return fmt.Errorf("error updating metrics. Metric %s. %w", metric.ShotString(), ErrorUnknownType)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error updating metrics. Commit error: %w", err)
	}

	return nil
}

// Get - Получение полность заполненной метрики
func (dbStore DataBaseStorage) Get(metric Metric) (Metric, error) {

	if len(metric.ID) < 1 {
		return Metric{}, fmt.Errorf("error get metric: %w", ErrorInvalidID)
	}

	if metric.MType != GaugeType && metric.MType != CounterType {
		return Metric{}, fmt.Errorf("error get metric: %w", ErrorUnknownType)
	}

	log.Printf("call Get metric: %s\n", metric.ShotString())

	db, err := dbStore.DB()
	if err != nil {
		return Metric{}, ErrorFailedConnection
	}
	defer db.Close()

	var (
		deltaNS sql.NullInt64
		valueNS sql.NullFloat64
	)

	query := `SELECT rm_delta, rm_value FROM runtimeMetrics 
              WHERE rm_name=$1 AND rm_type=$2;`
	rows := db.QueryRow(query, metric.ID, metric.MType)

	if err := rows.Scan(&deltaNS, &valueNS); err != nil {
		return Metric{}, fmt.Errorf("error get metric: %s: %w", err.Error(), ErrorNotFound)
	}

	err = rows.Err()
	if err != nil {
		return Metric{}, fmt.Errorf("error scan metric: %s: %w", err.Error(), ErrorNotFound)
	}

	if deltaNS.Valid {
		metric.Delta = &deltaNS.Int64
	}

	if valueNS.Valid {
		metric.Value = &valueNS.Float64
	}

	if len(dbStore.signKey) > 0 {
		if hash, err := Sign(metric, dbStore.signKey); err == nil {
			metric.Hash = hash
		} else {
			log.Printf("error sing metrinc: %v\n", err)
		}
	}

	return metric, nil
}

// GetData - Получение всех, полностью заполненных, метрик
func (dbStore DataBaseStorage) GetData() []Metric {

	db, err := dbStore.DB()
	if err != nil {
		log.Printf("%v\n", ErrorFailedConnection)
		return []Metric{}
	}
	defer db.Close()

	rows, err := db.Query("SELECT rm_name,rm_type,rm_delta,rm_value FROM runtimeMetrics;")
	if err != nil {
		log.Printf("%v\n", err)
		return []Metric{}
	}
	defer rows.Close()

	metrics := make([]Metric, 0)

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

		metric := Metric{
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
			log.Printf("%v\n", ErrorUnknownType)
			continue
		}

		metrics = append(metrics, metric)
	}

	err = rows.Err()
	if err != nil {
		log.Printf("error read metrics from DB: %v\n", err)
		return []Metric{}
	}

	if len(dbStore.signKey) > 0 {
		for idx := range metrics {
			if hash, err := Sign(metrics[idx], dbStore.signKey); err == nil {
				metrics[idx].Hash = hash
			} else {
				log.Printf("error sing metrinc: %v\n", err)
			}
		}
	}

	return metrics
}

// Delete - Удаление метрики
func (dbStore DataBaseStorage) Delete(metric Metric) error {

	db, err := dbStore.DB()
	if err != nil {
		return fmt.Errorf("error delete metric: %w", ErrorFailedConnection)
	}
	defer db.Close()

	query := "DELETE FROM runtimeMetrics WHERE rm_name=$1 AND rm_type=$2;"
	if _, err := db.Exec(query, metric.ID, metric.MType); err != nil {
		return fmt.Errorf("error delete metric: %w", err)
	}

	return nil
}

func (dbStore DataBaseStorage) Reset() error {
	db, err := dbStore.DB()
	if err != nil {
		return fmt.Errorf("error delete metric: %w", ErrorFailedConnection)
	}
	defer db.Close()

	if _, err := db.Exec("DELETE FROM runtimeMetrics;"); err != nil {
		return fmt.Errorf("error reset storage: %w", err)
	}

	return nil
}

func (dbStore DataBaseStorage) CheckHealth() bool {

	db, err := dbStore.DB()
	if err != nil {
		return false
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	errPing := db.PingContext(ctx)
	return errPing == nil
}

func (dbStore DataBaseStorage) Destroy() {
	log.Println("Destroy database storage... Goodbye :)")
}
