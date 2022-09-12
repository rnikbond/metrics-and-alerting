package storage

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"text/tabwriter"
)

const (
	GaugeType   string = "gauge"
	CounterType string = "counter"
)

type (
	OptionMetric func(*Metric)
)

type Metric struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
	Hash  string   `json:"hash,omitempty"`  // значение хеш-функции
}

func CreateMetric(typeMetric, id string, value ...interface{}) (Metric, error) {

	if len(id) < 1 {
		return Metric{}, ErrInvalidID
	}

	if len(typeMetric) < 1 {
		return Metric{}, ErrInvalidType
	}

	metric := Metric{
		ID:    id,
		MType: typeMetric,
	}

	if len(value) == 1 {
		switch typeMetric {
		case GaugeType:
			val, err := ToFloat64(value[0])
			if err != nil {
				return Metric{}, fmt.Errorf("can not create metric: %w", err)
			}

			metric.Value = &val

		case CounterType:
			val, err := ToInt64(value[0])
			if err != nil {
				return Metric{}, fmt.Errorf("can not create metric: %w", err)
			}
			metric.Delta = &val
		default:
			return Metric{}, fmt.Errorf("can not create metric: %w", ErrUnknownType)
		}
	}

	return metric, nil
}

func NewMetric(typeMetric, id string, opts ...OptionMetric) (Metric, error) {

	if len(id) < 1 {
		return Metric{}, ErrInvalidID
	}

	if len(typeMetric) < 1 {
		return Metric{}, ErrInvalidType
	}

	m := Metric{
		ID:    id,
		MType: typeMetric,
	}

	for _, opt := range opts {
		opt(&m)
	}

	return m, nil
}

func WithValueFloat64(val float64) OptionMetric {
	return func(metric *Metric) {

		switch metric.MType {

		case GaugeType:
			metric.Value = &val

		case CounterType:
			tmp := int64(val)
			metric.Delta = &tmp
		}
	}
}

func WithValueInt64(val int64) OptionMetric {
	return func(metric *Metric) {

		switch metric.MType {

		case GaugeType:
			tmp := float64(val)
			metric.Value = &tmp

		case CounterType:
			metric.Delta = &val
		}
	}
}

func (metric Metric) StringValue() string {

	switch metric.MType {
	case GaugeType:
		if metric.Value != nil {
			return strconv.FormatFloat(*metric.Value, 'f', -1, 64)
		}

	case CounterType:
		if metric.Delta != nil {
			return strconv.FormatInt(*metric.Delta, 10)
		}
	}
	return ``
}

// Sign - подпись метрики
func Sign(metric Metric, key []byte) (string, error) {

	if len(key) < 1 {
		return ``, nil
	}

	var src string

	switch metric.MType {
	case CounterType:
		if metric.Delta == nil {
			return ``, ErrInvalidValue
		}

		src = fmt.Sprintf("%s:%s:%d",
			metric.ID,
			metric.MType,
			*metric.Delta)

	case GaugeType:
		if metric.Value == nil {
			return ``, ErrInvalidValue
		}

		src = fmt.Sprintf("%s:%s:%f",
			metric.ID,
			metric.MType,
			*metric.Value)
	default:
		return ``, ErrUnknownType
	}

	h := hmac.New(sha256.New, key)
	if _, err := h.Write([]byte(src)); err != nil {
		return ``, err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func (metric Metric) Map() (map[string]string, error) {

	data := make(map[string]string, 3)
	data["id"] = metric.ID
	data["type"] = metric.MType

	switch metric.MType {
	case GaugeType:
		if metric.Value == nil {
			return nil, ErrInvalidValue
		}
		data["value"] = strconv.FormatFloat(*metric.Value, 'f', -1, 64)

	case CounterType:
		if metric.Delta == nil {
			return nil, ErrInvalidValue
		}
		data["value"] = strconv.FormatInt(*metric.Delta, 10)

	default:
		return nil, ErrUnknownType
	}

	return data, nil
}

func (metric Metric) ShotString() string {
	return metric.MType + "/" + metric.ID + "/" + metric.StringValue()
}

func (metric Metric) String() string {

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', tabwriter.AlignRight)

	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "ID\t", metric.ID)
	fmt.Fprintln(w, "TYPE\t", metric.MType)
	fmt.Fprintln(w, "HASH\t", metric.Hash)

	if metric.Delta != nil {
		fmt.Fprintln(w, "DELTA\t", strconv.FormatInt(*metric.Delta, 10))
	} else {
		fmt.Fprintln(w, "DELTA\t", "nil")
	}

	if metric.Value != nil {
		fmt.Fprintln(w, "VALUE\t", strconv.FormatFloat(*metric.Value, 'f', -1, 64))
	} else {
		fmt.Fprintln(w, "VALUE\t", "nil")
	}

	if err := w.Flush(); err != nil {
		return err.Error()
	}

	return buf.String()
}
