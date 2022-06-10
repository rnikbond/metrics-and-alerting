package handler

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"metrics-and-alerting/internal/storage"
)

const (
	idxType  = 0
	idxName  = 1
	idxValue = 2

	partsGetURL    = 2
	partsUpdateURL = 3
)
const (
	PartURLUpdates = "/updates"
	PartURLUpdate  = "/update"
	PartURLValue   = "/value"
	PartURLPing    = "/ping"
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

func GetMetrics(store storage.Storager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(ContentType, TextHTML)

		if r.Method != http.MethodGet {
			err := fmt.Errorf("method '%s' is not supported", r.Method)
			log.Printf("error in request: %v\n", err)
			http.Error(w, err.Error(), http.StatusMethodNotAllowed)
			return
		}

		html := ""
		metrics := store.GetData()
		for _, metric := range metrics {
			html += metric.ShotString() + "<br/>"
		}

		if r.Header.Get(AcceptEncoding) == GZip {
			if _, err := io.WriteString(w, html); err != nil {
				log.Printf("error writing compressed data: %v\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			if _, err := w.Write([]byte(html)); err != nil {
				log.Printf("error write response: %v\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	}
}

func Get(store storage.Storager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set(ContentType, "text/plain")

		if r.Method != http.MethodGet {
			err := fmt.Errorf("method '%s' is not supported", r.Method)
			log.Printf("error in request: %v\n", err)
			http.Error(w, err.Error(), http.StatusMethodNotAllowed)
			return
		}

		// оставляем из url только <ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>
		// затем разбиваем на массив:
		// [0] - Тип метрики
		// [1] - Название метрики
		partsURL := strings.Split(strings.ReplaceAll(r.URL.String(), PartURLValue+"/", ""), "/")

		if len(partsURL) != partsGetURL {
			err := fmt.Errorf("invalid URL: %s", r.URL.String())
			log.Printf("error in request: %v\n", err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		metric, err := storage.CreateMetric(partsURL[idxType], partsURL[idxName])
		if err != nil {
			log.Printf("error create metric: %v\n", err)
			http.Error(w, err.Error(), storage.ErrorHTTP(err))
			return
		}

		metric, err = store.Get(metric)
		if err != nil {
			log.Printf("error get metric: %v\n", err)
			http.Error(w, err.Error(), storage.ErrorHTTP(err))
			return
		}

		if r.Header.Get(AcceptEncoding) == GZip {
			if _, err := io.WriteString(w, metric.StringValue()); err != nil {
				log.Printf("error write gzip response: %v\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			if _, err := w.Write([]byte(metric.StringValue())); err != nil {
				log.Printf("error write body response: %v\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	}
}

func GetJSON(store storage.Storager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			err := fmt.Errorf("method '%s' is not supported", r.Method)
			log.Printf("error in request: %v\n", err)
			http.Error(w, err.Error(), http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get(ContentType) != ApplicationJSON {
			err := fmt.Errorf("content-type '%s' is not supported", r.Header.Get(ContentType))
			log.Printf("error content-type in request: %v\n", err)
			http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
			return
		}

		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Printf("error close body in handler GetJSON: %v\n", err)
			}
		}()

		w.Header().Set(ContentType, ApplicationJSON)

		reader, errReader := BodyReader(r)
		if errReader != nil {
			log.Printf("error get body reader: %v\n", errReader)
			http.Error(w, errReader.Error(), http.StatusBadRequest)
			return
		}
		defer func() {
			if err := reader.Close(); err != nil {
				log.Printf("error close reader: %v\n", err)
			}
		}()

		data, errBody := io.ReadAll(reader)
		if errBody != nil {
			log.Printf("error read body: %v\n", errBody)
			http.Error(w, errBody.Error(), http.StatusBadRequest)
			return
		}

		var metric storage.Metric
		if err := json.Unmarshal(data, &metric); err != nil {
			log.Printf("error decode body to JSON: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		metric, errStorage := store.Get(metric)
		if errStorage != nil {
			log.Printf("error get metric from storage: %v\n", errStorage)
			http.Error(w, errStorage.Error(), storage.ErrorHTTP(errStorage))
			return
		}

		encode, errEncode := json.Marshal(&metric)
		if errEncode != nil {
			log.Printf("error encode metric to JSON: %v\n", errEncode)
			http.Error(w, errEncode.Error(), http.StatusInternalServerError)
			return
		}

		if _, err := w.Write(encode); err != nil {
			log.Printf("error write body response: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func UpdateURL(store storage.Storager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "text/plain")

		if r.Method != http.MethodPost {
			err := fmt.Errorf("method '%s' is not supported", r.Method)
			log.Printf("error in request: %v\n", err)
			http.Error(w, err.Error(), http.StatusMethodNotAllowed)
			return
		}

		// оставляем из url только <ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>
		// затем разбиваем на массив:
		// [0] - Тип метрики
		// [1] - Название метрики
		// [2] - Значение метрики
		partsURL := strings.Split(strings.ReplaceAll(r.URL.String(), PartURLUpdate+"/", ""), "/")
		if len(partsURL) != partsUpdateURL {
			err := fmt.Errorf("invalid URL: %s", r.URL.String())
			log.Printf("error in request: %v\n", err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		metric, err := storage.CreateMetric(partsURL[idxType], partsURL[idxName], partsURL[idxValue])
		if err != nil {
			log.Printf("error create metric: %v\n", err)
			http.Error(w, err.Error(), storage.ErrorHTTP(err))
			return
		}

		if err := store.Upset(metric); err != nil {
			log.Printf("error update metric: %v\n", err)
			http.Error(w, err.Error(), storage.ErrorHTTP(err))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func UpdateJSON(store storage.Storager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set(ContentType, "text/plain")

		if r.Method != http.MethodPost {
			err := fmt.Errorf("method '%s' is not supported", r.Method)
			log.Printf("error in request: %v\n", err)
			http.Error(w, err.Error(), http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get(ContentType) != ApplicationJSON {
			err := fmt.Errorf("content-type '%s' is not supported", r.Header.Get(ContentType))
			log.Printf("error content-type in request: %v\n", err)
			http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
			return
		}

		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Printf("error close body in handler UpdateJSON: %v\n", err)
			}
		}()

		reader, errReader := BodyReader(r)
		if errReader != nil {
			log.Printf("error get body reader: %v\n", errReader)
			http.Error(w, errReader.Error(), http.StatusBadRequest)
			return
		}
		defer func() {
			if err := reader.Close(); err != nil {
				log.Printf("error close reader: %v\n", err)
			}
		}()

		data, err := io.ReadAll(reader)
		if err != nil {
			log.Printf("error read body request: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var metric storage.Metric
		if err := json.Unmarshal(data, &metric); err != nil {
			log.Printf("error decode JSON body: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := store.Upset(metric); err != nil {
			log.Printf("error update metric: %v\n", err)
			http.Error(w, err.Error(), storage.ErrorHTTP(err))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func UpdateDataJSON(store storage.Storager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set(ContentType, "text/plain")

		if r.Method != http.MethodPost {
			err := fmt.Errorf("method '%s' is not supported", r.Method)
			log.Printf("error in request: %v\n", err)
			http.Error(w, err.Error(), http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get(ContentType) != ApplicationJSON {
			err := fmt.Errorf("content-type '%s' is not supported", r.Header.Get(ContentType))
			log.Printf("error content-type in request: %v\n", err)
			http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
			return
		}

		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Printf("error close body in handler UpdateDataJSON: %v\n", err)
			}
		}()

		data, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("error read body request: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var metrics []storage.Metric
		if err := json.Unmarshal(data, &metrics); err != nil {
			log.Printf("error decode JSON body: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := store.UpsetData(metrics); err != nil {
			log.Printf("error update metric: %v\n", err)
			http.Error(w, err.Error(), storage.ErrorHTTP(err))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func Ping(store storage.Storager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodGet {
			err := fmt.Errorf("method '%s' is not supported", r.Method)
			log.Printf("error in request: %v\n", err)
			http.Error(w, err.Error(), http.StatusMethodNotAllowed)
			return
		}

		if store.CheckHealth() {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func GZipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !strings.Contains(r.Header.Get(AcceptEncoding), GZip) {
			next.ServeHTTP(w, r)
			return
		}

		writer := gzip.NewWriter(w)
		defer func() {
			if err := writer.Close(); err != nil {
				log.Printf("error close GZIP writer: %v\n", err)
			}
		}()

		w.Header().Set(ContentEncoding, GZip)
		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: writer}, r)
	})
}

func BodyReader(r *http.Request) (io.ReadCloser, error) {

	switch r.Header.Get(ContentEncoding) {
	case GZip:
		reader, errReader := gzip.NewReader(r.Body)
		if errReader != nil {
			return nil, errReader
		}

		return reader, nil
	}

	return r.Body, nil
}
