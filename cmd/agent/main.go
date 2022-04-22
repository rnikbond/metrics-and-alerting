package main

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	storage "github.com/rnikbond/metrics-and-alerting/internal/storage"
)

const (
	pollInterval   time.Duration = 2
	reportInterval time.Duration = 10
)

const (
	urlServer string = "http://127.0.0.1:8080/update/"
)

func float64ToString(value float64) string {
	return strconv.FormatFloat(value, 'f', 3, 64)
}

func int64ToString(value int64) string {
	return strconv.FormatInt(value, 10)
}

// Обновление всех метрик
func updateMetrics(metrics storage.Metrics) {

	var memstats runtime.MemStats
	runtime.ReadMemStats(&memstats)

	generator := rand.New(rand.NewSource(time.Now().UnixNano()))

	metrics.SetMetricGauge("RandomValue", generator.Float64())
	metrics.SetMetricGauge("Alloc", float64(memstats.Alloc))
	metrics.SetMetricGauge("BuckHashSys", float64(memstats.BuckHashSys))
	metrics.SetMetricGauge("Frees", float64(memstats.Frees))
	metrics.SetMetricGauge("GCCPUFraction", float64(memstats.GCCPUFraction))
	metrics.SetMetricGauge("GCSys", float64(memstats.GCSys))
	metrics.SetMetricGauge("HeapAlloc", float64(memstats.HeapAlloc))
	metrics.SetMetricGauge("HeapIdle", float64(memstats.HeapIdle))
	metrics.SetMetricGauge("HeapInuse", float64(memstats.HeapInuse))
	metrics.SetMetricGauge("HeapObjects", float64(memstats.HeapObjects))
	metrics.SetMetricGauge("HeapReleased", float64(memstats.HeapReleased))
	metrics.SetMetricGauge("HeapSys", float64(memstats.HeapSys))
	metrics.SetMetricGauge("LastGC", float64(memstats.LastGC))
	metrics.SetMetricGauge("Lookups", float64(memstats.Lookups))
	metrics.SetMetricGauge("MCacheInuse", float64(memstats.MCacheInuse))
	metrics.SetMetricGauge("MCacheSys", float64(memstats.MCacheSys))
	metrics.SetMetricGauge("MSpanInuse", float64(memstats.MSpanInuse))
	metrics.SetMetricGauge("MSpanSys", float64(memstats.MSpanSys))
	metrics.SetMetricGauge("Mallocs", float64(memstats.Mallocs))
	metrics.SetMetricGauge("NextGC", float64(memstats.NextGC))
	metrics.SetMetricGauge("NumForcedGC", float64(memstats.NumForcedGC))
	metrics.SetMetricGauge("NumGC", float64(memstats.NumGC))
	metrics.SetMetricGauge("OtherSys", float64(memstats.OtherSys))
	metrics.SetMetricGauge("PauseTotalNs", float64(memstats.PauseTotalNs))
	metrics.SetMetricGauge("StackInuse", float64(memstats.StackInuse))
	metrics.SetMetricGauge("StackSys", float64(memstats.StackSys))
	metrics.SetMetricGauge("Sys", float64(memstats.Sys))
	metrics.SetMetricGauge("TotalAlloc", float64(memstats.TotalAlloc))

	metrics.AppendToMetricCounter(1)
}

// Отправка запроса серверу на обновление метрики
func reportMetric(ctx context.Context, typeMetric string, nameMetric string, valueMetric string) {

	urlMetric := urlServer + typeMetric + "/" + nameMetric + "/" + valueMetric
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlMetric, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println(err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("failed update metric: %s. Reason: %s", resp.Status, err.Error())
		} else {
			fmt.Printf("failed update metric: %s. Reason: %s", resp.Status, string(respBody))
		}
	} else {
		fmt.Printf("sussecc update metric: %s\n", nameMetric)
	}
}

// Отправка всех метрик серверу
func reportMetrics(ctx context.Context, metrics storage.Metrics) {

	for metricName, metricValue := range metrics.GetMetricsGauge() {
		reportMetric(ctx, storage.GuageType, metricName, float64ToString(metricValue))
	}

	reportMetric(ctx, storage.CounterType, "PollCount", int64ToString(metrics.GetMetricCounter()))
}

// Обновление метрик с заданной частотой
func regularUpdateMetrics(ctx context.Context, waitGroup *sync.WaitGroup, metrics storage.Metrics) {
	waitGroup.Add(1)
	updateMetrics(metrics)

	for {
		select {
		case <-time.After(pollInterval * time.Second):
			updateMetrics(metrics)
		case <-ctx.Done():
			waitGroup.Done()
			return
		}
	}
}

// Отправка метрик серверу с заданной частотой
func regularReportMetrics(ctx context.Context, waitGroup *sync.WaitGroup, metrics storage.Metrics) {
	waitGroup.Add(1)

	for {
		select {
		case <-time.After(reportInterval * time.Second):
			reportMetrics(ctx, metrics)
		case <-ctx.Done():
			waitGroup.Done()
			return
		}
	}
}

func main() {

	var waitGroup sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	monitor := storage.MetricsData{}

	// Запуск горутины для обновления метрик
	go regularUpdateMetrics(ctx, &waitGroup, &monitor)
	// Запуск горутины для отправки метрик
	go regularReportMetrics(ctx, &waitGroup, &monitor)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	<-sigChan

	cancel()
	waitGroup.Wait()
}
