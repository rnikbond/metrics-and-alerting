package storage

import (
	"fmt"

	"metrics-and-alerting/pkg/errs"
	"metrics-and-alerting/pkg/metric"
)

type OptionsManager func(*MetricsManager)

type MetricsManager struct {
	storage  Repository
	signKey  []byte
	isVerify bool
}

func NewMetricsManager(storage Repository, opts ...OptionsManager) *MetricsManager {

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

func WithVerify() OptionsManager {
	return func(manager *MetricsManager) {
		manager.isVerify = true
	}
}

// VerifySign - Проверка подписи метрики
func (manager MetricsManager) VerifySign(metric metric.Metric) error {
	if !manager.isVerify {
		return nil
	}

	if len(manager.signKey) < 1 {
		return nil
	}

	hash, err := metric.Sign(manager.signKey)
	if err != nil {
		return err
	}

	if hash != metric.Hash {
		return errs.ErrSignFailed
	}
	return nil
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
			return fmt.Errorf("could not upsert metric: %w", err)
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

func (manager MetricsManager) CheckHealth() bool {
	return manager.storage.CheckHealth()
}

func (manager MetricsManager) Close() error {
	return manager.storage.Close()
}
