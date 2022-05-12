package storage

import (
	"encoding/json"
	"strconv"

	errst "metrics-and-alerting/pkg/errorsstorage"
)

const (
	GaugeType   string = "gauge"
	CounterType string = "counter"
)

type IStorage interface {
	Get(typeMetric, id string) (string, error)
	FillJSON(data []byte) ([]byte, error)
	Names(typeMetric string) []string
	Count(typeMetric string) int

	Clear()

	Set(typeMetric, id string, value interface{}) error
	Add(typeMetric, id string, value interface{}) error
	Update(typeMetric, id string, value interface{}) error
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

type SerializeMetric struct {
	ID    string `json:"id"`              // имя метрики
	MType string `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta string `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value string `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

func createMetric(typeMetric, id string) *Metrics {
	return &Metrics{
		ID:    id,
		MType: typeMetric,
		Delta: nil,
		Value: nil,
	}
}

func (metric *Metrics) String() string {
	s := metric.MType + "/" + metric.ID

	if metric.Delta != nil {
		s += "/" + strconv.FormatInt(*metric.Delta, 10)
	}

	if metric.Value != nil {
		s += "/" + strconv.FormatFloat(*metric.Value, 'f', -1, 64)
	}

	return s
}

func (metric Metrics) MarshalJSON() ([]byte, error) {

	aliasValue := SerializeMetric{
		ID:    metric.ID,
		MType: metric.MType,
	}

	switch metric.MType {
	case GaugeType:
		if metric.Value != nil {
			aliasValue.Value = strconv.FormatFloat(*metric.Value, 'f', -1, 64)
		}

	case CounterType:
		if metric.Delta != nil {
			aliasValue.Delta = strconv.FormatInt(*metric.Delta, 10)
		}
	}

	return json.Marshal(&aliasValue)
}

func (metric *Metrics) UnmarshalJSON(data []byte) error {

	deserializer := SerializeMetric{}
	if err := json.Unmarshal(data, &deserializer); err != nil {
		return errst.ErrorInvalidJSON
	}

	metric.ID = deserializer.ID
	metric.MType = deserializer.MType
	metric.Delta = nil
	metric.Value = nil

	switch deserializer.MType {
	case GaugeType:
		if val, err := strconv.ParseFloat(deserializer.Value, 64); err == nil {
			metric.Value = &val
		}

	case CounterType:
		if val, err := strconv.ParseInt(deserializer.Delta, 10, 64); err == nil {
			metric.Delta = &val
		}
	}

	return nil
}
