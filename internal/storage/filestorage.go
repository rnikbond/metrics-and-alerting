package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"log"
	"os"
	"sort"
	"strconv"

	errst "metrics-and-alerting/pkg/errorsstorage"
)

type FileStorage struct {
	FileName string
	metrics  []Metrics
}

func (st *FileStorage) MetricIdx(typeMetric, id string) (int, error) {

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

func (st *FileStorage) File(flag int) (*os.File, error) {
	if len(st.FileName) < 1 {
		return nil, errors.New("invalid path file")
	}

	return os.OpenFile(st.FileName, flag, 0777)
}

func (st *FileStorage) Read() error {

	st.metrics = []Metrics{}

	file, err := st.File(os.O_RDONLY)
	if err != nil {
		log.Println("error open file fo read: ", err)
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

func (st *FileStorage) Write() error {
	file, err := st.File(os.O_CREATE | os.O_WRONLY | os.O_TRUNC)
	if err != nil {
		log.Println("error open file for write: ", err)
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

func (st *FileStorage) UpdateJSON(data []byte) error {

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
func (st *FileStorage) Update(typeMetric, name string, value interface{}) error {

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
func (st *FileStorage) Set(typeMetric, id string, value interface{}) error {

	st.Read()

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

	st.Write()

	return nil
}

// Add Изменение значения метрики.
// Для типа "gauge" - value должно преобразовываться в float64.
// Для типа "counter" - value должно преобразовываться в int64.
func (st *FileStorage) Add(typeMetric, id string, value interface{}) error {

	st.Read()

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

	st.Write()

	return nil
}

func (st *FileStorage) FillJSON(data []byte) ([]byte, error) {

	if err := st.Read(); err != nil {
		return []byte{}, err
	}

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
func (st *FileStorage) Get(typeMetric, id string) (string, error) {

	if err := st.Read(); err != nil {
		return "", err
	}

	metricIdx, err := st.MetricIdx(typeMetric, id)
	if err != nil {
		return "", err
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

func (st *FileStorage) Names(typeMetric string) []string {

	if err := st.Read(); err != nil {
		return []string{}
	}

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
func (st *FileStorage) Count(typeMetric string) int {

	if err := st.Read(); err != nil {
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

func (st *FileStorage) Clear() {

	st.metrics = nil

	file, err := st.File(os.O_TRUNC)
	if err != nil {
		log.Println("error open file fo read: ", err)
		return
	}

	file.Close()

}

func (st *FileStorage) String() string {

	if err := st.Read(); err != nil {
		return ""
	}

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

func (st *FileStorage) Lock()   {}
func (st *FileStorage) Unlock() {}
