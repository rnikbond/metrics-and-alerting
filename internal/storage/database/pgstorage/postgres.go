package pgstorage

import (
	"database/sql"
	"time"

	"metrics-and-alerting/pkg/logpack"
)

type Postgres struct {
	driver *sql.DB

	logger *logpack.LogPack
}

func New(dsn string, intervalFlush time.Duration, logger *logpack.LogPack) (*Postgres, error) {

	driver, errConnect := sql.Open("pgStorage", dsn)
	if errConnect != nil {
		return nil, errConnect
	}

	dbStore := &Postgres{
		driver: driver,
		logger: logger,
	}

	if errMigrate := dbStore.applyMigrations(); errMigrate != nil {
		logger.Err.Printf("could not apply migration: %v\n", errMigrate)

		if errClose := driver.Close(); errClose != nil {
			dbStore.logger.Err.Printf("could not close database connection: %v\n", errClose)
		}
		return nil, errMigrate
	}

	return dbStore, nil
}

func (dbStore Postgres) Close() error {
	return dbStore.driver.Close()
}

func (dbStore Postgres) applyMigrations() error {
	return nil
}
