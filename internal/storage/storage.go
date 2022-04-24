package storage

import (
	"net/http"
	"strconv"
	"sync"
)

const (
	GuageType   string = "gauge"
	CounterType string = "counter"
)

type Metrics interface {
	Update(name, value, s string) int
	GetGauge(name string) (float64, int)
	GetCounter(name string) (int64, int)
	GetGauges() map[string]float64
	GetGaugesString() map[string]string
	GetCounters() map[string]int64
	Get(t, name string) (string, int)
	Clear()
}

type MetricsData struct {
	mu             sync.Mutex
	metricsGauge   map[string]float64
	metricsCounter map[string]int64
}

func (monitor *MetricsData) Update(name, value, t string) int {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	switch t {
	case GuageType:
		if monitor.metricsGauge == nil {
			monitor.metricsGauge = make(map[string]float64)
		}

		metricValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			//fmt.Println("uncorrect metric value '" + value + "' for type '" + GuageType + "'")
			return http.StatusBadRequest
		}

		monitor.metricsGauge[name] = metricValue

	case CounterType:
		if monitor.metricsCounter == nil {
			monitor.metricsCounter = make(map[string]int64)
		}

		metricValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			//fmt.Println("uncorrect metric value '" + value + "' of type '" + CounterType + "'")
			return http.StatusBadRequest
		}

		monitor.metricsCounter[name] += metricValue

	default:
		//fmt.Println("unknown  metric type: '" + t + "'")
		return http.StatusNotImplemented
	}

	return http.StatusOK
}

func (monitor *MetricsData) GetGauge(name string) (float64, int) {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	value, exist := monitor.metricsGauge[name]
	if !exist {
		return 0, http.StatusNotFound
	}

	return value, http.StatusOK
}

func (monitor *MetricsData) GetCounter(name string) (int64, int) {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	value, exist := monitor.metricsCounter[name]
	if !exist {
		return 0, http.StatusNotFound
	}

	return value, http.StatusOK
}

func (monitor *MetricsData) Get(t, name string) (string, int) {

	switch t {
	case GuageType:
		val, code := monitor.GetGauge(name)
		return strconv.FormatFloat(val, 'f', 3, 64), code
	case CounterType:
		val, code := monitor.GetCounter(name)
		return strconv.FormatInt(val, 10), code
	}

	return "", http.StatusNotFound
}

func (monitor *MetricsData) GetGauges() map[string]float64 {
	return monitor.metricsGauge
}

func (monitor *MetricsData) GetGaugesString() map[string]string {
	if monitor.metricsGauge == nil {
		monitor.metricsGauge = make(map[string]float64)
	}

	metrics := make(map[string]string)

	for k, v := range monitor.metricsGauge {
		metrics[k] = strconv.FormatFloat(v, 'f', 3, 64)
	}

	return metrics
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
