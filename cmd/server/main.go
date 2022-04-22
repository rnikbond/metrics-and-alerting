package main

import (
	"net/http"

	handler "github.com/rnikbond/metrics-and-alerting/internal/handlers/metricHandler"
	storage "github.com/rnikbond/metrics-and-alerting/internal/storage"
)

func main() {

	metrics := storage.MetricsData{}

	http.HandleFunc(handler.GaugeUrlPart, handler.UpdateMetricGauge(&metrics))
	http.HandleFunc(handler.CounterUrlPart, handler.UpdateMetricCounter(&metrics))
	http.ListenAndServe(":8080", nil)
}
