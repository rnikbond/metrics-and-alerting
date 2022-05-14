package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"log"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"metrics-and-alerting/pkg/config"
	errst "metrics-and-alerting/pkg/errorsstorage"
)

type MemoryStorage struct {
	mu      sync.Mutex
	metrics []Metrics
	cfg     config.Config
}

func (st *MemoryStorage) isStore() bool {

	return len(st.cfg.StoreFile) > 0
}

func (st *MemoryStorage) isStoreSync() bool {

	return st.isStore() && st.cfg.StoreInterval == 0
}

func (st *MemoryStorage) File(flag int, perm os.FileMode) (*os.File, error) {

	if len(st.cfg.StoreFile) < 1 {
		return nil, errors.New("invalid path file")
	}

	return os.OpenFile(st.cfg.StoreFile, flag, perm)
}

func (st *MemoryStorage) SetExternalStorage(cfg *config.Config) {
	st.cfg = *cfg

	if st.isStoreSync() {
		return
	}

	go func() {
		timer := time.NewTimer(st.cfg.StoreInterval)

		for {
			<-timer.C
			if err := st.Save(); err != nil {
				log.Println(err.Error())
			}
		}
	}()
}

func (st *MemoryStorage) Save() error {
	file, err := st.File(os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		log.Println("error open file for write - " + err.Error() + ". Path: " + st.cfg.StoreFile)
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, metric := range st.metrics {

		data, err := json.Marshal(&metric)
		if err != nil {
			log.Println(err)
			continue
		}

		if _, err = writer.Write(data); err == nil {
			writer.WriteByte('\n')
		} else {
			log.Println("can not write data: ", err)
		}
	}

	return writer.Flush()
}

func (st *MemoryStorage) Restore() error {
	st.metrics = []Metrics{}

	file, err := st.File(os.O_RDONLY, 0400)
	if err != nil {
		log.Println("error open file for read - " + err.Error() + ". Path: " + st.cfg.StoreFile)
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		data := scanner.Bytes()
		metric := Metrics{}

		if err := json.Unmarshal(data, &metric); err != nil {
			log.Println(err)
			continue
		}

		st.metrics = append(st.metrics, metric)
	}

	return nil
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

		val, err := ToFloat64(value)
		if err != nil {
			return err
		}

		st.metrics[metricIdx].Value = &val

	case CounterType:

		val, err := ToInt64(value)
		if err != nil {
			return err
		}

		st.metrics[metricIdx].Delta = &val

	default:
		return errst.ErrorUnknownType
	}

	if st.isStoreSync() {
		st.Save()
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

		val, err := ToFloat64(value)
		if err != nil {
			return err
		}

		if st.metrics[metricIdx].Value == nil {
			st.metrics[metricIdx].Value = &val
		} else {
			*st.metrics[metricIdx].Value += val
		}

	case CounterType:

		val, err := ToInt64(value)
		if err != nil {
			return err
		}

		if st.metrics[metricIdx].Delta == nil {
			st.metrics[metricIdx].Delta = &val
		} else {
			*st.metrics[metricIdx].Delta += val
		}

	default:
		return errst.ErrorUnknownType
	}

	if st.isStoreSync() {
		st.Save()
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
		if val, err := strconv.ParseFloat(val, 64); err == nil {
			metric.Value = &val
		}

	case CounterType:
		if val, err := strconv.ParseInt(val, 10, 64); err == nil {
			metric.Delta = &val
		}
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

func (st *MemoryStorage) String() string {

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
