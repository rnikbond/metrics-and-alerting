package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	handler "metrics-and-alerting/internal/server/handlers"
	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/config"

	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type Agent struct {
	mu    sync.RWMutex
	cfg   config.Config
	store storage.Storager
}

func (agent *Agent) Init(cfg config.Config, store storage.Storager) {
	agent.cfg = cfg
	agent.store = store
}

// Start Запуск агента для сбора и отправки метрик
func (agent *Agent) Start(ctx context.Context) error {

	if agent.store == nil {
		return fmt.Errorf("storage is not initialize")
	}

	if !strings.Contains(agent.cfg.Addr, "http://") {
		agent.cfg.Addr = "http://" + agent.cfg.Addr
	}

	if err := agent.updateRuntime(); err != nil {
		log.Printf("error update runtime metrics on start agent: %v\n", err)
	}

	if err := agent.updateWorkload(); err != nil {
		log.Printf("error update workload metrics on start agent: %v\n", err)
	}

	// Обновление runtime метрик
	go agent.collectRuntime(ctx)
	// Обновление метрик загрузки памяти и процессора
	go agent.collectWorkload(ctx)

	// Отправка метрик на сервер
	go agent.report(ctx)

	return nil
}

// report Отправка метрик на сервер
func (agent *Agent) report(ctx context.Context) {

	for {
		select {
		case <-time.After(agent.cfg.ReportInterval):
			if err := agent.reportMetrics(ctx); err != nil {
				log.Printf("error report metrics: %v\n", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// regularCollectRuntime Обновление runtime метрик
func (agent *Agent) collectRuntime(ctx context.Context) {

	for {
		select {
		case <-time.After(agent.cfg.PollInterval):
			if err := agent.updateRuntime(); err != nil {
				log.Printf("error regular update runtime metrics: %v\n", err)
			}

		case <-ctx.Done():
			return
		}
	}
}

// regularCollectWorkload Обновление метрик CPU и памяти
func (agent *Agent) collectWorkload(ctx context.Context) {

	for {
		select {
		case <-time.After(agent.cfg.PollInterval):
			if err := agent.updateWorkload(); err != nil {
				log.Printf("error regular update workload metrics: %v\n", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// updateGauge Обновление метрик в хранилище
func (agent *Agent) updateGauge(gaugeMetrics map[string]interface{}) error {

	agent.mu.Lock()
	defer agent.mu.Unlock()

	for id, value := range gaugeMetrics {
		metric, err := storage.CreateMetric(storage.GaugeType, id, value)
		if err != nil {
			return err
		}

		if err := agent.store.Update(metric); err != nil {
			return err
		}
	}

	return nil
}

// Обновление всех runtime метрик
func (agent *Agent) updateRuntime() error {

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	generator := rand.New(rand.NewSource(time.Now().UnixNano()))

	gaugeMetrics := make(map[string]interface{})
	gaugeMetrics["RandomValue"] = generator.Float64()
	gaugeMetrics["Alloc"] = ms.Alloc
	gaugeMetrics["BuckHashSys"] = ms.BuckHashSys
	gaugeMetrics["Frees"] = ms.Frees
	gaugeMetrics["GCCPUFraction"] = ms.GCCPUFraction
	gaugeMetrics["GCSys"] = ms.GCSys
	gaugeMetrics["HeapAlloc"] = ms.HeapAlloc
	gaugeMetrics["HeapIdle"] = ms.HeapIdle
	gaugeMetrics["HeapInuse"] = ms.HeapInuse
	gaugeMetrics["HeapObjects"] = ms.HeapObjects
	gaugeMetrics["HeapReleased"] = ms.HeapReleased
	gaugeMetrics["HeapSys"] = ms.HeapSys
	gaugeMetrics["LastGC"] = ms.LastGC
	gaugeMetrics["Lookups"] = ms.Lookups
	gaugeMetrics["MCacheInuse"] = ms.MCacheInuse
	gaugeMetrics["MCacheSys"] = ms.MCacheSys
	gaugeMetrics["MSpanInuse"] = ms.MSpanInuse
	gaugeMetrics["MSpanSys"] = ms.MSpanSys
	gaugeMetrics["Mallocs"] = ms.Mallocs
	gaugeMetrics["NextGC"] = ms.NextGC
	gaugeMetrics["NumForcedGC"] = ms.NumForcedGC
	gaugeMetrics["NumGC"] = ms.NumGC
	gaugeMetrics["OtherSys"] = ms.OtherSys
	gaugeMetrics["PauseTotalNs"] = ms.PauseTotalNs
	gaugeMetrics["StackInuse"] = ms.StackInuse
	gaugeMetrics["StackSys"] = ms.StackSys
	gaugeMetrics["Sys"] = ms.Sys
	gaugeMetrics["TotalAlloc"] = ms.TotalAlloc

	if err := agent.updateGauge(gaugeMetrics); err != nil {
		return fmt.Errorf("error update runtime %s metric: %w", storage.GaugeType, err)
	}

	metric, err := storage.CreateMetric(storage.CounterType, "PollCount", 1)
	if err != nil {
		return fmt.Errorf("error create runtime %s metric: %w", storage.CounterType, err)
	}

	agent.mu.Lock()
	defer agent.mu.Unlock()

	if err := agent.store.Update(metric); err != nil {
		return fmt.Errorf("error update runtime %s metric: %w", storage.CounterType, err)
	}

	return nil
}

// Обновление всех метрик загрузки памяти и процессора
func (agent *Agent) updateWorkload() error {

	vm, errVM := mem.VirtualMemory()
	if errVM != nil {
		return errVM
	}

	gaugeMetrics := make(map[string]interface{})
	gaugeMetrics["TotalMemory"] = vm.Total
	gaugeMetrics["FreeMemory"] = vm.Free

	percentage, _ := cpu.Percent(0, true)
	for cpuID, cpuUtilization := range percentage {

		name := "CPUutilization" + fmt.Sprint(cpuID+1)
		gaugeMetrics[name] = cpuUtilization
	}

	if err := agent.updateGauge(gaugeMetrics); err != nil {
		return fmt.Errorf("error update workload metric: %w", err)
	}

	return nil
}

// Отправка всех метрик
func (agent *Agent) reportMetrics(ctx context.Context) error {

	agent.mu.RLock()
	defer agent.mu.RUnlock()

	client := resty.New()

	metrics := agent.store.GetData()

	switch agent.cfg.ReportType {
	case config.ReportBatchJSON:
		if err := agent.reportAsBatchJSON(ctx, client); err != nil {
			return err
		}

	case config.ReportJSON:
		for _, metric := range metrics {
			if err := agent.reportAsJSON(ctx, client, metric); err != nil {
				return err
			}
		}

	case config.ReportURL:
		for _, metric := range metrics {
			if err := agent.reportAsURL(ctx, client, metric); err != nil {
				return err
			}
		}
	}

	metric, _ := storage.CreateMetric(storage.CounterType, "PollCount", 0)
	if err := agent.store.Delete(metric); err != nil {
		return err
	}

	return nil
}

// reportAsURL Отправка метрик черех URL
func (agent *Agent) reportAsURL(ctx context.Context, client *resty.Client, metric storage.Metric) error {

	data, err := metric.Map()
	if err != nil {
		return err
	}

	resp, err := client.R().
		SetPathParams(data).
		SetContext(ctx).
		Post(agent.cfg.Addr + handler.PartURLUpdate + "/{type}/{name}/{value}")

	if err != nil {
		return fmt.Errorf("error create POST request to [%s]: %w",
			agent.cfg.Addr+handler.PartURLUpdate+"/{type}/{name}/{value}",
			err)
	}

	if resp.StatusCode() != http.StatusOK {
		respBody := resp.Body()
		return fmt.Errorf("error report URL metric: %s. Status: %d", string(respBody), resp.StatusCode())
	}

	return nil
}

// reportAsJSON Отправка по одной метрике метрик через JSON
func (agent *Agent) reportAsJSON(ctx context.Context, client *resty.Client, metric storage.Metric) error {

	data, err := json.Marshal(&metric)
	if err != nil {
		return fmt.Errorf("error encode metric: %w", err)
	}

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(data).
		SetContext(ctx).
		Post(agent.cfg.Addr + handler.PartURLUpdate)

	if err != nil {
		return fmt.Errorf("error create POST request to [%s]: %w", agent.cfg.Addr+handler.PartURLUpdate, err)
	}

	if resp.StatusCode() != http.StatusOK {
		respBody := resp.Body()
		return fmt.Errorf("error report metric as JSON: %s. Status: %d", string(respBody), resp.StatusCode())
	}

	return nil
}

// reportAsJSON Отправка всех метрик черех JSON
func (agent *Agent) reportAsBatchJSON(ctx context.Context, client *resty.Client) error {

	metrics := agent.store.GetData()
	data, err := json.Marshal(&metrics)
	if err != nil {
		return fmt.Errorf("error encode metric: %w", err)
	}

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(data).
		SetContext(ctx).
		Post(agent.cfg.Addr + handler.PartURLUpdates)

	if err != nil {
		return fmt.Errorf("error create POST request to [%s]: %w", agent.cfg.Addr+handler.PartURLUpdates, err)
	}

	if resp.StatusCode() != http.StatusOK {
		respBody := resp.Body()
		return fmt.Errorf("error report Batch JSON metrics: %s. Status: %d", string(respBody), resp.StatusCode())
	}

	return nil
}
