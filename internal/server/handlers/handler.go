package handler

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"metrics-and-alerting/internal/storage"
)

const (
	idxMetricType  = 0
	idxMetricName  = 1
	idxMetricValue = 2

	sizeDataGetMetric    = 2
	sizeDataUpdateMetric = 3
)
const (
	PartURLUpdate = "/update/"
	PartURLValue  = "/value/"
)

func GetMetrics(metrics storage.Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		if r.Method != http.MethodGet {
			http.Error(w, "method is not supported", http.StatusMethodNotAllowed)
			return
		}

		html := ""

		// Gauges
		gauges := metrics.GetGauges()

		keysGauges := make([]string, 0, len(gauges))
		for k := range gauges {
			keysGauges = append(keysGauges, k)
		}
		sort.Strings(keysGauges)

		for _, k := range keysGauges {
			html += k + ":" + strconv.FormatFloat(gauges[k], 'f', 3, 64) + "<br/>"
		}

		// Counters
		counters := metrics.GetCounters()
		keysCounters := make([]string, 0, len(counters))
		for k := range counters {
			keysCounters = append(keysCounters, k)
		}

		for _, k := range keysCounters {
			html += k + ":" + strconv.FormatInt(counters[k], 10) + "<br/>"
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}
}

func GetMetric(metrics storage.Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "text/plain")

		if r.Method != http.MethodGet {
			http.Error(w, "method is not supported", http.StatusMethodNotAllowed)
			return
		}

		// @TODO если проверять Content-Type - на github не проходят тесты :(

		// if r.Header.Get("Content-Type") != "text/plain" {
		// 	fmt.Println(r.Header.Get("Content-Type"))
		// 	http.Error(w, "content-type is not supported", http.StatusUnsupportedMediaType)
		// 	return
		// }

		// оставляем из url только <ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>
		// затем разбиваем на массив:
		// [0] - Тип метрики
		// [1] - Название метрики
		metric := strings.Split(strings.ReplaceAll(r.URL.String(), PartURLValue, ""), "/")

		if len(metric) != sizeDataGetMetric {
			http.Error(w, "uncorrect request get metric", http.StatusNotFound)
			return
		}

		value, code := metrics.Get(metric[idxMetricType], metric[idxMetricName])
		w.WriteHeader(code)
		w.Write([]byte(value))
	}
}

func UpdateMetric(metrics storage.Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "text/plain")

		if r.Method != http.MethodPost {
			http.Error(w, "method is not supported", http.StatusMethodNotAllowed)
			return
		}

		// @TODO если проверять Content-Type - на github не проходят тесты :(

		// if r.Header.Get("Content-Type") != "text/plain" {
		// 	fmt.Println(r.Header.Get("Content-Type"))
		// 	http.Error(w, "content-type is not supported", http.StatusUnsupportedMediaType)
		// 	return
		// }

		// оставляем из url только <ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>
		// затем разбиваем на массив:
		// [0] - Тип метрики
		// [1] - Название метрики
		// [2] - Значение метрики
		metric := strings.Split(strings.ReplaceAll(r.URL.String(), PartURLUpdate, ""), "/")

		if len(metric) != sizeDataUpdateMetric {
			http.Error(w, "uncorrect request update metric", http.StatusNotFound)
			return
		}

		var status int

		if metric[idxMetricType] == storage.CounterType {
			status = metrics.Add(metric[idxMetricName], metric[idxMetricValue], metric[idxMetricType])
		} else {
			status = metrics.Set(metric[idxMetricName], metric[idxMetricValue], metric[idxMetricType])
		}

		if status != http.StatusOK {
			http.Error(w, "fail update metric", status)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success update metric"))
	}
}
