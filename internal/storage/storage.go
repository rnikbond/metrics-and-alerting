package storage

import (
	"reflect"
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
	Names(typeMetric string) []string
	Count(typeMetric string) int
	Clear()

	Set(typeMetric, name string, value interface{}) error
	Add(typeMetric, name string, value interface{}) error
	Update(typeMetric, name string, value interface{}) error

	Lock()
	Unlock()

	String() string
}

type MemoryStorage struct {
	mu       sync.Mutex
	gauges   map[string]float64
	counters map[string]int64
}

// Update Обновление значения метрики.
// Для типа "gauge" - значение обновляется на value
// Для типа "counter" -  старому значению добавляется новое значение value
func (st *MemoryStorage) Update(typeMetric, name string, value interface{}) error {

	if len(name) < 1 {
		return errst.ErrorIncorrectName
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

// Set Изменение значения метрики
// Для типа "gauge" - value должно преобразовываться в float64
// Для типа "counter" - value должно преобразовываться в int64
func (st *MemoryStorage) Set(typeMetric, name string, value interface{}) error {

	if len(name) < 1 {
		return errst.ErrorIncorrectName
	}

	switch typeMetric {
	case GaugeType:
		if st.gauges == nil {
			st.gauges = make(map[string]float64)
		}

		reflVal := reflect.ValueOf(value)

		switch reflVal.Kind() {
		case reflect.Float32, reflect.Float64:
			st.gauges[name] = reflVal.Float()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			st.gauges[name] = float64(reflVal.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			st.gauges[name] = float64(reflVal.Uint())
		case reflect.String:
			val, err := strconv.ParseFloat(reflVal.String(), 64)
			if err != nil {
				//log.Printf("MemoryStorage.Set() error type %v for metric: %s/%s\n", reflVal.Kind(), typeMetric, name)
				return errst.ErrorIncorrectValue
			}
			st.gauges[name] = val
		default:
			//log.Printf("MemoryStorage.Set() error type %v for metric: %s/%s\n", reflVal.Kind(), typeMetric, name)
			return errst.ErrorIncorrectValue
		}
	case CounterType:
		if st.counters == nil {
			st.counters = make(map[string]int64)
		}

		reflVal := reflect.ValueOf(value)

		switch reflVal.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			st.counters[name] = reflVal.Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			st.counters[name] = int64(reflVal.Uint())
		case reflect.String:
			val, err := strconv.ParseInt(reflVal.String(), 10, 64)
			if err != nil {
				//log.Printf("MemoryStorage.Set() error type %v for metric: %s/%s\n", reflVal.Kind(), typeMetric, name)
				return errst.ErrorIncorrectValue
			}
			st.counters[name] = val
		default:
			//log.Printf("MemoryStorage.Set() error type %v for metric: %s/%s", reflVal.Kind(), typeMetric, name)
			return errst.ErrorIncorrectValue
		}

		st.counters[name] = reflVal.Int()

	default:
		return errst.ErrorUnknownType
	}

	return nil
}

// Add Изменение значения метрики
// Для типа "gauge" - value должно преобразовываться в float64
// Для типа "counter" - value должно преобразовываться в int64
func (st *MemoryStorage) Add(typeMetric, name string, value interface{}) error {
	if len(name) < 1 {
		return errst.ErrorIncorrectName
	}

	switch typeMetric {
	case GaugeType:
		if st.gauges == nil {
			st.gauges = make(map[string]float64)
		}

		reflVal := reflect.ValueOf(value)
		if reflVal.Kind() != reflect.Float64 {
			return errst.ErrorIncorrectValue
		}

		switch reflVal.Kind() {
		case reflect.Float32, reflect.Float64:
			st.gauges[name] += reflVal.Float()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			st.gauges[name] += float64(reflVal.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			st.gauges[name] += float64(reflVal.Uint())
		case reflect.String:
			val, err := strconv.ParseFloat(reflVal.String(), 64)
			if err == nil {
				//log.Printf("MemoryStorage.Set() error type %v for metric: %s/%s\n", reflVal.Kind(), typeMetric, name)
				return errst.ErrorIncorrectValue
			}

			st.gauges[name] += val

		default:
			//log.Printf("MemoryStorage.Add() error type %v for metric: %s/%s\n", reflVal.Kind(), typeMetric, name)
			return errst.ErrorIncorrectValue
		}

	case CounterType:
		if st.counters == nil {
			st.counters = make(map[string]int64)
		}

		reflVal := reflect.ValueOf(value)

		switch reflVal.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			st.counters[name] += reflVal.Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			st.counters[name] += int64(reflVal.Uint())
		case reflect.String:
			val, err := strconv.ParseInt(reflVal.String(), 10, 64)
			if err != nil {
				//log.Printf("MemoryStorage.Set() error type %v for metric: %s/%s\n", reflVal.Kind(), typeMetric, name)
				return errst.ErrorIncorrectValue
			}
			st.counters[name] += val
		default:
			//log.Printf("MemoryStorage.Add() error type %v for metric: %s/%s\n", reflVal.Kind(), typeMetric, name)
			return errst.ErrorIncorrectValue
		}

	default:
		//log.Printf("MemoryStorage.Add() error unknown type %s\n", typeMetric)
		return errst.ErrorUnknownType
	}

	return nil
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
			return strconv.FormatFloat(value, 'f', 3, 64), nil
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
