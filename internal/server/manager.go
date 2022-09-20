package server

import (
	"context"
	"fmt"
	"time"

	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/errs"
	"metrics-and-alerting/pkg/logpack"
	metricPkg "metrics-and-alerting/pkg/metric"
)

type OptionsManager func(*MetricsManager)

type MetricsManager struct {
	storage       storage.Repository
	logger        *logpack.LogPack
	intervalFlush time.Duration
	restore       bool
	signKey       []byte
	ctx           context.Context
	cancel        context.CancelFunc
}

func New(storage storage.Repository, logger *logpack.LogPack, opts ...OptionsManager) *MetricsManager {

	manager := &MetricsManager{
		storage: storage,
		logger:  logger,
	}

	manager.ctx, manager.cancel = context.WithCancel(context.Background())

	for _, opt := range opts {
		opt(manager)
	}

	if manager.restore {
		if errRestore := storage.Restore(); errRestore != nil {
			logger.Err.Printf("Could not restore: %v\n", errRestore)
		}
	}

	if manager.intervalFlush > 0 {
		go manager.flushByTick(manager.ctx)
	}

	return manager
}

func WithSignKey(signKey []byte) OptionsManager {
	return func(manager *MetricsManager) {
		manager.signKey = signKey
	}
}

func WithFlush(interval time.Duration) OptionsManager {
	return func(manager *MetricsManager) {
		manager.intervalFlush = interval
	}
}

func WithRestore(restore bool) OptionsManager {
	return func(manager *MetricsManager) {
		manager.restore = restore
	}
}

func (manager MetricsManager) flushByTick(ctx context.Context) {

	ticker := time.NewTicker(manager.intervalFlush)

	for {
		select {
		case <-ticker.C:
			if err := manager.storage.Flush(); err != nil {
				manager.logger.Err.Printf("could not flush metrics: %v\n", err)
			}

		case <-ctx.Done():
			return
		}
	}
}

func (manager MetricsManager) accumulateCounter(metric *metricPkg.Metric) {
	if metric.MType != metricPkg.CounterType {
		return
	}

	knownCounter, err := manager.storage.Get(*metric)
	if err != nil {
		return
	}

	accum := *metric.Delta + *knownCounter.Delta
	metric.Delta = &accum

}

// verifySign - Проверка подписи метрики
func (manager MetricsManager) verifySign(metric metricPkg.Metric) error {
	if len(manager.signKey) == 0 {
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

func (manager MetricsManager) Upsert(metric metricPkg.Metric) error {

	if err := manager.verifySign(metric); err != nil {
		return fmt.Errorf("could not upsert metric: %w", err)
	}

	manager.accumulateCounter(&metric)

	err := manager.storage.Upsert(metric)

	if err == nil {
		if err = manager.Flush(); err != nil {
			manager.logger.Err.Printf("Could not flush metrics after upsert: %v\n", err)
		}

		return nil
	}

	return err
}

func (manager MetricsManager) UpsertBatch(metrics []metricPkg.Metric) error {

	for i, m := range metrics {
		if err := manager.verifySign(m); err != nil {
			return fmt.Errorf("could not upsert metrics %s: %w", m, err)
		}

		manager.accumulateCounter(&m)
		metrics[i].Delta = m.Delta

		if err := manager.storage.Upsert(m); err != nil {
			err = fmt.Errorf("could not update metric %s: %w", m.ShotString(), err)
			manager.logger.Err.Println(err)
			return err
		}
	}

	if err := manager.Flush(); err != nil {
		manager.logger.Err.Printf("Could not flush metrics after upsert: batch %v\n", err)
	}

	return nil
}

func (manager MetricsManager) Get(metric metricPkg.Metric) (metricPkg.Metric, error) {

	m, err := manager.storage.Get(metric)
	if err != nil {
		return metricPkg.Metric{}, err
	}

	if hash, err := m.Sign(manager.signKey); err == nil {
		m.Hash = hash
	} else {
		manager.logger.Err.Printf("could not get hash metric: %v\n", err)
	}

	return m, nil
}

func (manager MetricsManager) GetBatch() ([]metricPkg.Metric, error) {

	metrics, err := manager.storage.GetBatch()
	if err != nil {
		return nil, err
	}

	for i, m := range metrics {
		hash, err := m.Sign(manager.signKey)
		if err != nil {
			manager.logger.Err.Printf("could not get hash metric: %v\n", err)
			continue
		}

		metrics[i].Hash = hash
	}

	return metrics, nil
}

func (manager MetricsManager) Delete(metric metricPkg.Metric) error {

	err := manager.storage.Delete(metric)

	if err == nil {
		if err = manager.Flush(); err != nil {
			manager.logger.Err.Printf("Could not flush metrics after delete: %v\n", err)
		}

		return nil
	}

	return err
}

func (manager MetricsManager) Flush() error {

	if manager.intervalFlush == 0 {
		return manager.storage.Flush()
	}

	return nil
}

func (manager MetricsManager) Restore() error {
	return manager.storage.Restore()
}

func (manager MetricsManager) Close() error {
	return manager.storage.Close()
}

func (manager MetricsManager) Health() bool {
	return manager.storage.Health()
}
