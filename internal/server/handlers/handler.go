package handler

import (
	"fmt"
	"net/http"
	"strings"

	"metrics-and-alerting/internal/storage"
)

const (
	idxMetricType  = 0
	idxMetricName  = 1
	idxMetricValue = 2
	sizeDataMetric = 3
)
const (
	PartURLUpdate = "/update/"
)

func UpdateMetric(metrics storage.Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "text/plain")

		if r.Method != http.MethodPost {
			http.Error(w, "method is not supported", http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Content-Type") != "text/plain" {
			fmt.Println(r.Header.Get("Content-Type"))
			http.Error(w, "content-type is not supported", http.StatusUnsupportedMediaType)
			return
		}

		// оставляем из url только <ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>
		// затем разбиваем на массив:
		// [0] - Тип метрики
		// [1] - Название метрики
		// [2] - Значение метрики
		metric := strings.Split(strings.ReplaceAll(r.URL.String(), PartURLUpdate, ""), "/")

		if len(metric) != sizeDataMetric {
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
