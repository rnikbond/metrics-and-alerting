package dbstore

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"

	"metrics-and-alerting/internal/storage/memstore"
	"metrics-and-alerting/pkg/logpack"
	metricPkg "metrics-and-alerting/pkg/metric"
)

const (
	queryChangeGauge = `INSERT INTO runtimeMetrics (name,type,value)
                         VALUES ($1,$2,$3)
                         ON CONFLICT (name)
                         DO UPDATE
                         SET name=$1,type=$2,value=$3;`

	queryChangeCounter = `INSERT INTO runtimeMetrics (name,type,delta)
                           VALUES ($1,$2,$3)
                           ON CONFLICT (name)
                           DO UPDATE
                           SET name=$1,type=$2,delta=$3;`

	queryGetMetrics = `SELECT name,type,delta, value
                       FROM runtimeMetrics`
)

type Storage struct {
	db     *sql.DB
	logger *logpack.LogPack
	memory *memstore.Storage
}

func New(dsn string, logger *logpack.LogPack) (*Storage, error) {

	driver, errConnect := sql.Open("postgres", dsn)
	if errConnect != nil {
		logger.Err.Printf("Could not connect to database: %v\n", errConnect)
		return nil, errConnect
	}

	dbStore := &Storage{
		db:     driver,
		logger: logger,
		memory: memstore.New(),
	}

	if errMigrate := dbStore.applyMigrations(); errMigrate != nil {
		logger.Err.Printf("could not apply migration: %v\n", errMigrate)

		if errClose := driver.Close(); errClose != nil {
			logger.Err.Printf("could not close database connection: %v\n", errClose)
		}
	}

	if errRestore := dbStore.Restore(); errRestore != nil {
		logger.Err.Printf("could not restore metrics from database: %v\n", errRestore)
	}

	return dbStore, nil
}

func (store *Storage) Upsert(metric metricPkg.Metric) error {

	return store.memory.Upsert(metric)
}

func (store *Storage) UpsertBatch(metrics []metricPkg.Metric) error {

	return store.memory.UpsertBatch(metrics)
}

func (store Storage) Get(metric metricPkg.Metric) (metricPkg.Metric, error) {

	return store.memory.Get(metric)
}

func (store Storage) GetBatch() ([]metricPkg.Metric, error) {

	return store.memory.GetBatch()
}

func (store *Storage) Delete(metric metricPkg.Metric) error {

	if err := store.memory.Delete(metric); err != nil {
		return err
	}

	query := `DELETE FROM runtimeMetrics WHERE name=$1 AND type=$2;`
	if _, err := store.db.Exec(query, metric.ID, metric.MType); err != nil {
		return fmt.Errorf("could not delete metric from database: %w", err)
	}

	return nil
}

func (store Storage) Flush() error {

	tx, err := store.db.Begin()
	if err != nil {
		return fmt.Errorf("could not flush metrics to database: %w", err)
	}
	defer func() {
		if errRollBack := tx.Rollback(); errRollBack != nil {
			if !errors.Is(errRollBack, sql.ErrTxDone) {
				store.logger.Err.Printf("error rollback: %v\n", errRollBack)
			}
		}
	}()

	stmtGauge, err := tx.Prepare(queryChangeGauge)
	if err != nil {
		return fmt.Errorf("error prepare statement 'gauge' : %w", err)
	}
	defer func() {
		if errClose := stmtGauge.Close(); errClose != nil {
			store.logger.Err.Printf("error close gauge statement: %v\n", errClose)
		}
	}()

	stmtCounter, err := tx.Prepare(queryChangeCounter)
	if err != nil {
		return fmt.Errorf("error prepare statement 'counter': %w", err)
	}
	defer func() {
		if errClose := stmtCounter.Close(); errClose != nil {
			store.logger.Err.Printf("error close counter statement: %v\n", errClose)
		}
	}()

	metrics, err := store.memory.GetBatch()
	if err != nil {
		return fmt.Errorf("could not flush metrics to database: %w", err)
	}

	for _, metric := range metrics {

		var errExec error

		switch metric.MType {
		case metricPkg.GaugeType:
			if metric.Value == nil {
				store.logger.Err.Printf("could not flush metric without value: %s\n", metric.ShotString())
				continue
			}

			_, errExec = stmtGauge.Exec(metric.ID, metric.MType, *metric.Value)

		case metricPkg.CounterType:
			if metric.Delta == nil {
				store.logger.Err.Printf("could not flush metric without delta: %s\n", metric.ShotString())
				continue
			}

			_, errExec = stmtCounter.Exec(metric.ID, metric.MType, *metric.Delta)

		default:
			store.logger.Err.Printf("could not flush metric with unknown type: %s\n", metric.ShotString())
		}

		if errExec != nil {
			errExec = fmt.Errorf("could not flush metric: %w", errExec)
			return errExec
		}
	}

	if errCommit := tx.Commit(); errCommit != nil {
		errCommit = fmt.Errorf("could not commit flush transaction: %w", errCommit)
		store.logger.Err.Println(errCommit)
		return errCommit
	}

	return nil
}

func (store *Storage) Restore() error {

	rows, errQuery := store.db.Query(queryGetMetrics)
	if errQuery != nil {
		return fmt.Errorf("could not load metrics from database: %w", errQuery)
	}

	defer func() {
		if err := rows.Close(); err != nil {
			store.logger.Err.Printf("could not close rows: %v\n", err)
		}
	}()

	for rows.Next() {

		var (
			id    sql.NullString
			mtype sql.NullString
			delta sql.NullInt64
			value sql.NullFloat64
		)

		if err := rows.Scan(&id, &mtype, &delta, &value); err != nil {
			store.logger.Err.Printf("error scan: %v\n", err)
			continue
		}

		metric, err := metricPkg.CreateMetric(mtype.String, id.String)
		if err != nil {
			store.logger.Err.Printf("could not restore metric: [type: %s], [id: %s]\n", mtype.String, id.String)
			continue
		}

		switch metric.MType {
		case metricPkg.GaugeType:
			if value.Valid {
				metric.Value = &value.Float64
			}
		case metricPkg.CounterType:
			if delta.Valid {
				metric.Delta = &delta.Int64
			}
		}

		if errMem := store.memory.Upsert(metric); errMem != nil {
			store.logger.Err.Printf("could not restore metric: %s. %v\n", metric.ShotString(), errMem)
		}
	}

	if err := rows.Err(); err != nil {
		store.logger.Err.Printf("could not restore metric: %v\n", err)
		return err
	}

	return nil
}

func (store *Storage) Close() error {
	return store.db.Close()

}

func (store Storage) Health() bool {

	if store.db == nil {
		store.logger.Err.Println("database driver is nil")
		return false
	}

	err := store.db.Ping()
	if err != nil {
		store.logger.Err.Printf("ping driver returned error: %v\n", err)
		return false
	}

	return true
}

func (store Storage) applyMigrations() error {

	query := `CREATE TABLE IF NOT EXISTS runtimeMetrics (
              id     SERIAL,
		      name   CHARACTER VARYING(50) PRIMARY KEY,
		      type   CHARACTER VARYING(50),
		      delta  BIGINT,
		      value  DOUBLE PRECISION );`

	if _, err := store.db.Exec(query); err != nil {
		return err
	}

	return nil
}
