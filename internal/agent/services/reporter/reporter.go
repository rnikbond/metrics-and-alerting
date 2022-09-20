package reporter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/metric"

	"github.com/go-resty/resty/v2"
)

const (
	ReportAsURL       = "URL"
	ReportAsJSON      = "JSON"
	ReportAsBatchJSON = "BatchJSON"
)

type (
	OptionReporter func(*Reporter)

	Reporter struct {
		addr    string
		signKey []byte
		storage storage.Repository
	}
)

func NewReporter(addr string, storage storage.Repository, opts ...OptionReporter) *Reporter {

	r := &Reporter{
		addr:    addr,
		storage: storage,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func WithSignKey(key []byte) OptionReporter {
	return func(reporter *Reporter) {
		reporter.signKey = key
	}
}

func (r Reporter) Report(ctx context.Context, reportType string) error {

	switch reportType {
	case ReportAsURL:
		if err := r.reportURL(ctx); err != nil {
			return err
		}

	case ReportAsJSON:
		if err := r.reportJSON(ctx); err != nil {
			return err
		}

	case ReportAsBatchJSON:
		if err := r.reportBatchJSON(ctx); err != nil {
			return err
		}

	default:
		return fmt.Errorf("could not report metrics: unknown report type")
	}

	return nil
}

// reportURL Отправка метрик через URL отдельными запросами
func (r Reporter) reportURL(ctx context.Context) error {

	metrics, errStorage := r.storage.GetBatch()
	if errStorage != nil {
		return fmt.Errorf("could not report metrics: %v", errStorage)
	}

	client := resty.New()

	for _, m := range metrics {

		client.R()
		resp, err := client.R().
			SetHeader("Content-Type", "text/plain").
			SetPathParams(m.Map()).
			SetContext(ctx).
			Post(r.addr + "/update/" + "{type}/{name}/{value}")

		if err != nil {
			return fmt.Errorf("could not send metrics as URL: %w", err)
		}

		if resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("server return no success status on update metrics as URL: %d", resp.StatusCode())
		}
	}

	return nil
}

// reportJSON Отправка метрик в виде JSON отдельными запросами
func (r Reporter) reportJSON(ctx context.Context) error {

	metrics, errStorage := r.storage.GetBatch()
	if errStorage != nil {
		return fmt.Errorf("could not report metrics: %v", errStorage)
	}

	client := resty.New()

	for _, m := range metrics {

		sign, errSign := m.Sign(r.signKey)
		if errSign != nil {
			return fmt.Errorf("could not report metrics: %v", errSign)
		}

		m.Hash = sign

		data, err := json.Marshal(&m)
		if err != nil {
			return fmt.Errorf("error encode metric to JSON: %w", err)
		}

		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(data).
			SetContext(ctx).
			Post(r.addr + "/update")

		if err != nil {
			return fmt.Errorf("could not send metrics as JSON: %w", err)
		}

		if resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("server return no success status on update metrics as JSON: %d", resp.StatusCode())
		}
	}

	return nil
}

// reportBatchJSON Отправка метрик в виде JSON одним запросом
func (r Reporter) reportBatchJSON(ctx context.Context) error {

	metrics, errStorage := r.storage.GetBatch()
	if errStorage != nil {
		return fmt.Errorf("could not report metrics: %v", errStorage)
	}

	// TODO :: Разобраться, как изменять текущий слайс, а не записывать в новый
	metricsSigned := make([]metric.Metric, len(metrics))

	for i, m := range metrics {

		sign, errSign := m.Sign(r.signKey)
		if errSign != nil {
			return fmt.Errorf("could not report metrics: %v", errSign)
		}

		m.Hash = sign
		metricsSigned[i] = m

	}

	data, err := json.Marshal(&metricsSigned)
	if err != nil {
		return fmt.Errorf("error encode metrics to JSON: %w", err)
	}

	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(data).
		SetContext(ctx).
		Post(r.addr + "/updates")

	if err != nil {
		return fmt.Errorf("could not send metrics as Batch-JSON: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server return no success status on update metrics as Batch-JSON: %d", resp.StatusCode())
	}

	return nil
}
