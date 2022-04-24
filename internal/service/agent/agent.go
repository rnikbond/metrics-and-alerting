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

type AgentMetrics interface {
	Start(ctx context.Context, wg *sync.WaitGroup)

	regularReport(ctx context.Context, wg *sync.WaitGroup)
	regularUpdate(ctx context.Context, wg *sync.WaitGroup)

	reportAll(ctx context.Context) error
	report(ctx context.Context, nameMetric, valueMetric, typeMetric string) error

	updateAll()
}

type AgentMeticsData struct {
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
func (agent *AgentMeticsData) Start(ctx context.Context, wg *sync.WaitGroup) {
	// запуск горутины для обновления метрик
	go agent.regularUpdate(ctx, wg)

	// запуск горутины для отправки метрик
	go agent.regularReport(ctx, wg)

}

// Отправка метрик с частотой агента
func (agent *AgentMeticsData) regularReport(ctx context.Context, wg *sync.WaitGroup) {
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
func (agent *AgentMeticsData) regularUpdate(ctx context.Context, wg *sync.WaitGroup) {
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
func (agent *AgentMeticsData) reportAll(ctx context.Context) {

	client := resty.New()

	for metricName, metricValue := range agent.Metrics.GetGauges() {
		agent.report(ctx, client, metricName, float64ToString(metricValue), storage.GuageType)
	}

	for metricName, metricValue := range agent.Metrics.GetCounters() {
		agent.report(ctx, client, metricName, int64ToString(metricValue), storage.CounterType)
	}

	agent.Metrics.Set("PollCount", "0", storage.CounterType)
}

// Обновление метрики
func (agent *AgentMeticsData) report(ctx context.Context, client *resty.Client, nameMetric, valueMetric, typeMetric string) error {

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
func (agent *AgentMeticsData) updateAll() {

	var memstats runtime.MemStats
	runtime.ReadMemStats(&memstats)

	generator := rand.New(rand.NewSource(time.Now().UnixNano()))

	agent.Metrics.Set("RandomValue", float64ToString(generator.Float64()), storage.GuageType)
	agent.Metrics.Set("Alloc", uint64ToString(memstats.Alloc), storage.GuageType)
	agent.Metrics.Set("BuckHashSys", uint64ToString(memstats.BuckHashSys), storage.GuageType)
	agent.Metrics.Set("Frees", uint64ToString(memstats.Frees), storage.GuageType)
	agent.Metrics.Set("GCCPUFraction", float64ToString(memstats.GCCPUFraction), storage.GuageType)
	agent.Metrics.Set("GCSys", uint64ToString(memstats.GCSys), storage.GuageType)
	agent.Metrics.Set("HeapAlloc", uint64ToString(memstats.HeapAlloc), storage.GuageType)
	agent.Metrics.Set("HeapIdle", uint64ToString(memstats.HeapIdle), storage.GuageType)
	agent.Metrics.Set("HeapInuse", uint64ToString(memstats.HeapInuse), storage.GuageType)
	agent.Metrics.Set("HeapObjects", uint64ToString(memstats.HeapObjects), storage.GuageType)
	agent.Metrics.Set("HeapReleased", uint64ToString(memstats.HeapReleased), storage.GuageType)
	agent.Metrics.Set("HeapSys", uint64ToString(memstats.HeapSys), storage.GuageType)
	agent.Metrics.Set("LastGC", uint64ToString(memstats.LastGC), storage.GuageType)
	agent.Metrics.Set("Lookups", uint64ToString(memstats.Lookups), storage.GuageType)
	agent.Metrics.Set("MCacheInuse", uint64ToString(memstats.MCacheInuse), storage.GuageType)
	agent.Metrics.Set("MCacheSys", uint64ToString(memstats.MCacheSys), storage.GuageType)
	agent.Metrics.Set("MSpanInuse", uint64ToString(memstats.MSpanInuse), storage.GuageType)
	agent.Metrics.Set("MSpanSys", uint64ToString(memstats.MSpanSys), storage.GuageType)
	agent.Metrics.Set("Mallocs", uint64ToString(memstats.Mallocs), storage.GuageType)
	agent.Metrics.Set("NextGC", uint64ToString(memstats.NextGC), storage.GuageType)
	agent.Metrics.Set("NumForcedGC", uint64ToString(uint64(memstats.NumForcedGC)), storage.GuageType)
	agent.Metrics.Set("NumGC", uint64ToString(uint64(memstats.NumGC)), storage.GuageType)
	agent.Metrics.Set("OtherSys", uint64ToString(memstats.OtherSys), storage.GuageType)
	agent.Metrics.Set("PauseTotalNs", uint64ToString(memstats.PauseTotalNs), storage.GuageType)
	agent.Metrics.Set("StackInuse", uint64ToString(memstats.StackInuse), storage.GuageType)
	agent.Metrics.Set("StackSys", uint64ToString(memstats.StackSys), storage.GuageType)
	agent.Metrics.Set("Sys", uint64ToString(memstats.Sys), storage.GuageType)
	agent.Metrics.Set("TotalAlloc", uint64ToString(memstats.TotalAlloc), storage.GuageType)
	agent.Metrics.Add("PollCount", int64ToString(1), storage.CounterType)
}
