package agent

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	"metrics-and-alerting/internal/storage"
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
	UrlServer      string
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

	client := &http.Client{}

	for metricName, metricValue := range agent.Metrics.GetGauges() {
		agent.report(ctx, client, metricName, float64ToString(metricValue), storage.GuageType)
	}

	for metricName, metricValue := range agent.Metrics.GetCounters() {
		agent.report(ctx, client, metricName, int64ToString(metricValue), storage.CounterType)
	}
}

// Обновление метрики
func (agent *AgentMeticsData) report(ctx context.Context, client *http.Client, nameMetric, valueMetric, typeMetric string) error {

	if len(nameMetric) < 1 {
		return errors.New("name metric can not be empty")
	}

	if len(valueMetric) < 1 {
		return errors.New("value metric can not be empty")
	}

	if len(typeMetric) < 1 {
		return errors.New("type metric can not be empty")
	}

	urlMetric := agent.UrlServer + string(typeMetric) + "/" + nameMetric + "/" + valueMetric
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlMetric, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "text/plain; charset=utf-8")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		} else {
			return errors.New("failed update metric: " + resp.Status + ". Reason: " + string(respBody))
		}
	}

	return nil
}

// Обновление всех метрик
func (agent *AgentMeticsData) updateAll() {

	var memstats runtime.MemStats
	runtime.ReadMemStats(&memstats)

	generator := rand.New(rand.NewSource(time.Now().UnixNano()))

	agent.Metrics.Update("RandomValue", float64ToString(generator.Float64()), storage.GuageType)
	agent.Metrics.Update("Alloc", uint64ToString(memstats.Alloc), storage.GuageType)
	agent.Metrics.Update("BuckHashSys", uint64ToString(memstats.BuckHashSys), storage.GuageType)
	agent.Metrics.Update("Frees", uint64ToString(memstats.Frees), storage.GuageType)
	agent.Metrics.Update("GCCPUFraction", float64ToString(memstats.GCCPUFraction), storage.GuageType)
	agent.Metrics.Update("GCSys", uint64ToString(memstats.GCSys), storage.GuageType)
	agent.Metrics.Update("HeapAlloc", uint64ToString(memstats.HeapAlloc), storage.GuageType)
	agent.Metrics.Update("HeapIdle", uint64ToString(memstats.HeapIdle), storage.GuageType)
	agent.Metrics.Update("HeapInuse", uint64ToString(memstats.HeapInuse), storage.GuageType)
	agent.Metrics.Update("HeapObjects", uint64ToString(memstats.HeapObjects), storage.GuageType)
	agent.Metrics.Update("HeapReleased", uint64ToString(memstats.HeapReleased), storage.GuageType)
	agent.Metrics.Update("HeapSys", uint64ToString(memstats.HeapSys), storage.GuageType)
	agent.Metrics.Update("LastGC", uint64ToString(memstats.LastGC), storage.GuageType)
	agent.Metrics.Update("Lookups", uint64ToString(memstats.Lookups), storage.GuageType)
	agent.Metrics.Update("MCacheInuse", uint64ToString(memstats.MCacheInuse), storage.GuageType)
	agent.Metrics.Update("MCacheSys", uint64ToString(memstats.MCacheSys), storage.GuageType)
	agent.Metrics.Update("MSpanInuse", uint64ToString(memstats.MSpanInuse), storage.GuageType)
	agent.Metrics.Update("MSpanSys", uint64ToString(memstats.MSpanSys), storage.GuageType)
	agent.Metrics.Update("Mallocs", uint64ToString(memstats.Mallocs), storage.GuageType)
	agent.Metrics.Update("NextGC", uint64ToString(memstats.NextGC), storage.GuageType)
	agent.Metrics.Update("NumForcedGC", uint64ToString(uint64(memstats.NumForcedGC)), storage.GuageType)
	agent.Metrics.Update("NumGC", uint64ToString(uint64(memstats.NumGC)), storage.GuageType)
	agent.Metrics.Update("OtherSys", uint64ToString(memstats.OtherSys), storage.GuageType)
	agent.Metrics.Update("PauseTotalNs", uint64ToString(memstats.PauseTotalNs), storage.GuageType)
	agent.Metrics.Update("StackInuse", uint64ToString(memstats.StackInuse), storage.GuageType)
	agent.Metrics.Update("StackSys", uint64ToString(memstats.StackSys), storage.GuageType)
	agent.Metrics.Update("Sys", uint64ToString(memstats.Sys), storage.GuageType)
	agent.Metrics.Update("TotalAlloc", uint64ToString(memstats.TotalAlloc), storage.GuageType)
	agent.Metrics.Update(storage.CounterName, uint64ToString(memstats.TotalAlloc), storage.CounterType)
}
