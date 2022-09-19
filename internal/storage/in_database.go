package storage

//import (
//	"context"
//	"dbstore/sql"
//	"errors"
//	"fmt"
//	"log"
//	"time"
//
//	"metrics-and-alerting/pkg/metric"
//
//	sq "github.com/Masterminds/squirrel"
//	_ "github.com/lib/pq"
//)

/*
const (
	driverDB = "pgStorage"
)

const (
	queryChangeGauge = `INSERT INTO runtimeMetrics (name,type,value)
                        VALUES ($1,$2,$3)
                        ON CONFLICT (name)
                        DO UPDATE
                        SET name=$1,type=$2,value=$3;`
	queryChangeCounter = `INSERT INTO runtimeMetrics (name,type,delta)
                          VALUES ($1,$2,$3)
                          ON CONFLICT(name)
                          DO UPDATE
                          SET delta=(SELECT delta
                                     FROM runtimeMetrics
                                     WHERE name=$1)+$3;`
)

type DataBaseStorage struct {
	dsn      string
	signKey  []byte
	isVerify bool
}

func (dbStore DataBaseStorage) DB() (*sql.DB, error) {
	if dbStore.dsn == "" {
		return nil, fmt.Errorf("error create DB connection: %w", ErrInvalidDSN)
	}

	return sql.Open(driverDB, dbStore.dsn)
}

func (dbStore DataBaseStorage) CreateTable() error {

	db, err := dbStore.DB()
	if err != nil {
		return fmt.Errorf("error create table: %w", ErrFailedConnection)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("error close connectiond with DB after create table: %v\n", err)
		}
	}()

	query := `CREATE TABLE IF NOT EXISTS runtimeMetrics (
              id     SERIAL,
		      name   CHARACTER VARYING(50) PRIMARY KEY,
		      type   CHARACTER VARYING(50),
		      delta  BIGINT,
		      value  DOUBLE PRECISION );`

	if _, err := db.Exec(query); err != nil {
		return err
	}
	return nil
}

// VerifySign - Проверка подписи метрики
func (dbStore DataBaseStorage) VerifySign(metric metric.Metric) error {
	if !dbStore.isVerify {
		return nil
	}

	if len(dbStore.signKey) < 1 {
		return nil
	}

	hash, err := metric.Sign(metric, dbStore.signKey)
	if err != nil {
		return err
	}

	if hash != metric.Hash {
		return ErrSignFailed
	}
	return nil
}

func (dbStore *DataBaseStorage) Init(cfg config.Config) error {

	dbStore.dsn = cfg.DatabaseDSN
	dbStore.signKey = []byte(cfg.SecretKey)
	dbStore.isVerify = cfg.VerifyOnUpdate

	if err := dbStore.CreateTable(); err != nil {
		return fmt.Errorf("can not init DB: %w", err)
	}

	return nil
}

// Upsert Обновление значения метрики
func (dbStore DataBaseStorage) Upsert(metric metric.Metric) error {

	if err := dbStore.VerifySign(metric); err != nil {
		return fmt.Errorf("error updating metric: %w", err)
	}

	db, err := dbStore.DB()
	if err != nil {
		return fmt.Errorf("error update metric: %w", ErrFailedConnection)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("error close connectiond with DB after Update: %v\n", err)
		}
	}()

	var errExec error
	switch metric.MType {
	case metric.GaugeType:
		if metric.Value == nil {
			return ErrInvalidValue
		}

		_, errExec = db.Exec(queryChangeGauge, metric.ID, metric.MType, *metric.Value)

	case metric.CounterType:
		if metric.Delta == nil {
			return ErrInvalidValue
		}

		_, errExec = db.Exec(queryChangeCounter, metric.ID, metric.MType, *metric.Delta)
	}

	if errExec != nil {
		return fmt.Errorf("error updating metric: %w", errExec)
	}

	return nil
}

// UpsertData - Обновление всех метрик
func (dbStore DataBaseStorage) UpsertData(metrics []metric.Metric) error {

	db, err := dbStore.DB()
	if err != nil {
		return ErrFailedConnection
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("error close connectiond with DB after UpdateData: %v\n", err)
		}
	}()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error updating all metrics: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			if !errors.Is(err, sql.ErrTxDone) {
				log.Printf("error rollback: %v\n", err)
			}
		}
	}()

	stmtGauge, err := tx.Prepare(queryChangeGauge)
	if err != nil {
		return fmt.Errorf("error prepare statement 'gauge' : %w", err)
	}
	defer func() {
		if err := stmtGauge.Close(); err != nil {
			log.Printf("error close gauge statement in UpdateData: %v\n", err)
		}
	}()

	stmtCounter, err := tx.Prepare(queryChangeCounter)
	if err != nil {
		return fmt.Errorf("error prepare statement 'counter' type: %w", err)
	}
	defer func() {
		if err := stmtCounter.Close(); err != nil {
			log.Printf("error close counter statement in UpdateData: %v\n", err)
		}
	}()

	for _, metric := range metrics {

		if metric.ID == "" {
			return fmt.Errorf("error updating metrics. Metric %s. %w", metric.ShotString(), ErrInvalidID)
		}

		if err := dbStore.VerifySign(metric); err != nil {
			return fmt.Errorf("error updating metric: %w", err)
		}

		switch metric.MType {
		case metric.GaugeType:
			if metric.Value == nil {
				return fmt.Errorf("error updating metrics. Metric %s. %w", metric.ShotString(), ErrInvalidValue)
			}

			if _, err := stmtGauge.Exec(metric.ID, metric.MType, *metric.Value); err != nil {
				return fmt.Errorf("error updating metrics: %w", err)
			}

		case metric.CounterType:
			if metric.Delta == nil {
				return fmt.Errorf("error updating metrics. Metric %s. %w", metric.ShotString(), ErrInvalidValue)
			}

			if _, err := stmtCounter.Exec(metric.ID, metric.MType, *metric.Delta); err != nil {
				return fmt.Errorf("error updating metrics: %w", err)
			}

		default:
			return fmt.Errorf("error updating metrics. Metric %s. %w", metric.ShotString(), ErrUnknownType)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error updating metrics. Commit error: %w", err)
	}

	return nil
}

// Get - Получение полность заполненной метрики
func (dbStore DataBaseStorage) Get(metric metric.Metric) (metric.Metric, error) {

	if len(metric.ID) < 1 {
		return metric.Metric{}, fmt.Errorf("error get metric: %w", ErrInvalidID)
	}

	if metric.MType != metric.GaugeType && metric.MType != metric.CounterType {
		return metric.Metric{}, fmt.Errorf("error get metric: %w", ErrUnknownType)
	}

	db, err := dbStore.DB()
	if err != nil {
		return metric.Metric{}, ErrFailedConnection
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("error close connectiond with DB after Get: %v\n", err)
		}
	}()

	var (
		deltaNS sql.NullInt64
		valueNS sql.NullFloat64
	)

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	query := psql.Select("delta", "value").
		From("runtimeMetrics").
		Where(sq.And{
			sq.Eq{"name": metric.ID},
			sq.Eq{"type": metric.MType}})

	rows := query.RunWith(db).QueryRow()

	if err := rows.Scan(&deltaNS, &valueNS); err != nil {
		return metric.Metric{}, fmt.Errorf("error get metric: %s: %w", err.Error(), ErrNotFound)
	}

	if deltaNS.Valid {
		metric.Delta = &deltaNS.Int64
	}

	if valueNS.Valid {
		metric.Value = &valueNS.Float64
	}

	if len(dbStore.signKey) > 0 {
		if hash, err := metric.Sign(metric, dbStore.signKey); err == nil {
			metric.Hash = hash
		}
	}

	return metric, nil
}

// GetData - Получение всех, полностью заполненных, метрик
func (dbStore DataBaseStorage) GetData() []metric.Metric {

	db, err := dbStore.DB()
	if err != nil {
		log.Printf("%v\n", ErrFailedConnection)
		return []metric.Metric{}
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("error close connectiond with DB after GetData: %v\n", err)
		}
	}()

	rows, err := db.Query("SELECT name,type,delta,value FROM runtimeMetrics;")
	if err != nil {
		log.Printf("%v\n", err)
		return []metric.Metric{}
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("error close Rows in GetData: %v\n", err)
		}
	}()

	metrics := make([]metric.Metric, 0)

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

		metric := metric.Metric{
			ID:    idNS.String,
			MType: mtypeNS.String,
		}

		switch metric.MType {
		case metric.GaugeType:
			if valueNS.Valid {
				metric.Value = &valueNS.Float64
			}

		case metric.CounterType:
			if deltaNS.Valid {
				metric.Delta = &deltaNS.Int64
			}

		default:
			log.Printf("%v\n", ErrUnknownType)
			continue
		}

		metrics = append(metrics, metric)
	}

	err = rows.Err()
	if err != nil {
		log.Printf("error read metrics from DB: %v\n", err)
		return []metric.Metric{}
	}

	if len(dbStore.signKey) > 0 {
		for idx := range metrics {
			if hash, err := metric.Sign(metrics[idx], dbStore.signKey); err == nil {
				metrics[idx].Hash = hash
			}
		}
	}

	return metrics
}

// Delete - Удаление метрики
func (dbStore DataBaseStorage) Delete(metric metric.Metric) error {

	db, err := dbStore.DB()
	if err != nil {
		return fmt.Errorf("error delete metric: %w", ErrFailedConnection)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("error close connectiond with DB after Delete: %v\n", err)
		}
	}()

	query := "DELETE FROM runtimeMetrics WHERE name=$1 AND type=$2;"
	if _, err := db.Exec(query, metric.ID, metric.MType); err != nil {
		return fmt.Errorf("error delete metric: %w", err)
	}

	return nil
}

func (dbStore DataBaseStorage) Reset() error {
	db, err := dbStore.DB()
	if err != nil {
		return fmt.Errorf("error delete metric: %w", ErrFailedConnection)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("error close connectiond with DB after Reset: %v\n", err)
		}
	}()

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
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("error close connectiond with DB after CheckHealth: %v\n", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	errPing := db.PingContext(ctx)
	return errPing == nil
}

func (dbStore DataBaseStorage) Destroy() {
	log.Println("Destroy dbstore storage... Goodbye :)")
}
*/
