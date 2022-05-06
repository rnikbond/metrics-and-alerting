package storage

import (
	"encoding/json"
	"sort"
	"strconv"
	"sync"

	errst "metrics-and-alerting/pkg/errorsstorage"
)

const (
	GaugeType   string = "gauge"
	CounterType string = "counter"
)

type IStorage interface {
	Get(typeMetric, name string) (string, error)
	FillJSON(data []byte) ([]byte, error)
	Names(typeMetric string) []string
	Count(typeMetric string) int
	Clear()

	Set(typeMetric, name string, value interface{}) error
	Add(typeMetric, name string, value interface{}) error
	Update(typeMetric, name string, value interface{}) error
	UpdateJSON(data []byte) error

	Lock()
	Unlock()

	String() string
}

type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

type MemoryStorage struct {
	mu       sync.Mutex
	gauges   map[string]float64
	counters map[string]int64
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
func (st *MemoryStorage) Set(typeMetric, name string, value interface{}) error {

	if len(name) < 1 {
		return errst.ErrorInvalidName
	}

	switch typeMetric {
	case GaugeType:
		if st.gauges == nil {
			st.gauges = make(map[string]float64)
		}

		if val, err := ToFloat64(value); err != nil {
			return err
		} else {
			st.gauges[name] = val
		}

	case CounterType:
		if st.counters == nil {
			st.counters = make(map[string]int64)
		}

		if val, err := ToInt64(value); err != nil {
			return err
		} else {
			st.counters[name] = val
		}

	default:
		return errst.ErrorUnknownType
	}

	return nil
}

// Add Изменение значения метрики.
// Для типа "gauge" - value должно преобразовываться в float64.
// Для типа "counter" - value должно преобразовываться в int64.
func (st *MemoryStorage) Add(typeMetric, name string, value interface{}) error {
	if len(name) < 1 {
		return errst.ErrorInvalidName
	}

	switch typeMetric {
	case GaugeType:
		if st.gauges == nil {
			st.gauges = make(map[string]float64)
		}

		if val, err := ToFloat64(value); err != nil {
			return err
		} else {
			st.gauges[name] += val
		}

	case CounterType:
		if st.counters == nil {
			st.counters = make(map[string]int64)
		}

		if val, err := ToInt64(value); err != nil {
			return err
		} else {
			st.counters[name] += val
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
func (st *MemoryStorage) Get(typeMetric, name string) (string, error) {

	if len(name) < 1 {
		return "", errst.ErrorNotFound
	}

	switch typeMetric {
	case GaugeType:
		if st.gauges == nil {
			return "", errst.ErrorNotFound
		}

		if value, found := st.gauges[name]; found {
			return strconv.FormatFloat(value, 'f', -1, 64), nil
		}

	case CounterType:
		if st.counters == nil {
			return "", errst.ErrorNotFound
		}

		if value, found := st.counters[name]; found {
			return strconv.FormatInt(value, 10), nil
		}
	}

	return "", errst.ErrorNotFound
}

func (st *MemoryStorage) Names(typeMetric string) []string {

	var keys []string

	switch typeMetric {
	case GaugeType:
		if st.gauges == nil {
			return []string{}
		}

		for key := range st.gauges {
			keys = append(keys, key)
		}

	case CounterType:
		if st.counters == nil {
			return []string{}
		}

		for key := range st.counters {
			keys = append(keys, key)
		}
	}

	sort.Strings(keys)
	return keys
}

// Count количество метрик типа typeMetric
func (st *MemoryStorage) Count(typeMetric string) int {

	switch typeMetric {
	case GaugeType:
		return len(st.gauges)
	case CounterType:
		return len(st.counters)
	}

	return 0
}

func (st *MemoryStorage) Clear() {
	st.gauges = make(map[string]float64)
	st.counters = make(map[string]int64)
}

func (st *MemoryStorage) String() string {

	var s string

	types := []string{GaugeType, CounterType}
	for _, typeMetric := range types {
		names := st.Names(typeMetric)
		for _, name := range names {
			val, err := st.Get(typeMetric, name)
			if err == nil {
				s += typeMetric + "/" + name + "/" + val + "\n"
			}
		}
	}

	return s
}

func (st *MemoryStorage) Lock() {
	st.mu.Lock()
}

func (st *MemoryStorage) Unlock() {
	st.mu.Unlock()
}
