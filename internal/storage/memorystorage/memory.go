package memorystorage

import (
	"fmt"
	"strings"
	"sync"

	"metrics-and-alerting/pkg/errs"
	metricPkg "metrics-and-alerting/pkg/metric"
)

type MemoryStorage struct {
	mu      sync.RWMutex
	metrics []metricPkg.Metric
}

func NewStorage() *MemoryStorage {
	return &MemoryStorage{
		metrics: make([]metricPkg.Metric, 0),
	}
}

// Find - Поиск метрики в слайсе
// Возвращается индекс метрики в слайсе и ошибку, если такой метрики не существует
func (ms MemoryStorage) Find(mSeek metricPkg.Metric) (int, error) {

	for i, m := range ms.metrics {
		if m.MType == mSeek.MType && m.ID == mSeek.ID {
			return i, nil
		}
	}

	return -1, errs.ErrNotFound
}
func (ms *MemoryStorage) Set(metric metricPkg.Metric) error {

	ms.mu.Lock()
	defer ms.mu.Unlock()

	if idx, err := ms.Find(metric); err != nil {
		ms.metrics = append(ms.metrics, metric)
	} else {

		ms.metrics[idx].Hash = metric.Hash

		switch metric.MType {
		case metricPkg.GaugeType:
			ms.metrics[idx].Value = metric.Value
		case metricPkg.CounterType:
			*ms.metrics[idx].Delta = *metric.Delta
		}
	}

	return nil
}

// Upsert Обновление значения метрики, или добавление метрики, если ранее её не существовало
func (ms *MemoryStorage) Upsert(metric metricPkg.Metric) error {

	ms.mu.Lock()
	defer ms.mu.Unlock()

	if idx, err := ms.Find(metric); err != nil {
		ms.metrics = append(ms.metrics, metric)
	} else {

		ms.metrics[idx].Hash = metric.Hash

		switch metric.MType {
		case metricPkg.GaugeType:
			ms.metrics[idx].Value = metric.Value
		case metricPkg.CounterType:
			*ms.metrics[idx].Delta += *metric.Delta
		}
	}

	return nil
}

// UpsertSlice - Обновление всех метрик
func (ms *MemoryStorage) UpsertSlice(metrics []metricPkg.Metric) error {

	for _, m := range metrics {
		if err := ms.Upsert(m); err != nil {
			return fmt.Errorf("can not upsert metrics: %w", err)
		}
	}

	return nil
}

// Get - Получение полность заполненной метрики
func (ms *MemoryStorage) Get(metric metricPkg.Metric) (metricPkg.Metric, error) {

	ms.mu.RLock()
	defer ms.mu.RUnlock()

	idx, err := ms.Find(metric)
	if err != nil {
		return metricPkg.Metric{}, err
	}

	return ms.metrics[idx], nil
}

// GetSlice - Получение всех метрик в виде списка
func (ms MemoryStorage) GetSlice() ([]metricPkg.Metric, error) {

	return ms.metrics, nil
}

// Delete - Удаление метрики
func (ms *MemoryStorage) Delete(metric metricPkg.Metric) error {

	ms.mu.Lock()
	defer ms.mu.Unlock()

	idx, err := ms.Find(metric)
	if err != nil {
		return err
	}

	ms.metrics = append(ms.metrics[:idx], ms.metrics[idx+1:]...)
	return nil
}

func (ms MemoryStorage) String() string {

	builder := strings.Builder{}

	builder.WriteString("\nSTORAGE DATA\n")
	builder.WriteString(fmt.Sprintf("Items in storage: %d\n", len(ms.metrics)))

	for _, m := range ms.metrics {

		builder.WriteString(m.String())
	}

	return builder.String()
}

func (ms MemoryStorage) CheckHealth() bool {
	return true
}

func (ms MemoryStorage) Close() error {
	return nil
}
