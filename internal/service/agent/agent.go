package agent

import (
	"context"
	"errors"
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
)

type Agent struct {
	Config  config.Config
	Storage *storage.MemoryStorage
}

// Start Запуск агента для сбора и отправки метрик
func (agent *Agent) Start(ctx context.Context) {

	if agent.Storage == nil {
		panic("storage is not initialized")
	}

	if !strings.Contains(agent.Config.Addr, "http://") {
		agent.Config.Addr = "http://" + agent.Config.Addr
	}

	// запуск горутины для обновления метрик
	go agent.regularUpdate(ctx)

	// запуск горутины для отправки метрик
	go agent.regularReport(ctx)

}

// Отправка метрик с частотой агента
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

// Обновление метрик с частотой агента
func (agent *Agent) regularUpdate(ctx context.Context) {
	agent.updateMetrics()

	for {
		select {
		case <-time.After(agent.Config.PollInterval):
			agent.updateMetrics()
		case <-ctx.Done():
			return
		}
	}
}

// Отправление всех метрик
func (agent *Agent) reportAll(ctx context.Context) {

	client := resty.New()

	//metrics := agent.Storage.Data()
	//for _, metric := range metrics {
	//
	//	if err := agent.reportJSON(ctx, client, &metric); err != nil {
	//		log.Println(err.Error())
	//	}
	//
	//	if err := agent.reportURL(ctx, client, &metric); err != nil {
	//		log.Println(err.Error())
	//	}
	//}

	if err := agent.reportBatchJSON(ctx, client); err != nil {
		log.Println(err.Error())
	}

	metric := storage.NewMetric(storage.CounterType, "PollCount", 0)

	if err := agent.Storage.Set(&metric); err != nil {
		log.Println(err.Error())
	}
}

// Обновление метрики
func (agent *Agent) reportURL(ctx context.Context, client *resty.Client, metric *storage.Metrics) error {

	data, err := metric.ToMap()
	if err != nil {
		return err
	}

	resp, err := client.R().
		SetPathParams(data).
		SetContext(ctx).
		Post(agent.Config.Addr + handler.PartURLUpdate + "/{type}/{name}/{value}")

	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		respBody := resp.Body()
		return errors.New("failed update metric: " + resp.Status() + ". " + string(respBody))
	}

	return nil
}

func (agent *Agent) reportJSON(ctx context.Context, client *resty.Client, metric *storage.Metrics) error {

	data, err := metric.ToJSON()
	if err != nil {
		return err
	}

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(data).
		SetContext(ctx).
		Post(agent.Config.Addr + handler.PartURLUpdate)

	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		respBody := resp.Body()
		return errors.New("\nJSON: " + string(data) + "\n" + metric.String() + ". " + string(respBody))
	}

	return nil
}

func (agent *Agent) reportBatchJSON(ctx context.Context, client *resty.Client) error {

	var jsonMetrics []string

	for _, metric := range agent.Storage.Data() {
		encodeData, err := metric.ToJSON()
		if err != nil {
			return err
		}

		jsonMetrics = append(jsonMetrics, string(encodeData))

	}

	jsonJoin := strings.Join(jsonMetrics, ";")

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody([]byte(jsonJoin)).
		SetContext(ctx).
		Post(agent.Config.Addr + handler.PartURLUpdates)

	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		respBody := resp.Body()
		return errors.New("\nJSON batch: " + jsonJoin + "\n" + string(respBody))
	}

	return nil
}

// Обновление всех метрик
func (agent *Agent) updateMetrics() {

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
		metric := storage.NewMetric(storage.GaugeType, id, value)
		if err := agent.Storage.Set(&metric); err != nil {
			log.Printf("error update metric '%s'. %s\n", metric.ShotString(), err.Error())
		}
	}

	metric := storage.NewMetric(storage.CounterType, "PollCount", 1)
	if err := agent.Storage.Add(&metric); err != nil {
		log.Printf("error update metric '%s'. %s\n", metric.ShotString(), err.Error())
	}
}
