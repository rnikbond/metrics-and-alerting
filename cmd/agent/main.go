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

	"metrics-and-alerting/internal/storage"
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

func uint64ToString(value uint64) string {
	return strconv.FormatUint(value, 10)
}

// Обновление всех метрик
func updateMetrics(metrics storage.Metrics) {

	var memstats runtime.MemStats
	runtime.ReadMemStats(&memstats)

	generator := rand.New(rand.NewSource(time.Now().UnixNano()))

	metrics.Update("RandomValue", float64ToString(generator.Float64()), storage.GuageType)
	metrics.Update("Alloc", uint64ToString(memstats.Alloc), storage.GuageType)
	metrics.Update("BuckHashSys", uint64ToString(memstats.BuckHashSys), storage.GuageType)
	metrics.Update("Frees", uint64ToString(memstats.Frees), storage.GuageType)
	metrics.Update("GCCPUFraction", float64ToString(memstats.GCCPUFraction), storage.GuageType)
	metrics.Update("GCSys", uint64ToString(memstats.GCSys), storage.GuageType)
	metrics.Update("HeapAlloc", uint64ToString(memstats.HeapAlloc), storage.GuageType)
	metrics.Update("HeapIdle", uint64ToString(memstats.HeapIdle), storage.GuageType)
	metrics.Update("HeapInuse", uint64ToString(memstats.HeapInuse), storage.GuageType)
	metrics.Update("HeapObjects", uint64ToString(memstats.HeapObjects), storage.GuageType)
	metrics.Update("HeapReleased", uint64ToString(memstats.HeapReleased), storage.GuageType)
	metrics.Update("HeapSys", uint64ToString(memstats.HeapSys), storage.GuageType)
	metrics.Update("LastGC", uint64ToString(memstats.LastGC), storage.GuageType)
	metrics.Update("Lookups", uint64ToString(memstats.Lookups), storage.GuageType)
	metrics.Update("MCacheInuse", uint64ToString(memstats.MCacheInuse), storage.GuageType)
	metrics.Update("MCacheSys", uint64ToString(memstats.MCacheSys), storage.GuageType)
	metrics.Update("MSpanInuse", uint64ToString(memstats.MSpanInuse), storage.GuageType)
	metrics.Update("MSpanSys", uint64ToString(memstats.MSpanSys), storage.GuageType)
	metrics.Update("Mallocs", uint64ToString(memstats.Mallocs), storage.GuageType)
	metrics.Update("NextGC", uint64ToString(memstats.NextGC), storage.GuageType)
	metrics.Update("NumForcedGC", uint64ToString(uint64(memstats.NumForcedGC)), storage.GuageType)
	metrics.Update("NumGC", uint64ToString(uint64(memstats.NumGC)), storage.GuageType)
	metrics.Update("OtherSys", uint64ToString(memstats.OtherSys), storage.GuageType)
	metrics.Update("PauseTotalNs", uint64ToString(memstats.PauseTotalNs), storage.GuageType)
	metrics.Update("StackInuse", uint64ToString(memstats.StackInuse), storage.GuageType)
	metrics.Update("StackSys", uint64ToString(memstats.StackSys), storage.GuageType)
	metrics.Update("Sys", uint64ToString(memstats.Sys), storage.GuageType)
	metrics.Update("TotalAlloc", uint64ToString(memstats.TotalAlloc), storage.GuageType)
	metrics.Update(storage.CounterName, uint64ToString(memstats.TotalAlloc), storage.CounterType)
}

// Отправка запроса серверу на обновление метрики
func reportMetric(ctx context.Context, nameMetric, valueMetric, typeMetric string) {

	urlMetric := urlServer + string(typeMetric) + "/" + nameMetric + "/" + valueMetric
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlMetric, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	req.Header.Set("Content-Type", "text/plain; charset=utf-8")

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
	}
}

// Отправка всех метрик серверу
func reportMetrics(ctx context.Context, metrics storage.Metrics) {

	for metricName, metricValue := range metrics.GetGauges() {
		reportMetric(ctx, metricName, float64ToString(metricValue), storage.GuageType)
	}

	for metricName, metricValue := range metrics.GetCounters() {
		reportMetric(ctx, metricName, int64ToString(metricValue), storage.CounterType)
	}
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
