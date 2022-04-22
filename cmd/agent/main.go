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

	metrics.SetValueGaugeType("RandomValue", generator.Float64())
	metrics.SetValueGaugeType("Alloc", float64(memstats.Alloc))
	metrics.SetValueGaugeType("BuckHashSys", float64(memstats.BuckHashSys))
	metrics.SetValueGaugeType("Frees", float64(memstats.Frees))
	metrics.SetValueGaugeType("GCCPUFraction", float64(memstats.GCCPUFraction))
	metrics.SetValueGaugeType("GCSys", float64(memstats.GCSys))
	metrics.SetValueGaugeType("HeapAlloc", float64(memstats.HeapAlloc))
	metrics.SetValueGaugeType("HeapIdle", float64(memstats.HeapIdle))
	metrics.SetValueGaugeType("HeapInuse", float64(memstats.HeapInuse))
	metrics.SetValueGaugeType("HeapObjects", float64(memstats.HeapObjects))
	metrics.SetValueGaugeType("HeapReleased", float64(memstats.HeapReleased))
	metrics.SetValueGaugeType("HeapSys", float64(memstats.HeapSys))
	metrics.SetValueGaugeType("LastGC", float64(memstats.LastGC))
	metrics.SetValueGaugeType("Lookups", float64(memstats.Lookups))
	metrics.SetValueGaugeType("MCacheInuse", float64(memstats.MCacheInuse))
	metrics.SetValueGaugeType("MCacheSys", float64(memstats.MCacheSys))
	metrics.SetValueGaugeType("MSpanInuse", float64(memstats.MSpanInuse))
	metrics.SetValueGaugeType("MSpanSys", float64(memstats.MSpanSys))
	metrics.SetValueGaugeType("Mallocs", float64(memstats.Mallocs))
	metrics.SetValueGaugeType("NextGC", float64(memstats.NextGC))
	metrics.SetValueGaugeType("NumForcedGC", float64(memstats.NumForcedGC))
	metrics.SetValueGaugeType("NumGC", float64(memstats.NumGC))
	metrics.SetValueGaugeType("OtherSys", float64(memstats.OtherSys))
	metrics.SetValueGaugeType("PauseTotalNs", float64(memstats.PauseTotalNs))
	metrics.SetValueGaugeType("StackInuse", float64(memstats.StackInuse))
	metrics.SetValueGaugeType("StackSys", float64(memstats.StackSys))
	metrics.SetValueGaugeType("Sys", float64(memstats.Sys))
	metrics.SetValueGaugeType("TotalAlloc", float64(memstats.TotalAlloc))

	metrics.AddValueCounterType(1)
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

	for metricName, metricValue := range metrics.ValuesGaugeType() {
		reportMetric(ctx, storage.GuageType, metricName, float64ToString(metricValue))
	}

	reportMetric(ctx, storage.CounterType, "PollCount", int64ToString(metrics.ValueCounterType()))
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
