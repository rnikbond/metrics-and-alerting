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
		if monitor.metricsGauge == nil {
			monitor.metricsGauge = make(map[string]float64)

			monitor.metricsGauge["RandomValue"] = 0
			monitor.metricsGauge["Alloc"] = 0
			monitor.metricsGauge["BuckHashSys"] = 0
			monitor.metricsGauge["Frees"] = 0
			monitor.metricsGauge["GCCPUFraction"] = 0
			monitor.metricsGauge["GCSys"] = 0
			monitor.metricsGauge["HeapAlloc"] = 0
			monitor.metricsGauge["HeapIdle"] = 0
			monitor.metricsGauge["HeapInuse"] = 0
			monitor.metricsGauge["HeapObjects"] = 0
			monitor.metricsGauge["HeapReleased"] = 0
			monitor.metricsGauge["HeapSys"] = 0
			monitor.metricsGauge["LastGC"] = 0
			monitor.metricsGauge["Lookups"] = 0
			monitor.metricsGauge["MCacheInuse"] = 0
			monitor.metricsGauge["MCacheSys"] = 0
			monitor.metricsGauge["MSpanInuse"] = 0
			monitor.metricsGauge["MSpanSys"] = 0
			monitor.metricsGauge["Mallocs"] = 0
			monitor.metricsGauge["NextGC"] = 0
			monitor.metricsGauge["NumForcedGC"] = 0
			monitor.metricsGauge["NumGC"] = 0
			monitor.metricsGauge["OtherSys"] = 0
			monitor.metricsGauge["PauseTotalNs"] = 0
			monitor.metricsGauge["StackInuse"] = 0
			monitor.metricsGauge["StackSys"] = 0
			monitor.metricsGauge["Sys"] = 0
			monitor.metricsGauge["TotalAlloc"] = 0
		}

		if _, exist := monitor.metricsGauge[name]; !exist {
			return errors.New("unknown metric '" + name + "' of type '" + GuageType + "'")
		}

		metricValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return errors.New("uncorrect metric value '" + value + "' for type '" + GuageType + "'")
		}

		monitor.metricsGauge[name] = metricValue

	case CounterType:
		if monitor.metricsCounter == nil {
			monitor.metricsCounter = make(map[string]int64)

			monitor.metricsCounter[CounterName] = 0
		}

		if _, exist := monitor.metricsCounter[name]; !exist {
			return errors.New("unknown metric '" + name + "' of type '" + CounterType + "'")
		}

		metricValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return errors.New("uncorrect metric value '" + value + "' of type '" + CounterType + "'")
		}

		monitor.metricsCounter[CounterName] += metricValue

	default:
		return errors.New("unknown  metric type: '" + t + "'")
	}

	return nil
}

func (monitor *MetricsData) GetGauge(name string) (float64, error) {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	value, exist := monitor.metricsGauge[name]
	if !exist {
		return 0, errors.New("metric '" + name + "' does not exist")
	}

	return value, nil
}

func (monitor *MetricsData) GetCounter(name string) (int64, error) {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	value, exist := monitor.metricsCounter[name]
	if !exist {
		return 0, errors.New("metric '" + name + "' does not exist")
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
