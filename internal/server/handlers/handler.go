package handler

import (
	"compress/gzip"
	"io"
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
	PartURLUpdate = "/update"
	PartURLValue  = "/value"
)

const (
	ContentType     = "Content-Type"
	ContentEncoding = "Content-Encoding"
	AcceptEncoding  = "Accept-Encoding"

	TextHtml        = "text/html"
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

func GetMetrics(st storage.IStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(ContentType, TextHtml)

		if r.Method != http.MethodGet {
			http.Error(w, "method is not supported", http.StatusMethodNotAllowed)
			return
		}

		html := ""
		types := []string{storage.GaugeType, storage.CounterType}

		st.Lock()
		defer st.Unlock()

		for _, typeMetric := range types {
			names := st.Names(typeMetric)
			for _, metric := range names {
				val, err := st.Get(typeMetric, metric)
				if err == nil {
					html += metric + ":" + val + "<br/>"
				}
			}
		}

		if r.Header.Get(AcceptEncoding) == GZip {
			if _, err := io.WriteString(w, html); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			w.Write([]byte(html))
		}

		//w.WriteHeader(http.StatusOK)
	}
}

func GetMetric(st storage.IStorage) http.HandlerFunc {
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
		metric := strings.Split(strings.ReplaceAll(r.URL.String(), PartURLValue+"/", ""), "/")

		if len(metric) != sizeDataGetMetric {
			//log.Printf("error request get metric %v - %s", metric, r.URL)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		st.Lock()
		defer st.Unlock()

		val, err := st.Get(metric[idxMetricType], metric[idxMetricName])
		if err != nil {
			http.Error(w, err.Error(), errst.ConvertToHTTP(err))
			return
		}

		if r.Header.Get(AcceptEncoding) == GZip {
			if _, err := io.WriteString(w, val); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			w.Write([]byte(val))
		}

		w.WriteHeader(http.StatusOK)
	}
}

func UpdateMetricURL(st storage.IStorage) http.HandlerFunc {
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
		metric := strings.Split(strings.ReplaceAll(r.URL.String(), PartURLUpdate+"/", ""), "/")

		if len(metric) != sizeDataUpdateMetric {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		st.Lock()
		defer st.Unlock()

		err := st.Update(metric[idxMetricType], metric[idxMetricName], metric[idxMetricValue])
		if err != nil {
			http.Error(w, err.Error(), errst.ConvertToHTTP(err))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func UpdateMetricJSON(st storage.IStorage) http.HandlerFunc {
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

		st.Lock()
		defer st.Unlock()
		if err = st.UpdateJSON(data); err != nil {
			http.Error(w, "JSON request: "+string(data)+"\n"+err.Error(), errst.ConvertToHTTP(err))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func GetMetricJSON(st storage.IStorage) http.HandlerFunc {
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

		metric, err := st.FillJSON(data)
		if err != nil {
			http.Error(w, err.Error(), errst.ConvertToHTTP(err))
			return
		}

		if r.Header.Get(AcceptEncoding) == GZip {
			if _, err := io.WriteString(w, string(metric)); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			w.Write(metric)
		}

		w.WriteHeader(http.StatusOK)
	}
}
