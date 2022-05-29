package storage

import (
	"metrics-and-alerting/pkg/config"
)

type Storager interface {
	Update(metric Metric) error
	UpdateData(metrics []Metric) error
	Get(metric Metric) (Metric, error)
	GetData() []Metric

	Delete(metric Metric) error
	Reset() error

	Init(cfg config.Config) error
	CheckHealth() bool
	Destroy()
}
