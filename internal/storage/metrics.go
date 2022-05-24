package storage

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"text/tabwriter"
)

const (
	GaugeType   string = "gauge"
	CounterType string = "counter"
)

type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
	Hash  string   `json:"hash,omitempty"`  // значение хеш-функции
}

func NewMetric(typeMetric, id string, value ...interface{}) Metrics {
	metric := Metrics{
		ID:    id,
		MType: typeMetric,
	}

	if len(value) == 1 {
		switch typeMetric {
		case GaugeType:
			val, err := ToFloat64(value[0])
			if err == nil {
				metric.Value = &val
			}

		case CounterType:
			val, err := ToInt64(value[0])
			if err == nil {
				metric.Delta = &val
			}
		}
	}

	return metric
}

// Sign - подпись метрики
func Sign(metric *Metrics, key []byte) (string, error) {

	if len(key) < 1 {
		// @TODO Добавить ошибку для ключа подписи
		return ``, nil
	}

	var src string

	switch metric.MType {
	case CounterType:
		if metric.Delta == nil {
			return ``, ErrorInvalidValue
		}

		src = fmt.Sprintf("%s:%s:%d",
			metric.ID,
			metric.MType,
			*metric.Delta)

	case GaugeType:
		if metric.Value == nil {
			return ``, ErrorInvalidValue
		}

		src = fmt.Sprintf("%s:%s:%f",
			metric.ID,
			metric.MType,
			*metric.Value)
	default:
		return ``, ErrorUnknownType
	}

	h := hmac.New(sha256.New, key)
	if _, err := h.Write([]byte(src)); err != nil {
		return ``, err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// FromJSON - Преобразование JSON данных в структуру метрики
func FromJSON(data []byte) (Metrics, error) {
	var metric Metrics

	if errDecode := json.Unmarshal(data, &metric); errDecode != nil {
		return Metrics{}, ErrorInvalidJSON
	}

	return metric, nil
}

// ToJSON - Преобразование метрики в JSON вид
func (metric *Metrics) ToJSON() ([]byte, error) {

	data, errEncode := json.Marshal(&metric)
	if errEncode != nil {
		return []byte{}, errEncode
	}

	return data, nil
}

func (metric *Metrics) ToMap() (map[string]string, error) {
	data := make(map[string]string)
	data["type"] = metric.MType
	data["name"] = metric.ID

	switch metric.MType {
	case GaugeType:
		if metric.Value == nil {
			return nil, ErrorInvalidValue
		}

		data["value"] = strconv.FormatFloat(*metric.Value, 'f', -1, 64)

	case CounterType:
		if metric.Delta == nil {
			return nil, ErrorInvalidValue
		}

		data["value"] = strconv.FormatInt(*metric.Delta, 10)

	default:
		return nil, ErrorInvalidType
	}

	return data, nil
}

func (metric Metrics) StringValue() string {

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

func (metric Metrics) ShotString() string {
	return metric.MType + "/" + metric.ID + "/" + metric.StringValue()
}

func (metric Metrics) String() string {

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
