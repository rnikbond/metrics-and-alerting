package storageMetrics

import (
	"errors"
	"sync"
)

const (
	GuageType   string = "gauge"
	CounterType string = "counter"
)

type Metrics interface {
	GetMetricsGauge() map[string]float64
	GetMetricGauge(name string) (float64, error)
	GetMetricCounter() int64
	SetMetricGauge(name string, value float64)
	AppendToMetricCounter(value int64)
}

type MetricsData struct {
	mu        sync.Mutex
	pollCount int64
	Metrics   map[string]float64
}

// обновление значения метрики name типа 'gauge'
func (monitor *MetricsData) SetMetricGauge(name string, value float64) {

	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	if monitor.Metrics == nil {
		monitor.Metrics = make(map[string]float64)
	}

	monitor.Metrics[name] = value
}

// увеличение значения метрики типа 'counter'
func (monitor *MetricsData) AppendToMetricCounter(value int64) {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	monitor.pollCount += value
}

// получение всех метрик
func (monitor *MetricsData) GetMetricsGauge() map[string]float64 {
	return monitor.Metrics
}

// получение значения метрики name типа 'gauge'
func (monitor *MetricsData) GetMetricGauge(name string) (float64, error) {

	if monitor.Metrics == nil {
		return 0, errors.New("metric '" + name + "' does not exist")
	}

	value, exist := monitor.Metrics[name]
	if !exist {
		return 0, errors.New("metric '" + name + "' does not exist")
	}

	return value, nil
}

// получение значение метрики типа 'counter'
func (monitor *MetricsData) GetMetricCounter() int64 {
	return monitor.pollCount
}
