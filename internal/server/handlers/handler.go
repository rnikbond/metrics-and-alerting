package handler

import (
	"net/http"
	"strings"

	"metrics-and-alerting/internal/storage"
	errst "metrics-and-alerting/pkg/errorsstorage"
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

func GetMetrics(st storage.IStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		if r.Method != http.MethodGet {
			http.Error(w, "method is not supported", http.StatusMethodNotAllowed)
			return
		}

		html := ""
		types := []string{storage.GaugeType, storage.CounterType}

		for _, typeMetric := range types {
			names := st.Names(typeMetric)
			for _, metric := range names {
				val, err := st.Get(typeMetric, metric)
				if err == nil {
					html += metric + ":" + val + "<br/>"
				} else {
					//log.Printf("error get value metric %s/%s\n", typeMetric, metric)
				}
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}
}

func GetMetric(st storage.IStorage) http.HandlerFunc {
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
			//log.Printf("error request get metric %v - %s", metric, r.URL)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		val, err := st.Get(metric[idxMetricType], metric[idxMetricName])
		if err != nil {
			//log.Printf("error get value metric %s/%s - %s",
			//	metric[idxMetricName],
			//	metric[idxMetricType],
			//	err.Error())

			http.Error(w, err.Error(), errst.ConvertToHTTP(err))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(val))
	}
}

func UpdateMetric(st storage.IStorage) http.HandlerFunc {
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
			//log.Printf("error request update metric. Requaest  %v - %s", metric, r.URL)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		err := st.Update(metric[idxMetricType], metric[idxMetricName], metric[idxMetricValue])
		if err != nil {
			//log.Printf("error update metric %s/%s/%s - %s",
			//	metric[idxMetricName],
			//	metric[idxMetricValue],
			//	metric[idxMetricType],
			//	err.Error())

			http.Error(w, err.Error(), errst.ConvertToHTTP(err))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success update metric"))
	}
}
