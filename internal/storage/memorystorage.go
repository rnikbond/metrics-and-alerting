package storage

import (
	"encoding/json"
	"sort"
	"strconv"
	"sync"

	errst "metrics-and-alerting/pkg/errorsstorage"
)

type MemoryStorage struct {
	mu      sync.Mutex
	metrics []Metrics
}

func (st *MemoryStorage) MetricIdx(typeMetric, id string) (int, error) {

	if len(typeMetric) < 1 {
		return 0, errst.ErrorInvalidType
	}

	if len(id) < 1 {
		return 0, errst.ErrorInvalidName
	}

	for i, m := range st.metrics {
		if m.MType == typeMetric && m.ID == id {
			return i, nil
		}
	}

	return 0, errst.ErrorNotFound
}

func (st *MemoryStorage) UpdateJSON(data []byte) error {

	var metric Metrics

	if err := json.Unmarshal(data, &metric); err != nil {
		return errst.ErrorInvalidJSON
	}

	switch metric.MType {
	case GaugeType:
		if metric.Value == nil {
			return errst.ErrorInvalidValue
		}

		return st.Update(metric.MType, metric.ID, *metric.Value)
	case CounterType:
		if metric.Delta == nil {
			return errst.ErrorInvalidValue
		}

		return st.Update(metric.MType, metric.ID, *metric.Delta)
	default:
		return errst.ErrorUnknownType
	}
}

// Update Обновление значения метрики.
// Для типа "gauge" - значение обновляется на value.
// Для типа "counter" -  старому значению добавляется новое значение value.
func (st *MemoryStorage) Update(typeMetric, name string, value interface{}) error {

	if len(name) < 1 {
		return errst.ErrorInvalidName
	}

	switch typeMetric {
	case GaugeType:
		return st.Set(typeMetric, name, value)
	case CounterType:
		return st.Add(typeMetric, name, value)
	default:
		return errst.ErrorUnknownType
	}
}

// Set Изменение значения метрики.
// Для типа "gauge" - value должно преобразовываться в float64.
// Для типа "counter" - value должно преобразовываться в int64.
func (st *MemoryStorage) Set(typeMetric, id string, value interface{}) error {

	metricIdx, errFoundMetric := st.MetricIdx(typeMetric, id)
	if errFoundMetric != nil {
		st.metrics = append(st.metrics, *createMetric(typeMetric, id))
		metricIdx = len(st.metrics) - 1
	}

	switch typeMetric {
	case GaugeType:

		if val, err := ToFloat64(value); err != nil {
			return err
		} else {
			st.metrics[metricIdx].Value = &val
		}

	case CounterType:

		if val, err := ToInt64(value); err != nil {
			return err
		} else {
			st.metrics[metricIdx].Delta = &val
		}

	default:
		return errst.ErrorUnknownType
	}

	return nil
}

// Add Изменение значения метрики.
// Для типа "gauge" - value должно преобразовываться в float64.
// Для типа "counter" - value должно преобразовываться в int64.
func (st *MemoryStorage) Add(typeMetric, id string, value interface{}) error {

	metricIdx, errFoundMetric := st.MetricIdx(typeMetric, id)
	if errFoundMetric != nil {
		st.metrics = append(st.metrics, *createMetric(typeMetric, id))
		metricIdx = len(st.metrics) - 1
	}

	switch typeMetric {
	case GaugeType:

		if val, err := ToFloat64(value); err != nil {
			return err
		} else {
			*st.metrics[metricIdx].Value += val
		}

	case CounterType:

		if val, err := ToInt64(value); err != nil {
			return err
		} else {
			*st.metrics[metricIdx].Delta += val
		}

	default:
		return errst.ErrorUnknownType
	}

	return nil
}

func (st *MemoryStorage) FillJSON(data []byte) ([]byte, error) {
	var metric Metrics

	if err := json.Unmarshal(data, &metric); err != nil {
		return []byte{}, errst.ErrorInvalidJSON
	}

	val, err := st.Get(metric.MType, metric.ID)
	if err != nil {
		return []byte{}, err
	}

	switch metric.MType {
	case GaugeType:
		valFloat, _ := strconv.ParseFloat(val, 64)
		metric.Value = &valFloat
	case CounterType:
		valInt, _ := strconv.ParseInt(val, 10, 64)
		metric.Delta = &valInt
	}

	readyData, err := json.Marshal(&metric)
	if err != nil {
		return []byte{}, errst.ErrorInternal
	}

	return readyData, nil
}

// Get Получение значения метрики
func (st *MemoryStorage) Get(typeMetric, id string) (string, error) {

	metricIdx, err := st.MetricIdx(typeMetric, id)
	if err != nil {
		return "", errst.ErrorNotFound
	}

	switch st.metrics[metricIdx].MType {
	case GaugeType:
		if st.metrics[metricIdx].Value != nil {
			return strconv.FormatFloat(*st.metrics[metricIdx].Value, 'f', -1, 64), nil
		}

	case CounterType:
		if st.metrics[metricIdx].Delta != nil {
			return strconv.FormatInt(*st.metrics[metricIdx].Delta, 10), nil
		}
	}

	return "", errst.ErrorNotFound
}

func (st *MemoryStorage) Names(typeMetric string) []string {

	var keys []string

	for _, metric := range st.metrics {
		if metric.MType == typeMetric {
			keys = append(keys, metric.ID)
		}
	}

	sort.Strings(keys)
	return keys
}

// Count количество метрик типа typeMetric
func (st *MemoryStorage) Count(typeMetric string) int {

	if st.metrics == nil {
		return 0
	}

	count := 0
	for _, metric := range st.metrics {
		if metric.MType == typeMetric {
			count++
		}
	}
	return count
}

func (st MemoryStorage) String() string {

	var s string

	for _, metric := range st.metrics {
		val, err := st.Get(metric.MType, metric.ID)
		if err == nil {
			s += metric.MType + "/" + metric.ID + "/" + val + "\n"
		}
	}

	return s
}

func (st *MemoryStorage) Clear() {
	st.metrics = nil
}

func (st *MemoryStorage) Lock() {
	st.mu.Lock()
}

func (st *MemoryStorage) Unlock() {
	st.mu.Unlock()
}
