package handler

import (
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

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		if r.Method != http.MethodPost {
			http.Error(w, "method is not supported", http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Content-Type") != "text/plain; charset=utf-8" {
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

		// проверка наличия названия метрики в запросе
		if len(metric[idxMetricName]) < 1 {
			http.Error(w, "uncorrect request update metric without name", http.StatusNotFound)
			return
		}

		err := metrics.Update(metric[idxMetricName], metric[idxMetricValue], metric[idxMetricType])
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
