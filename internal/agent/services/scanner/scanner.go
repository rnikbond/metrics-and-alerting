package scanner

import (
	"fmt"
	"math/rand"
	"runtime"
	"time"

	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/metric"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type Scanner struct {
	storage storage.Repository
}

func NewScanner(storage storage.Repository) *Scanner {
	return &Scanner{
		storage: storage,
	}
}

func (scan *Scanner) Scan() error {

	// TODO :: use errGroup

	if err := scan.updateRuntime(); err != nil {
		return err
	}

	if err := scan.updateWorkload(); err != nil {
		return err
	}

	return nil
}

// updateRuntime Обновление runtime метрик
func (scan *Scanner) updateRuntime() error {

	// TODO :: set length slice
	metrics := make([]metric.Metric, 0)

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	generator := rand.New(rand.NewSource(time.Now().UnixNano()))

	RandomValue, _ := metric.CreateMetric(metric.GaugeType, "RandomValue", metric.WithValueFloat(generator.Float64()))
	Alloc, _ := metric.CreateMetric(metric.GaugeType, "Alloc", metric.WithValueInt(int64(ms.Alloc)))
	BuckHashSys, _ := metric.CreateMetric(metric.GaugeType, "BuckHashSys", metric.WithValueInt(int64(ms.BuckHashSys)))
	Frees, _ := metric.CreateMetric(metric.GaugeType, "Frees", metric.WithValueInt(int64(ms.Frees)))
	GCCPUFraction, _ := metric.CreateMetric(metric.GaugeType, "GCCPUFraction", metric.WithValueInt(int64(ms.GCCPUFraction)))
	GCSys, _ := metric.CreateMetric(metric.GaugeType, "GCSys", metric.WithValueInt(int64(ms.GCSys)))
	HeapAlloc, _ := metric.CreateMetric(metric.GaugeType, "HeapAlloc", metric.WithValueInt(int64(ms.HeapAlloc)))
	HeapIdle, _ := metric.CreateMetric(metric.GaugeType, "HeapIdle", metric.WithValueInt(int64(ms.HeapIdle)))
	HeapInuse, _ := metric.CreateMetric(metric.GaugeType, "HeapInuse", metric.WithValueInt(int64(ms.HeapInuse)))
	HeapObjects, _ := metric.CreateMetric(metric.GaugeType, "HeapObjects", metric.WithValueInt(int64(ms.HeapObjects)))
	HeapReleased, _ := metric.CreateMetric(metric.GaugeType, "HeapReleased", metric.WithValueInt(int64(ms.HeapReleased)))
	HeapSys, _ := metric.CreateMetric(metric.GaugeType, "HeapSys", metric.WithValueInt(int64(ms.HeapSys)))
	LastGC, _ := metric.CreateMetric(metric.GaugeType, "LastGC", metric.WithValueInt(int64(ms.LastGC)))
	Lookups, _ := metric.CreateMetric(metric.GaugeType, "Lookups", metric.WithValueInt(int64(ms.Lookups)))
	MCacheInuse, _ := metric.CreateMetric(metric.GaugeType, "MCacheInuse", metric.WithValueInt(int64(ms.MCacheInuse)))
	MCacheSys, _ := metric.CreateMetric(metric.GaugeType, "MCacheSys", metric.WithValueInt(int64(ms.MCacheSys)))
	MSpanInuse, _ := metric.CreateMetric(metric.GaugeType, "MSpanInuse", metric.WithValueInt(int64(ms.MSpanInuse)))
	MSpanSys, _ := metric.CreateMetric(metric.GaugeType, "MSpanSys", metric.WithValueInt(int64(ms.MSpanSys)))
	Mallocs, _ := metric.CreateMetric(metric.GaugeType, "Mallocs", metric.WithValueInt(int64(ms.Mallocs)))
	NextGC, _ := metric.CreateMetric(metric.GaugeType, "NextGC", metric.WithValueInt(int64(ms.NextGC)))
	NumForcedGC, _ := metric.CreateMetric(metric.GaugeType, "NumForcedGC", metric.WithValueInt(int64(ms.NumForcedGC)))
	NumGC, _ := metric.CreateMetric(metric.GaugeType, "NumGC", metric.WithValueInt(int64(ms.NumGC)))
	OtherSys, _ := metric.CreateMetric(metric.GaugeType, "OtherSys", metric.WithValueInt(int64(ms.OtherSys)))
	PauseTotalNs, _ := metric.CreateMetric(metric.GaugeType, "PauseTotalNs", metric.WithValueInt(int64(ms.PauseTotalNs)))
	StackInuse, _ := metric.CreateMetric(metric.GaugeType, "StackInuse", metric.WithValueInt(int64(ms.StackInuse)))
	StackSys, _ := metric.CreateMetric(metric.GaugeType, "StackSys", metric.WithValueInt(int64(ms.StackSys)))
	Sys, _ := metric.CreateMetric(metric.GaugeType, "Sys", metric.WithValueInt(int64(ms.Sys)))
	TotalAlloc, _ := metric.CreateMetric(metric.GaugeType, "TotalAlloc", metric.WithValueInt(int64(ms.TotalAlloc)))
	PollCount, _ := metric.CreateMetric(metric.CounterType, "PollCount", metric.WithValueInt(1))

	metrics = append(metrics, RandomValue)
	metrics = append(metrics, Alloc)
	metrics = append(metrics, BuckHashSys)
	metrics = append(metrics, Frees)
	metrics = append(metrics, GCCPUFraction)
	metrics = append(metrics, GCSys)
	metrics = append(metrics, HeapAlloc)
	metrics = append(metrics, HeapIdle)
	metrics = append(metrics, HeapInuse)
	metrics = append(metrics, HeapObjects)
	metrics = append(metrics, HeapReleased)
	metrics = append(metrics, HeapSys)
	metrics = append(metrics, LastGC)
	metrics = append(metrics, Lookups)
	metrics = append(metrics, MCacheInuse)
	metrics = append(metrics, MCacheSys)
	metrics = append(metrics, MSpanInuse)
	metrics = append(metrics, MSpanSys)
	metrics = append(metrics, Mallocs)
	metrics = append(metrics, NextGC)
	metrics = append(metrics, NumForcedGC)
	metrics = append(metrics, NumGC)
	metrics = append(metrics, OtherSys)
	metrics = append(metrics, PauseTotalNs)
	metrics = append(metrics, StackInuse)
	metrics = append(metrics, StackSys)
	metrics = append(metrics, Sys)
	metrics = append(metrics, TotalAlloc)
	metrics = append(metrics, PollCount)

	return scan.storage.UpsertSlice(metrics)
}

// updateRuntime Обновление метрик загрузки памяти и ядер процессора
func (scan *Scanner) updateWorkload() error {

	vm, errVM := mem.VirtualMemory()
	if errVM != nil {
		return errVM
	}

	// TODO :: set length slice: 2 + cpuN
	metrics := make([]metric.Metric, 0)

	TotalMemory, _ := metric.CreateMetric(metric.GaugeType, "TotalMemory", metric.WithValueInt(int64(vm.Total)))
	FreeMemory, _ := metric.CreateMetric(metric.GaugeType, "FreeMemory", metric.WithValueInt(int64(vm.Free)))

	metrics = append(metrics, TotalMemory)
	metrics = append(metrics, FreeMemory)

	percentage, _ := cpu.Percent(0, true)
	for cpuID, cpuUtilization := range percentage {

		name := "CPUutilization" + fmt.Sprint(cpuID+1)
		cpuN, _ := metric.CreateMetric(metric.GaugeType, name, metric.WithValueFloat(cpuUtilization))
		metrics = append(metrics, cpuN)
	}

	return scan.storage.UpsertSlice(metrics)
}
