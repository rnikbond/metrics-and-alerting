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
	"sync"
	"time"

	"metrics-and-alerting/internal/storage"

	"github.com/go-resty/resty/v2"
)

type Service interface {
	Start(ctx context.Context, wg *sync.WaitGroup)

	regularReport(ctx context.Context, wg *sync.WaitGroup)
	regularUpdate(ctx context.Context, wg *sync.WaitGroup)

	reportAll(ctx context.Context) error
	reportURL(ctx context.Context, nameMetric, valueMetric, typeMetric string) error
	reportJSON(ctx context.Context, nameMetric, valueMetric, typeMetric string) error

	updateAll()
}

type Agent struct {
	ServerURL      string
	PollInterval   time.Duration
	ReportInterval time.Duration
	Storage        storage.IStorage
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
	// запуск горутины для обновления метрик
	go agent.regularUpdate(ctx)

	// запуск горутины для отправки метрик
	go agent.regularReport(ctx)

}

// Отправка метрик с частотой агента
func (agent *Agent) regularReport(ctx context.Context) {

	for {
		select {
		case <-time.After(agent.ReportInterval * time.Second):
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
		case <-time.After(agent.PollInterval * time.Second):
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
			value, err := agent.Storage.Get(typeMetric, name)
			if err != nil {
				log.Printf("Agent.reportAll() - error report metric %s/%s - %s",
					typeMetric, name, err.Error())
				continue
			}

			if err = agent.reportJSON(ctx, client, typeMetric, name, value); err != nil {
				log.Println(err.Error())
			}

			//if err = agent.reportURL(ctx, client, typeMetric, name, value); err != nil {
			//	log.Println(err.Error())
			//}
		}
	}

	if err := agent.Storage.Set(storage.CounterType, "PollCount", 0); err != nil {
		log.Println(err.Error())
	}
}

// Обновление метрики
func (agent *Agent) reportURL(ctx context.Context, client *resty.Client, typeMetric, nameMetric, valueMetric string) error {

	resp, err := client.R().SetPathParams(map[string]string{
		"type":  typeMetric,
		"name":  nameMetric,
		"value": valueMetric,
	}).SetContext(ctx).Post(agent.ServerURL + "/{type}/{name}/{value}")

	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		respBody := resp.Body()
		return errors.New("failed update metric: " + resp.Status() + ". " + string(respBody))
	}

	return nil
}

func (agent *Agent) reportJSON(ctx context.Context, client *resty.Client, typeMetric, nameMetric, valueMetric string) error {

	var metric storage.Metrics

	switch typeMetric {
	case storage.GaugeType:
		val, err := strconv.ParseFloat(valueMetric, 64)
		if err != nil {
			return err
		}

		metric = storage.Metrics{
			ID:    nameMetric,
			MType: typeMetric,
			Value: &val,
		}
	case storage.CounterType:
		val, err := strconv.ParseInt(valueMetric, 10, 64)
		if err != nil {
			return err
		}

		metric = storage.Metrics{
			ID:    nameMetric,
			MType: typeMetric,
			Delta: &val,
		}
	}

	data, err := json.Marshal(metric)
	if err != nil {
		return err
	}

	resp, err := client.R().
		SetBody(data).
		SetContext(ctx).
		Post(agent.ServerURL)

	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		respBody := resp.Body()
		return errors.New("failed update metric: " + resp.Status() + ". " + string(respBody))
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

	agent.Storage.Set(storage.GaugeType, "RandomValue", generator.Float64())
	agent.Storage.Set(storage.GaugeType, "Alloc", ms.Alloc)
	agent.Storage.Set(storage.GaugeType, "BuckHashSys", ms.BuckHashSys)
	agent.Storage.Set(storage.GaugeType, "Frees", ms.Frees)
	agent.Storage.Set(storage.GaugeType, "GCCPUFraction", ms.GCCPUFraction)
	agent.Storage.Set(storage.GaugeType, "GCSys", ms.GCSys)
	agent.Storage.Set(storage.GaugeType, "HeapAlloc", ms.HeapAlloc)
	agent.Storage.Set(storage.GaugeType, "HeapIdle", ms.HeapIdle)
	agent.Storage.Set(storage.GaugeType, "HeapInuse", ms.HeapInuse)
	agent.Storage.Set(storage.GaugeType, "HeapObjects", ms.HeapObjects)
	agent.Storage.Set(storage.GaugeType, "HeapReleased", ms.HeapReleased)
	agent.Storage.Set(storage.GaugeType, "HeapSys", ms.HeapSys)
	agent.Storage.Set(storage.GaugeType, "LastGC", ms.LastGC)
	agent.Storage.Set(storage.GaugeType, "Lookups", ms.Lookups)
	agent.Storage.Set(storage.GaugeType, "MCacheInuse", ms.MCacheInuse)
	agent.Storage.Set(storage.GaugeType, "MCacheSys", ms.MCacheSys)
	agent.Storage.Set(storage.GaugeType, "MSpanInuse", ms.MSpanInuse)
	agent.Storage.Set(storage.GaugeType, "MSpanSys", ms.MSpanSys)
	agent.Storage.Set(storage.GaugeType, "Mallocs", ms.Mallocs)
	agent.Storage.Set(storage.GaugeType, "NextGC", ms.NextGC)
	agent.Storage.Set(storage.GaugeType, "NumForcedGC", ms.NumForcedGC)
	agent.Storage.Set(storage.GaugeType, "NumGC", ms.NumGC)
	agent.Storage.Set(storage.GaugeType, "OtherSys", ms.OtherSys)
	agent.Storage.Set(storage.GaugeType, "PauseTotalNs", ms.PauseTotalNs)
	agent.Storage.Set(storage.GaugeType, "StackInuse", ms.StackInuse)
	agent.Storage.Set(storage.GaugeType, "StackSys", ms.StackSys)
	agent.Storage.Set(storage.GaugeType, "Sys", ms.Sys)
	agent.Storage.Set(storage.GaugeType, "TotalAlloc", ms.TotalAlloc)
	agent.Storage.Add(storage.CounterType, "PollCount", 1)

	//log.Println(agent.Storage)
}
