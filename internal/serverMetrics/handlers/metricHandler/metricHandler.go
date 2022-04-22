package metricHandler

import (
	"net/http"
	"strconv"
	"strings"

	storage "github.com/rnikbond/metrics-and-alerting/internal/storage"
)

const (
	idxMetricName  = 0
	idxMetricValue = 1
	sizeDataMetric = 2
)
const (
	GaugeUrlPart   = "/update/" + storage.GuageType + "/"
	CounterUrlPart = "/update/" + storage.CounterType + "/"
)

func UpdateMetricGauge(metrics storage.Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

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
		metric := strings.Split(strings.ReplaceAll(r.URL.String(), GaugeUrlPart, ""), "/")

		if len(metric) != sizeDataMetric {
			http.Error(w, "uncorrect request update metric", http.StatusBadRequest)
			return
		}

		metricValue, err := strconv.ParseFloat(metric[idxMetricValue], 64)
		if err != nil {
			http.Error(w, "uncorrect value type 'gauge' metric", http.StatusBadRequest)
			return
		}

		metrics.SetValueGaugeType(metric[idxMetricName], metricValue)
		w.WriteHeader(http.StatusOK)
	}
}

func UpdateMetricCounter(metrics storage.Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

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
		metric := strings.Split(strings.ReplaceAll(r.URL.String(), CounterUrlPart, ""), "/")

		if len(metric) != sizeDataMetric {
			http.Error(w, "uncorrect request metric update", http.StatusBadRequest)
			return
		}

		metricValue, err := strconv.ParseInt(metric[idxMetricValue], 10, 64)
		if err != nil {
			http.Error(w, "uncorrect value type 'counter' metric", http.StatusBadRequest)
			return
		}

		metrics.AddValueCounterType(metricValue)
		w.WriteHeader(http.StatusOK)
	}
}
