package storage

import (
	"encoding/json"
	"strconv"
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
	ID    string          `json:"id"`              // имя метрики
	MType string          `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta json.RawMessage `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value json.RawMessage `json:"value,omitempty"` // значение метрики в случае передачи gauge
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
