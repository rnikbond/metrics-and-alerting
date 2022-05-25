package handler

import (
	"compress/gzip"
	"io"
	"net/http"
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
	PartURLUpdate = "/update"
	PartURLValue  = "/value"
	PartURLPing   = "/ping"
)

const (
	ContentType     = "Content-Type"
	ContentEncoding = "Content-Encoding"
	AcceptEncoding  = "Accept-Encoding"

	TextHTML        = "text/html"
	ApplicationJSON = "application/json"
	GZip            = "gzip"
)

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func GZipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !strings.Contains(r.Header.Get(AcceptEncoding), GZip) {
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		defer gz.Close()

		w.Header().Set(ContentEncoding, GZip)
		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

func GetMetrics(memStore *storage.MemoryStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(ContentType, TextHTML)

		if r.Method != http.MethodGet {
			http.Error(w, "method is not supported", http.StatusMethodNotAllowed)
			return
		}

		html := ""
		metrics := memStore.Data()

		for _, metric := range metrics {
			html += metric.ShotString() + "<br/>"
		}

		if r.Header.Get(AcceptEncoding) == GZip {
			if _, err := io.WriteString(w, html); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			w.Write([]byte(html))
		}
	}
}

func GetMetric(memStore *storage.MemoryStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set(ContentType, "text/plain")

		if r.Method != http.MethodGet {
			http.Error(w, "method is not supported", http.StatusMethodNotAllowed)
			return
		}

		// @TODO если проверять Content-Type - на github не проходят тесты :(
		// if r.Header.Get(ContentType) != "text/plain" {
		// 	fmt.Println(r.Header.Get(ContentType))
		// 	http.Error(w, "content-type is not supported", http.StatusUnsupportedMediaType)
		// 	return
		// }

		// оставляем из url только <ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>
		// затем разбиваем на массив:
		// [0] - Тип метрики
		// [1] - Название метрики
		partsURL := strings.Split(strings.ReplaceAll(r.URL.String(), PartURLValue+"/", ""), "/")

		if len(partsURL) != sizeDataGetMetric {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		metric, err := memStore.Get(partsURL[idxMetricType], partsURL[idxMetricName])
		if err != nil {
			http.Error(w, err.Error(), storage.ErrorHTTP(err))
			return
		}

		if r.Header.Get(AcceptEncoding) == GZip {
			if _, err := io.WriteString(w, metric.StringValue()); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			w.Write([]byte(metric.StringValue()))
		}
	}
}

func UpdateMetricURL(memStore *storage.MemoryStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "text/plain")

		if r.Method != http.MethodPost {
			http.Error(w, "method is not supported", http.StatusMethodNotAllowed)
			return
		}

		// @TODO если проверять Content-Type - на github не проходят тесты :(
		// if r.Header.Get(ContentType) != "text/plain" {
		// 	fmt.Println(r.Header.Get(ContentType))
		// 	http.Error(w, "content-type is not supported", http.StatusUnsupportedMediaType)
		// 	return
		// }

		// оставляем из url только <ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>
		// затем разбиваем на массив:
		// [0] - Тип метрики
		// [1] - Название метрики
		// [2] - Значение метрики
		partsURL := strings.Split(strings.ReplaceAll(r.URL.String(), PartURLUpdate+"/", ""), "/")

		if len(partsURL) != sizeDataUpdateMetric {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		metric := storage.NewMetric(partsURL[idxMetricType], partsURL[idxMetricName], partsURL[idxMetricValue])
		if err := memStore.Update(&metric); err != nil {
			http.Error(w, err.Error(), storage.ErrorHTTP(err))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func UpdateMetricJSON(memStore *storage.MemoryStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set(ContentType, "text/plain")

		if r.Method != http.MethodPost {
			http.Error(w, "method is not supported", http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get(ContentType) != ApplicationJSON {
			http.Error(w, "content-type is not supported", http.StatusUnsupportedMediaType)
			return
		}

		defer r.Body.Close()

		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "can not read request body", http.StatusBadRequest)
			return
		}

		metric, err := storage.FromJSON(data)
		if err != nil {
			http.Error(w, "JSON request: "+string(data)+"\n"+err.Error(), storage.ErrorHTTP(err))
			return
		}

		if err = memStore.Update(&metric); err != nil {
			http.Error(w, "JSON request: "+string(data)+"\n"+err.Error(), storage.ErrorHTTP(err))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func GetMetricJSON(memStore *storage.MemoryStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set(ContentType, ApplicationJSON)

		if r.Method != http.MethodPost {
			http.Error(w, "method is not supported", http.StatusMethodNotAllowed)
			return
		}

		defer r.Body.Close()

		data, errBody := io.ReadAll(r.Body)
		if errBody != nil {
			http.Error(w, "can not read request body", http.StatusBadRequest)
			return
		}

		metric, err := storage.FromJSON(data)
		if err != nil {
			http.Error(w, "JSON request: "+string(data)+"\n"+err.Error(), storage.ErrorHTTP(err))
			return
		}

		metric, err = memStore.Get(metric.MType, metric.ID)
		if err != nil {
			http.Error(w, err.Error(), storage.ErrorHTTP(err))
			return
		}

		data, err = metric.ToJSON()
		if err != nil {
			http.Error(w, err.Error(), storage.ErrorHTTP(err))
			return
		}

		if r.Header.Get(AcceptEncoding) == GZip {
			if _, err := io.WriteString(w, string(data)); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			w.Write(data)
		}
	}
}

func CheckHealthStorage(memStore *storage.MemoryStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodGet {
			http.Error(w, "method is not supported", http.StatusMethodNotAllowed)
			return
		}

		if memStore.ExternalStorage().Ping() {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
