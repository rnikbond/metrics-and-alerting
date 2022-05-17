package storage

import (
	"bytes"
	"fmt"
	"strconv"
	"text/tabwriter"

	"metrics-and-alerting/pkg/config"
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

	Save() error
	Restore() error
	SetExternalStorage(cfg *config.Config)
}

type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

func createMetric(typeMetric, id string) *Metrics {
	return &Metrics{
		ID:    id,
		MType: typeMetric,
		Delta: nil,
		Value: nil,
	}
}

func (metric Metrics) String() string {

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "ID\t", metric.ID)
	fmt.Fprintln(w, "TYPE\t", metric.MType)

	if metric.Delta != nil {
		fmt.Fprintln(w, "DELTA\t", strconv.FormatInt(*metric.Delta, 10))
	} else {
		fmt.Fprintln(w, "DELTA\tnil")
	}

	if metric.Value != nil {
		fmt.Fprintln(w, "VALUE\t", strconv.FormatFloat(*metric.Value, 'f', -1, 64))
	} else {
		fmt.Fprintln(w, "VALUE\tnil")
	}

	if err := w.Flush(); err != nil {
		return err.Error()
	}

	return buf.String()
}
