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
)

const (
	pollInterval   time.Duration = 2
	reportInterval time.Duration = 10
)

const (
	urlServer   string = "http://127.0.0.1:8080/update/"
	guageType   string = "gauge"
	counterType string = "counter"
)

type MetricsMonitor struct {
	mutex     sync.Mutex
	pollCount int64
	data      map[string]float64
}

func float64ToString(value float64) string {
	return strconv.FormatFloat(value, 'f', 3, 64)
}

func int64ToString(value int64) string {
	return strconv.FormatInt(value, 10)
}

// Обновление всех метрик
func updateMetrics(monitor *MetricsMonitor) {

	monitor.mutex.Lock()
	defer monitor.mutex.Unlock()

	var memstats runtime.MemStats
	runtime.ReadMemStats(&memstats)

	generator := rand.New(rand.NewSource(time.Now().UnixNano()))

	monitor.data["RandomValue"] = generator.Float64()
	monitor.data["Alloc"] = float64(memstats.Alloc)
	monitor.data["BuckHashSys"] = float64(memstats.BuckHashSys)
	monitor.data["Frees"] = float64(memstats.Frees)
	monitor.data["GCCPUFraction"] = float64(memstats.GCCPUFraction)
	monitor.data["GCSys"] = float64(memstats.GCSys)
	monitor.data["HeapAlloc"] = float64(memstats.HeapAlloc)
	monitor.data["HeapIdle"] = float64(memstats.HeapIdle)
	monitor.data["HeapInuse"] = float64(memstats.HeapInuse)
	monitor.data["HeapObjects"] = float64(memstats.HeapObjects)
	monitor.data["HeapReleased"] = float64(memstats.HeapReleased)
	monitor.data["HeapSys"] = float64(memstats.HeapSys)
	monitor.data["LastGC"] = float64(memstats.LastGC)
	monitor.data["Lookups"] = float64(memstats.Lookups)
	monitor.data["MCacheInuse"] = float64(memstats.MCacheInuse)
	monitor.data["MCacheSys"] = float64(memstats.MCacheSys)
	monitor.data["MSpanInuse"] = float64(memstats.MSpanInuse)
	monitor.data["MSpanSys"] = float64(memstats.MSpanSys)
	monitor.data["Mallocs"] = float64(memstats.Mallocs)
	monitor.data["NextGC"] = float64(memstats.NextGC)
	monitor.data["NumForcedGC"] = float64(memstats.NumForcedGC)
	monitor.data["NumGC"] = float64(memstats.NumGC)
	monitor.data["OtherSys"] = float64(memstats.OtherSys)
	monitor.data["PauseTotalNs"] = float64(memstats.PauseTotalNs)
	monitor.data["StackInuse"] = float64(memstats.StackInuse)
	monitor.data["StackSys"] = float64(memstats.StackSys)
	monitor.data["Sys"] = float64(memstats.Sys)
	monitor.data["TotalAlloc"] = float64(memstats.TotalAlloc)

	monitor.pollCount++
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
func reportMetrics(ctx context.Context, monitor *MetricsMonitor) {

	monitor.mutex.Lock()
	defer monitor.mutex.Unlock()

	for metricName, metricValue := range monitor.data {
		reportMetric(ctx, guageType, metricName, float64ToString(metricValue))
	}

	reportMetric(ctx, counterType, "PollCount", int64ToString(monitor.pollCount))
}

// Обновление метрик с заданной частотой
func regularUpdateMetrics(ctx context.Context, waitGroup *sync.WaitGroup, monitor *MetricsMonitor) {
	waitGroup.Add(1)
	updateMetrics(monitor)

	for {
		select {
		case <-time.After(pollInterval * time.Second):
			updateMetrics(monitor)
		case <-ctx.Done():
			waitGroup.Done()
			return
		}
	}
}

// Отправка метрик серверу с заданной частотой
func regularReportMetrics(ctx context.Context, waitGroup *sync.WaitGroup, monitor *MetricsMonitor) {
	waitGroup.Add(1)

	for {
		select {
		case <-time.After(reportInterval * time.Second):
			reportMetrics(ctx, monitor)
		case <-ctx.Done():
			waitGroup.Done()
			return
		}
	}
}

func main() {

	var waitGroup sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	monitor := MetricsMonitor{data: make(map[string]float64)}

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
