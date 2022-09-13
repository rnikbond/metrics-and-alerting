package storage

import (
	"metrics-and-alerting/pkg/metric"
)

type Repository interface {
	Set(metric metric.Metric) error
	Upsert(metric metric.Metric) error
	UpsertSlice(metrics []metric.Metric) error

	Get(metric metric.Metric) (metric.Metric, error)
	GetSlice() ([]metric.Metric, error)

	Delete(metric metric.Metric) error

	String() string

	CheckHealth() bool
	Close() error
}
