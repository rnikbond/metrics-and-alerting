package server

import (
	"fmt"

	storage2 "metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/errs"
	"metrics-and-alerting/pkg/metric"
)

type OptionsManager func(*MetricsManager)

type MetricsManager struct {
	storage storage2.Repository
	signKey []byte
}

func NewMetricsManager(storage storage2.Repository, opts ...OptionsManager) *MetricsManager {

	manager := &MetricsManager{
		storage: storage,
	}

	for _, opt := range opts {
		opt(manager)
	}

	return manager
}

func WithSignKey(signKey []byte) OptionsManager {
	return func(manager *MetricsManager) {
		manager.signKey = signKey
	}
}

// VerifySign - Проверка подписи метрики
func (manager MetricsManager) VerifySign(metric metric.Metric) error {
	if len(manager.signKey) == 0 {
		return nil
	}

	hash, err := metric.Sign(manager.signKey)
	if err != nil {
		return err
	}

	if hash != metric.Hash {

		fmt.Printf("wait hash: %s\n", hash)
		fmt.Printf("metric hash: %s\n", metric.Hash)

		return errs.ErrSignFailed
	}

	return nil
}

func (manager MetricsManager) Set(metric metric.Metric) error {

	if err := manager.VerifySign(metric); err != nil {
		return fmt.Errorf("could not set metric: %w", err)
	}

	return manager.storage.Set(metric)
}

func (manager MetricsManager) Upsert(metric metric.Metric) error {

	if err := manager.VerifySign(metric); err != nil {
		return fmt.Errorf("could not upsert metric: %w", err)
	}

	return manager.storage.Upsert(metric)
}

func (manager MetricsManager) UpsertSlice(metrics []metric.Metric) error {

	for _, m := range metrics {
		if err := manager.VerifySign(m); err != nil {
			return fmt.Errorf("could not upsert metrics %s: %w", m, err)
		}
	}

	return manager.storage.UpsertSlice(metrics)
}

func (manager MetricsManager) Get(metric metric.Metric) (metric.Metric, error) {

	return manager.storage.Get(metric)
}

func (manager MetricsManager) GetSlice() ([]metric.Metric, error) {

	return manager.storage.GetSlice()
}

// Delete - Удаление метрики
func (manager MetricsManager) Delete(metric metric.Metric) error {

	return manager.storage.Delete(metric)
}

func (manager MetricsManager) String() string {
	return manager.storage.String()
}

func (manager MetricsManager) CheckHealth() bool {
	return manager.storage.CheckHealth()
}

func (manager MetricsManager) Close() error {
	return manager.storage.Close()
}
