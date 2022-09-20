package storage

import (
	"metrics-and-alerting/pkg/metric"
)

type Repository interface {
	Upsert(metric metric.Metric) error
	UpsertBatch(metrics []metric.Metric) error
	Get(metric metric.Metric) (metric.Metric, error)
	GetBatch() ([]metric.Metric, error)
	Delete(metric metric.Metric) error

	Flush() error
	Restore() error
	Close() error

	Health() bool
}
