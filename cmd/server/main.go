package main

import (
	"net/http"
	"strconv"
	"strings"
)

const (
	idxMetricName  = 0
	idxMetricValue = 1
	sizeDataMetric = 2
)
const (
	gaugeUrlPart   = "/update/gauge/"
	counterUrlPart = "/update/counter/"
)

var metrics map[string]float64
var counter int64

func UpdateMetrics(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/plain")

	if r.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, "content-type is not supported", http.StatusUnsupportedMediaType)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method is not supported", http.StatusMethodNotAllowed)
	}

	// оставляем из url только <ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>
	// затем разбиваем на массив:
	// [0] - Название метрики
	// [1] - Значение метрики
	metric := strings.Split(strings.ReplaceAll(r.URL.String(), gaugeUrlPart, ""), "/")

	if len(metric) != sizeDataMetric {
		http.Error(w, "uncorrect request update metric", http.StatusBadRequest)
		return
	}

	if metrics == nil {
		metrics = make(map[string]float64)
	}

	metricValue, err := strconv.ParseFloat(metric[idxMetricValue], 64)
	if err != nil {
		http.Error(w, "uncorrect value type 'gauge' metric", http.StatusBadRequest)
		return
	}

	// обновляем значение метрики
	metrics[metric[idxMetricName]] = metricValue
	w.WriteHeader(http.StatusOK)
}

func UpdateCounter(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/plain")

	if r.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, "content-type is not supported", http.StatusUnsupportedMediaType)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method is not supported", http.StatusMethodNotAllowed)
	}

	// оставляем из url только <ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>
	// затем разбиваем на массив:
	// [0] - Название метрики
	// [1] - Значение метрики
	metric := strings.Split(strings.ReplaceAll(r.URL.String(), counterUrlPart, ""), "/")

	if len(metric) != sizeDataMetric {
		http.Error(w, "uncorrect request metric update", http.StatusBadRequest)
		return
	}

	if metrics == nil {
		metrics = make(map[string]float64)
	}

	metricValue, err := strconv.ParseInt(metric[idxMetricValue], 10, 64)
	if err != nil {
		http.Error(w, "uncorrect value type 'counter' metric", http.StatusBadRequest)
		return
	}

	counter += metricValue
	w.WriteHeader(http.StatusOK)
}

func main() {

	http.HandleFunc(gaugeUrlPart, UpdateMetrics)
	http.HandleFunc(counterUrlPart, UpdateCounter)
	http.ListenAndServe(":8080", nil)
}
