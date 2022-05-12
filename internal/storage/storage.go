package storage

import (
	"encoding/json"

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
	ID    string  `json:"id"`              // имя метрики
	MType string  `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

func createMetric(typeMetric, id string) *Metrics {
	var delta int64
	var value float64

	return &Metrics{
		ID:    id,
		MType: typeMetric,
		Delta: &delta,
		Value: &value,
	}
}

func (metric Metrics) MarshalJSON() ([]byte, error) {

	var delta int64
	var value float64

	if metric.Delta != nil {
		delta = *metric.Delta
	}

	if metric.Value != nil {
		value = *metric.Value
	}

	aliasValue := SerializeMetric{
		ID:    metric.ID,
		MType: metric.MType,
		Delta: delta,
		Value: value,
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
		metric.Value = &deserializer.Value
	case CounterType:
		metric.Delta = &deserializer.Delta
	}

	return nil
}
