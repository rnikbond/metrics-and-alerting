package main

import (
	"context"
	"fmt"
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
	mutex       sync.Mutex
	pollCount   int64
	randomValue float64
	data        map[string]float64
}

func floatToString(value float64) string {
	return strconv.FormatFloat(value, 'f', 3, 64)
}

func int64ToString(value int64) string {
	return strconv.FormatInt(value, 10)
}

func updateMetrics(monitor *MetricsMonitor) {

	monitor.mutex.Lock()
	defer monitor.mutex.Unlock()

	var memstats runtime.MemStats
	runtime.ReadMemStats(&memstats)

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

	generator := rand.New(rand.NewSource(time.Now().UnixNano()))
	monitor.randomValue = generator.Float64()
}

func reportMetric(client *http.Client, typeMetric string, nameMetric string, valueMetric string) {

	urlMetric := urlServer + typeMetric + "/" + nameMetric + "/" + valueMetric
	req, err := http.NewRequest(http.MethodPost, urlMetric, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	req.Header.Set("Content-Type", "text/plain; charset=utf-8")

	resp, err := client.Do(req)

	if err != nil {
		fmt.Println(err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println(resp.Status)
	}
}

func reportMetrics(monitor *MetricsMonitor) {

	monitor.mutex.Lock()
	defer monitor.mutex.Unlock()

	client := &http.Client{}

	for metricName, metricValue := range monitor.data {
		reportMetric(client, guageType, metricName, floatToString(metricValue))
	}

	reportMetric(client, counterType, "PollCount", int64ToString(monitor.pollCount))
	reportMetric(client, guageType, "RandomValue", floatToString(monitor.randomValue))
}

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

func regularReportMetrics(ctx context.Context, waitGroup *sync.WaitGroup, monitor *MetricsMonitor) {
	waitGroup.Add(1)

	for {
		select {
		case <-time.After(reportInterval * time.Second):
			reportMetrics(monitor)
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
