package memstore

import (
	"fmt"

	"metrics-and-alerting/pkg/errs"
	metricPkg "metrics-and-alerting/pkg/metric"
)

type Storage struct {
	metrics []metricPkg.Metric
}

func New() *Storage {
	return &Storage{
		metrics: make([]metricPkg.Metric, 0),
	}
}

// Find - Поиск метрики в слайсе
// Возвращается индекс метрики в слайсе и ошибку, если такой метрики не существует
func (store Storage) Find(mSeek metricPkg.Metric) (int, error) {

	for i, m := range store.metrics {
		if m.MType == mSeek.MType && m.ID == mSeek.ID {
			return i, nil
		}
	}

	return -1, errs.ErrNotFound
}

// Upsert Обновление значения метрики, или добавление метрики, если ранее её не существовало
func (store *Storage) Upsert(metric metricPkg.Metric) error {

	if idx, err := store.Find(metric); err != nil {
		store.metrics = append(store.metrics, metric)
	} else {

		store.metrics[idx].Hash = metric.Hash

		switch metric.MType {
		case metricPkg.GaugeType:
			store.metrics[idx].Value = metric.Value
		case metricPkg.CounterType:
			store.metrics[idx].Delta = metric.Delta
		}
	}

	return nil
}

// UpsertBatch Обновление набора метрик
func (store *Storage) UpsertBatch(metrics []metricPkg.Metric) error {

	for _, m := range metrics {
		if err := store.Upsert(m); err != nil {
			return fmt.Errorf("can not upsert metrics: %w", err)
		}
	}

	return nil
}

// Get - Получение полность заполненной метрики
func (store Storage) Get(metric metricPkg.Metric) (metricPkg.Metric, error) {

	idx, err := store.Find(metric)
	if err != nil {
		return metricPkg.Metric{}, err
	}

	return store.metrics[idx], nil
}

// GetBatch Получение всех метрик в виде слайса
func (store Storage) GetBatch() ([]metricPkg.Metric, error) {

	return store.metrics, nil
}

// Delete - Удаление метрики
func (store *Storage) Delete(metric metricPkg.Metric) error {

	idx, err := store.Find(metric)
	if err != nil {
		return err
	}

	store.metrics = append(store.metrics[:idx], store.metrics[idx+1:]...)
	return nil
}

func (store Storage) Flush() error {
	return nil
}

func (store Storage) Restore() error {
	return nil
}

func (store Storage) Close() error {
	return nil
}

func (store Storage) Health() bool {
	return true
}
