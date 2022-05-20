package agent

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	handler "metrics-and-alerting/internal/server/handlers"
	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/config"
	errst "metrics-and-alerting/pkg/errorsstorage"

	"github.com/go-resty/resty/v2"
)

type Service interface {
	Start(ctx context.Context, wg *sync.WaitGroup)

	regularReport(ctx context.Context, wg *sync.WaitGroup)
	regularUpdate(ctx context.Context, wg *sync.WaitGroup)

	reportAll(ctx context.Context) error
	reportURL(ctx context.Context, nameMetric, valueMetric, typeMetric string) error
	reportJSON(ctx context.Context, nameMetric, metric *storage.Metrics) error

	updateAll()
}

type Agent struct {
	Config  *config.Config
	Storage storage.IStorage
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

// Start Запуск агента для сбора и отправки метрик
func (agent *Agent) Start(ctx context.Context) {

	if agent.Config == nil {
		panic(errors.New("not configured"))
	}

	if !strings.Contains(agent.Config.Addr, "http://") {
		agent.Config.Addr = "http://" + agent.Config.Addr
	}

	agent.Config.StoreFile = ""
	agent.Storage.SetExternalStorage(agent.Config)

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
	agent.updateAll()

	for {
		select {
		case <-time.After(agent.Config.PollInterval):
			agent.updateAll()
		case <-ctx.Done():
			return
		}
	}
}

// Отправление всех метрик
func (agent *Agent) reportAll(ctx context.Context) {

	agent.Storage.Lock()
	defer agent.Storage.Unlock()

	client := resty.New()
	types := []string{storage.GaugeType, storage.CounterType}

	for _, typeMetric := range types {
		names := agent.Storage.Names(typeMetric)

		for _, name := range names {

			metric, err := agent.Storage.Metric(typeMetric, name)
			if err != nil {
				log.Println(err)
				continue
			}

			if err = agent.reportJSON(ctx, client, &metric); err != nil {
				log.Println(err.Error())
			}

			//if err = agent.reportURL(ctx, client, &metric); err != nil {
			//	log.Println(err.Error())
			//}
		}
	}

	if err := agent.Storage.Set(storage.CounterType, "PollCount", 0); err != nil {
		log.Println(err.Error())
	}
}

// Обновление метрики
func (agent *Agent) reportURL(ctx context.Context, client *resty.Client, metric *storage.Metrics) error {

	if len(metric.MType) < 1 || len(metric.ID) < 1 {
		return errors.New("invalid metric params")
	}

	data := make(map[string]string)
	data["type"] = metric.MType
	data["name"] = metric.ID

	switch metric.MType {
	case storage.GaugeType:
		if metric.Value == nil {
			return errst.ErrorInvalidValue
		}

		data["value"] = strconv.FormatFloat(*metric.Value, 'f', -1, 64)

	case storage.CounterType:

		if metric.Delta == nil {
			return errst.ErrorInvalidValue
		}

		data["value"] = strconv.FormatInt(*metric.Delta, 10)

	default:
		return errst.ErrorInvalidType
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

	data, err := json.Marshal(metric)
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

// Обновление всех метрик
func (agent *Agent) updateAll() {

	agent.Storage.Lock()
	defer agent.Storage.Unlock()

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
		if err := agent.Storage.Set(storage.GaugeType, id, value); err != nil {
			log.Printf("error set value metric %s/%s/%v: %s\n", storage.GaugeType, id, value, err.Error())
		}
	}

	if err := agent.Storage.Add(storage.CounterType, "PollCount", 1); err != nil {
		log.Printf("error set value metric %s/%s/%v: %s\n", storage.GaugeType, "PollCount", 1, err.Error())
	}
}
