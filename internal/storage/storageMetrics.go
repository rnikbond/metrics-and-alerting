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
	ValuesGaugeType() map[string]float64
	ValueGaugeType(name string) (float64, error)
	ValueCounterType() int64
	SetValueGaugeType(name string, value float64)
	AddValueCounterType(value int64)
}

type MetricsData struct {
	mu        sync.Mutex
	pollCount int64
	Metrics   map[string]float64
}

// обновление значения метрики name типа 'gauge'
func (monitor *MetricsData) SetValueGaugeType(name string, value float64) {

	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	if monitor.Metrics == nil {
		monitor.Metrics = make(map[string]float64)
	}

	monitor.Metrics[name] = value
}

// увеличение значения метрики типа 'counter'
func (monitor *MetricsData) AddValueCounterType(value int64) {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	monitor.pollCount += value
}

// получение всех метрик
func (monitor *MetricsData) ValuesGaugeType() map[string]float64 {
	return monitor.Metrics
}

// получение значения метрики name типа 'gauge'
func (monitor *MetricsData) ValueGaugeType(name string) (float64, error) {

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
func (monitor *MetricsData) ValueCounterType() int64 {
	return monitor.pollCount
}
