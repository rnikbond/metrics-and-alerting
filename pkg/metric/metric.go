package metric

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"metrics-and-alerting/pkg/errs"
)

const (
	GaugeType   string = "gauge"
	CounterType string = "counter"
)

type (
	OptionsMetric func(*Metric) error

	Metric struct {
		ID    string   `json:"id"`              // имя метрики
		MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
		Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
		Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
		Hash  string   `json:"hash,omitempty"`  // значение метрики
	}
)

// CreateMetric Создание метрики
// Используется паттерн "Функциональные опции"
func CreateMetric(typeMetric, id string, opts ...OptionsMetric) (Metric, error) {

	if len(id) < 1 {
		return Metric{}, errs.ErrInvalidID
	}

	if len(typeMetric) < 1 {
		return Metric{}, errs.ErrInvalidID
	}

	metric := Metric{
		ID:    id,
		MType: typeMetric,
	}

	for _, opt := range opts {
		if err := opt(&metric); err != nil {
			return Metric{}, err
		}
	}

	return metric, nil
}

// WithValue Опция конструктора метрики - инициализация значения метрики
// Подходит для всех типов метрик, значение data конвертируется.
func WithValue(data string) OptionsMetric {
	return func(metric *Metric) error {

		switch metric.MType {
		case GaugeType:

			val, err := strconv.ParseFloat(data, 64)
			if err != nil {
				return fmt.Errorf("could not create metric: %w", errs.ErrInvalidValue)
			}

			metric.Value = &val

		case CounterType:
			val, err := strconv.ParseInt(data, 10, 64)
			if err != nil {
				return fmt.Errorf("could not create metric: %w", errs.ErrInvalidValue)
			}
			metric.Delta = &val

		default:
			return fmt.Errorf("could not create metric: %w", errs.ErrUnknownType)
		}

		return nil
	}
}

// WithValueFloat Опция конструктора метрики - инициализация значения метрики
// Подходит для всех типов метрик, значение value конвертируется при необходимости в int64.
func WithValueFloat(value float64) OptionsMetric {
	return func(metric *Metric) error {

		switch metric.MType {
		case GaugeType:
			metric.Value = &value

		case CounterType:
			val := int64(value)
			metric.Delta = &val

		default:
			return fmt.Errorf("could not change data metric: %w", errs.ErrUnknownType)
		}

		return nil
	}
}

// WithValueInt Опция конструктора метрики - инициализация значения метрики
// Подходит для всех типов метрик, значение value конвертируется при необходимости в float64.
func WithValueInt(value int64) OptionsMetric {
	return func(metric *Metric) error {

		switch metric.MType {
		case GaugeType:
			val := float64(value)
			metric.Value = &val

		case CounterType:
			metric.Delta = &value

		default:
			return fmt.Errorf("could not change data metric: %w", errs.ErrUnknownType)
		}

		return nil
	}
}

// Sign Подпись метрики
// Данные метрики преобразуются в строку формата <id>:<type>:<value>
// и при помощи алгоритка SHA256 и ключа key вычиляется хеш метрики
func (metric Metric) Sign(key []byte) (string, error) {

	if len(key) == 0 {
		return ``, nil
	}

	var src string

	switch metric.MType {
	case CounterType:
		if metric.Delta == nil {
			return ``, errs.ErrInvalidValue
		}

		src = fmt.Sprintf("%s:%s:%d",
			metric.ID,
			metric.MType,
			*metric.Delta)

	case GaugeType:
		if metric.Value == nil {
			return ``, errs.ErrInvalidValue
		}

		src = fmt.Sprintf("%s:%s:%f",
			metric.ID,
			metric.MType,
			*metric.Value)
	default:
		return ``, errs.ErrUnknownType
	}

	h := hmac.New(sha256.New, key)
	if _, err := h.Write([]byte(src)); err != nil {
		return ``, err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// Map Преобразование структуры метрики в map
// Возвращаемый map содержит ключи "type","name","value"
func (metric Metric) Map() map[string]string {

	data := make(map[string]string, 3)

	data["type"] = metric.MType
	data["name"] = metric.ID
	data["value"] = ""

	switch metric.MType {
	case GaugeType:
		if metric.Value != nil {
			data["value"] = strconv.FormatFloat(*metric.Value, 'f', -1, 64)
		}

	case CounterType:
		if metric.Delta != nil {
			data["value"] = strconv.FormatInt(*metric.Delta, 10)
		}
	}

	return data
}

// StringValue Преобразование значения метрики в строку
func (metric Metric) StringValue() string {
	switch metric.MType {
	case GaugeType:
		if metric.Value != nil {
			return fmt.Sprintf("%f", *metric.Value)
		}

	case CounterType:
		if metric.Delta != nil {
			return fmt.Sprintf("%d", *metric.Delta)
		}
	}

	return ``
}

// ShotString Данные метрики в виде строки в компактном виде
// Возвращаемая строка имеет формат: <type>/<id>/<value>
func (metric Metric) ShotString() string {
	builder := strings.Builder{}

	builder.WriteString(metric.MType)
	builder.WriteString(" / ")
	builder.WriteString(metric.ID)
	builder.WriteString(" / ")

	switch metric.MType {
	case GaugeType:
		if metric.Value != nil {
			builder.WriteString(fmt.Sprintf("%f", *metric.Value))
		}

	case CounterType:
		if metric.Delta != nil {
			builder.WriteString(fmt.Sprintf("%d", *metric.Delta))
		}
	}

	return builder.String()
}

// String Данные метрики в виде строки  развернутом виде
// Реализация интерфейса Stringer
func (metric Metric) String() string {

	builder := strings.Builder{}

	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("\t ID: %s\n", metric.ID))
	builder.WriteString(fmt.Sprintf("\t TYPE: %s\n", metric.MType))
	builder.WriteString(fmt.Sprintf("\t HASH: %s\n", metric.Hash))

	if metric.Delta != nil {
		builder.WriteString(fmt.Sprintf("\t DELTA: %d\n", *metric.Delta))
	} else {
		builder.WriteString("\t DELTA: nil\n")
	}

	if metric.Value != nil {
		builder.WriteString(fmt.Sprintf("\t VALUE: %f\n", *metric.Value))
	} else {
		builder.WriteString("\t VALUE: nil\n")
	}

	return builder.String()
}
