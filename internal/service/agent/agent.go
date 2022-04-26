package agent

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	"metrics-and-alerting/internal/storage"

	"github.com/go-resty/resty/v2"
)

type Service interface {
	Start(ctx context.Context, wg *sync.WaitGroup)

	regularReport(ctx context.Context, wg *sync.WaitGroup)
	regularUpdate(ctx context.Context, wg *sync.WaitGroup)

	reportAll(ctx context.Context) error
	report(ctx context.Context, nameMetric, valueMetric, typeMetric string) error

	updateAll()
}

type Agent struct {
	ServerURL      string
	PollInterval   time.Duration
	ReportInterval time.Duration
	Metrics        storage.Metrics
}

func float64ToString(value float64) string {
	return strconv.FormatFloat(value, 'f', 3, 64)
}

func int64ToString(value int64) string {
	return strconv.FormatInt(value, 10)
}

func uint64ToString(value uint64) string {
	return strconv.FormatUint(value, 10)
}

// Запуск агента для сбора и отправки метрик
func (agent *Agent) Start(ctx context.Context, wg *sync.WaitGroup) {
	// запуск горутины для обновления метрик
	go agent.regularUpdate(ctx, wg)

	// запуск горутины для отправки метрик
	go agent.regularReport(ctx, wg)

}

// Отправка метрик с частотой агента
func (agent *Agent) regularReport(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)

	for {
		select {
		case <-time.After(agent.ReportInterval * time.Second):
			agent.reportAll(ctx)
		case <-ctx.Done():
			wg.Done()
			return
		}
	}
}

// Обновление метрик с частотой агента
func (agent *Agent) regularUpdate(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	agent.updateAll()

	for {
		select {
		case <-time.After(agent.PollInterval * time.Second):
			agent.updateAll()
		case <-ctx.Done():
			wg.Done()
			return
		}
	}
}

// Отправление всех метрик
func (agent *Agent) reportAll(ctx context.Context) {

	client := resty.New()

	for metricName, metricValue := range agent.Metrics.GetGauges() {
		agent.report(ctx, client, metricName, float64ToString(metricValue), storage.GaugeType)
	}

	for metricName, metricValue := range agent.Metrics.GetCounters() {
		agent.report(ctx, client, metricName, int64ToString(metricValue), storage.CounterType)
	}

	agent.Metrics.Set("PollCount", "0", storage.CounterType)
}

// Обновление метрики
func (agent *Agent) report(ctx context.Context, client *resty.Client, nameMetric, valueMetric, typeMetric string) error {

	if len(nameMetric) < 1 {
		return errors.New("name metric can not be empty")
	}

	if len(valueMetric) < 1 {
		return errors.New("value metric can not be empty")
	}

	if len(typeMetric) < 1 {
		return errors.New("type metric can not be empty")
	}

	resp, err := client.R().SetPathParams(map[string]string{
		"type":  typeMetric,
		"name":  nameMetric,
		"value": valueMetric,
	}).Post(agent.ServerURL + "/{type}/{name}/{value}")

	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		respBody := resp.Body()
		return errors.New("failed update metric: " + resp.Status() + ". " + string(respBody))
	}

	return nil
}

// Обновление всех метрик
func (agent *Agent) updateAll() {

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	generator := rand.New(rand.NewSource(time.Now().UnixNano()))

	agent.Metrics.Set("RandomValue", float64ToString(generator.Float64()), storage.GaugeType)
	agent.Metrics.Set("Alloc", uint64ToString(ms.Alloc), storage.GaugeType)
	agent.Metrics.Set("BuckHashSys", uint64ToString(ms.BuckHashSys), storage.GaugeType)
	agent.Metrics.Set("Frees", uint64ToString(ms.Frees), storage.GaugeType)
	agent.Metrics.Set("GCCPUFraction", float64ToString(ms.GCCPUFraction), storage.GaugeType)
	agent.Metrics.Set("GCSys", uint64ToString(ms.GCSys), storage.GaugeType)
	agent.Metrics.Set("HeapAlloc", uint64ToString(ms.HeapAlloc), storage.GaugeType)
	agent.Metrics.Set("HeapIdle", uint64ToString(ms.HeapIdle), storage.GaugeType)
	agent.Metrics.Set("HeapInuse", uint64ToString(ms.HeapInuse), storage.GaugeType)
	agent.Metrics.Set("HeapObjects", uint64ToString(ms.HeapObjects), storage.GaugeType)
	agent.Metrics.Set("HeapReleased", uint64ToString(ms.HeapReleased), storage.GaugeType)
	agent.Metrics.Set("HeapSys", uint64ToString(ms.HeapSys), storage.GaugeType)
	agent.Metrics.Set("LastGC", uint64ToString(ms.LastGC), storage.GaugeType)
	agent.Metrics.Set("Lookups", uint64ToString(ms.Lookups), storage.GaugeType)
	agent.Metrics.Set("MCacheInuse", uint64ToString(ms.MCacheInuse), storage.GaugeType)
	agent.Metrics.Set("MCacheSys", uint64ToString(ms.MCacheSys), storage.GaugeType)
	agent.Metrics.Set("MSpanInuse", uint64ToString(ms.MSpanInuse), storage.GaugeType)
	agent.Metrics.Set("MSpanSys", uint64ToString(ms.MSpanSys), storage.GaugeType)
	agent.Metrics.Set("Mallocs", uint64ToString(ms.Mallocs), storage.GaugeType)
	agent.Metrics.Set("NextGC", uint64ToString(ms.NextGC), storage.GaugeType)
	agent.Metrics.Set("NumForcedGC", uint64ToString(uint64(ms.NumForcedGC)), storage.GaugeType)
	agent.Metrics.Set("NumGC", uint64ToString(uint64(ms.NumGC)), storage.GaugeType)
	agent.Metrics.Set("OtherSys", uint64ToString(ms.OtherSys), storage.GaugeType)
	agent.Metrics.Set("PauseTotalNs", uint64ToString(ms.PauseTotalNs), storage.GaugeType)
	agent.Metrics.Set("StackInuse", uint64ToString(ms.StackInuse), storage.GaugeType)
	agent.Metrics.Set("StackSys", uint64ToString(ms.StackSys), storage.GaugeType)
	agent.Metrics.Set("Sys", uint64ToString(ms.Sys), storage.GaugeType)
	agent.Metrics.Set("TotalAlloc", uint64ToString(ms.TotalAlloc), storage.GaugeType)
	agent.Metrics.Add("PollCount", int64ToString(1), storage.CounterType)
}
