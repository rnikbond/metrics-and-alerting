package reporter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"metrics-and-alerting/internal/storage"

	"github.com/go-resty/resty/v2"
)

const (
	ReportAsURL       = "URL"
	ReportAsJSON      = "JSON"
	ReportAsBatchJSON = "BatchJSON"
)

type Reporter struct {
	addr    string
	storage storage.Repository
}

func NewReporter(addr string, storage storage.Repository) *Reporter {
	return &Reporter{
		addr:    addr,
		storage: storage,
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

	metrics, errStorage := r.storage.GetSlice()
	if errStorage != nil {
		return fmt.Errorf("could not report metrics: %v", errStorage)
	}

	client := resty.New()

	for _, m := range metrics {

		client.R()
		resp, err := client.R().
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

	metrics, errStorage := r.storage.GetSlice()
	if errStorage != nil {
		return fmt.Errorf("could not report metrics: %v", errStorage)
	}

	client := resty.New()

	for _, m := range metrics {
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

	metrics, errStorage := r.storage.GetSlice()
	if errStorage != nil {
		return fmt.Errorf("could not report metrics: %v", errStorage)
	}

	data, err := json.Marshal(&metrics)
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
