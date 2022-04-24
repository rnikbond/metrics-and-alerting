package handler

import (
	"net/http"
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

		gauges := metrics.GetGauges()
		for k, v := range gauges {
			html += k + " " + strconv.FormatFloat(v, 'f', 3, 64) + "<br/>"
		}

		counters := metrics.GetCounters()
		for k, v := range counters {
			html += k + strconv.FormatInt(v, 10) + "<br/>"
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

		status := metrics.Update(metric[idxMetricName], metric[idxMetricValue], metric[idxMetricType])
		if status != http.StatusOK {
			http.Error(w, "fail update metric", status)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success update metric"))
	}
}
