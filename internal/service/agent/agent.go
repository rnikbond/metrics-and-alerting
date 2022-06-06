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
	"time"

	handler "metrics-and-alerting/internal/server/handlers"
	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/config"

	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type Agent struct {
	Config  config.Config
	Storage *storage.InMemoryStorage
}

// Start Запуск агента для сбора и отправки метрик
func (agent *Agent) Start(ctx context.Context) {

	if agent.Storage == nil {
		panic("storage is not initialized")
	}

	if !strings.Contains(agent.Config.Addr, "http://") {
		agent.Config.Addr = "http://" + agent.Config.Addr
	}

	if err := agent.updateRuntime(); err != nil {
		log.Printf("error update runtime metrics on start agent: %v\n", err)
	}

	if err := agent.updateWorkload(); err != nil {
		log.Printf("error update workload metrics on start agent: %v\n", err)
	}

	// запуск горутины для обновления runtime метрик
	go agent.regularCollectRuntime(ctx)

	// запуск горутины для обновления mem/cpu метрик
	go agent.regularCollectWorkload(ctx)

	// запуск горутины для отправки метрик
	go agent.regularReport(ctx)

}

// regularReport Отправка метрик с частотой агента
func (agent *Agent) regularReport(ctx context.Context) {

	for {
		select {
		case <-time.After(agent.Config.ReportInterval):
			agent.reportAll(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// regularCollectRuntime Обновление runtime метрик
func (agent *Agent) regularCollectRuntime(ctx context.Context) {

	for {
		select {
		case <-time.After(agent.Config.PollInterval):
			if err := agent.updateRuntime(); err != nil {
				log.Printf("error regular update runtime metrics: %v\n", err)
			}

		case <-ctx.Done():
			return
		}
	}
}

// regularCollectWorkload Обновление метрик CPU и памяти
func (agent *Agent) regularCollectWorkload(ctx context.Context) {

	for {
		select {
		case <-time.After(agent.Config.PollInterval):
			if err := agent.updateWorkload(); err != nil {
				log.Printf("error regular update workload metrics: %v\n", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// Отправление всех метрик
func (agent *Agent) reportAll(ctx context.Context) {

	client := resty.New()

	metrics := agent.Storage.GetData()

	switch agent.Config.ReportType {
	case config.ReportBatchJSON:
		if err := agent.reportBatchJSON(ctx, client); err != nil {
			log.Println(err.Error())
		}

	case config.ReportJSON:
		for _, metric := range metrics {
			if err := agent.reportJSON(ctx, client, metric); err != nil {
				log.Println(err.Error())
			}
		}

	case config.ReportURL:
		for _, metric := range metrics {
			if err := agent.reportURL(ctx, client, metric); err != nil {
				log.Println(err.Error())
			}
		}
	}

	metric, _ := storage.CreateMetric(storage.CounterType, "PollCount", 0)
	if err := agent.Storage.Delete(metric); err != nil {
		log.Println(err.Error())
	}
}

// Обновление метрики
func (agent *Agent) reportURL(ctx context.Context, client *resty.Client, metric storage.Metric) error {

	data, err := metric.Map()
	if err != nil {
		return err
	}

	resp, err := client.R().
		SetPathParams(data).
		SetContext(ctx).
		Post(agent.Config.Addr + handler.PartURLUpdate + "/{type}/{name}/{value}")

	if err != nil {
		return fmt.Errorf("error create POST request to [%s]: %w",
			agent.Config.Addr+handler.PartURLUpdate+"/{type}/{name}/{value}",
			err)
	}

	if resp.StatusCode() != http.StatusOK {
		respBody := resp.Body()
		return fmt.Errorf("error report URL metric: %s. Status: %d", string(respBody), resp.StatusCode())
	}

	return nil
}

func (agent *Agent) reportJSON(ctx context.Context, client *resty.Client, metric storage.Metric) error {

	data, err := json.Marshal(&metric)
	if err != nil {
		return fmt.Errorf("error encode metric: %w", err)
	}

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(data).
		SetContext(ctx).
		Post(agent.Config.Addr + handler.PartURLUpdate)

	if err != nil {
		return fmt.Errorf("error create POST request to [%s]: %w", agent.Config.Addr+handler.PartURLUpdate, err)
	}

	if resp.StatusCode() != http.StatusOK {
		respBody := resp.Body()
		return fmt.Errorf("error report JSON metric: %s. Status: %d", string(respBody), resp.StatusCode())
	}

	return nil
}

func (agent *Agent) reportBatchJSON(ctx context.Context, client *resty.Client) error {

	metrics := agent.Storage.GetData()
	data, err := json.Marshal(&metrics)
	if err != nil {
		return fmt.Errorf("error encode metric: %w", err)
	}

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(data).
		SetContext(ctx).
		Post(agent.Config.Addr + handler.PartURLUpdates)

	if err != nil {
		return fmt.Errorf("error create POST request to [%s]: %w", agent.Config.Addr+handler.PartURLUpdates, err)
	}

	if resp.StatusCode() != http.StatusOK {
		respBody := resp.Body()
		return fmt.Errorf("error report Batch JSON metrics: %s. Status: %d", string(respBody), resp.StatusCode())
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

	for id, value := range gaugeMetrics {
		metric, err := storage.CreateMetric(storage.GaugeType, id, value)
		if err != nil {
			return err
		}

		if err := agent.Storage.Update(metric); err != nil {
			return err
		}
	}

	metric, err := storage.CreateMetric(storage.CounterType, "PollCount", 1)
	if err != nil {
		return err
	}

	if err := agent.Storage.Update(metric); err != nil {
		return err
	}

	return nil
}

// Обновление всех метрик
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

	for id, value := range gaugeMetrics {
		metric, err := storage.CreateMetric(storage.GaugeType, id, value)
		if err != nil {
			return err
		}

		if err := agent.Storage.Update(metric); err != nil {
			log.Printf("error update metric '%s': %v\n", metric.ShotString(), err)
		}
	}

	return nil
}
