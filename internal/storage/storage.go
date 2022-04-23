package storage

import (
	"errors"
	"strconv"
	"sync"
)

const (
	GuageType   string = "gauge"
	CounterType string = "counter"
	CounterName string = "PollCount"
)

type Metrics interface {
	Update(name, value, s string) error
	GetGauge(name string) (float64, error)
	GetCounter(name string) (int64, error)
	GetGauges() map[string]float64
	GetCounters() map[string]int64
	Clear()
}

type MetricsData struct {
	mu             sync.Mutex
	metricsGauge   map[string]float64
	metricsCounter map[string]int64
}

func (monitor *MetricsData) Update(name, value, t string) error {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	switch t {
	case GuageType:
		if name == CounterName {
			return errors.New("Metric name '" + name + "' is not '" + string(GuageType) + "' type")
		}

		metricValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return errors.New("Uncorrect metric value '" + value + "' for type '" + string(GuageType) + "'")
		}

		if monitor.metricsGauge == nil {
			monitor.metricsGauge = make(map[string]float64)
		}

		monitor.metricsGauge[name] = metricValue

	case CounterType:
		if name != CounterName {
			return errors.New("Metric name '" + name + "' is not '" + string(CounterType) + "' type")
		}

		metricValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return errors.New("Uncorrect metric value '" + value + "' for type '" + string(CounterType) + "'")
		}

		if monitor.metricsCounter == nil {
			monitor.metricsCounter = make(map[string]int64)
		}

		if _, exist := monitor.metricsCounter[name]; !exist {
			monitor.metricsCounter[CounterName] = metricValue
		} else {
			monitor.metricsCounter[CounterName] += metricValue
		}

	default:
		return errors.New("Unknown  metric type: '" + string(t) + "'")
	}

	return nil
}

func (monitor *MetricsData) GetGauge(name string) (float64, error) {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	value, exist := monitor.metricsGauge[name]
	if !exist {
		return 0, errors.New("Metric '" + name + "' does not exist")
	}

	return value, nil
}

func (monitor *MetricsData) GetCounter(name string) (int64, error) {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	value, exist := monitor.metricsCounter[name]
	if !exist {
		return 0, errors.New("Metric '" + name + "' does not exist")
	}

	return value, nil
}

func (monitor *MetricsData) GetGauges() map[string]float64 {
	return monitor.metricsGauge
}

func (monitor *MetricsData) GetCounters() map[string]int64 {
	return monitor.metricsCounter
}

func (monitor *MetricsData) Clear() {

	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	if monitor.metricsGauge != nil {
		monitor.metricsGauge = make(map[string]float64)
	}

	if monitor.metricsCounter != nil {
		monitor.metricsCounter = make(map[string]int64)
	}
}