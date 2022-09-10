package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"metrics-and-alerting/pkg/errs"
	metricPkg "metrics-and-alerting/pkg/metric"
)

func (h Handler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		switch r.Method {
		case http.MethodGet:
			h.GetAsText()(w, r)

		case http.MethodPost:
			h.GetAsJSON()(w, r)

		default:
			h.logger.Err.Printf("request endpoint %s with unsupported HTTP method %s\n", r.URL, r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}
}

func (h Handler) GetAsText() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set(ContentType, TextPlain)

		// оставляем из url только <ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>
		// затем разбиваем на массив:
		// [0] - Тип метрики
		// [1] - Название метрики
		dataURL := strings.ReplaceAll(r.URL.String(), "/value/", "")
		partsURL := strings.Split(dataURL, "/")

		if len(partsURL) != partsGetURL {

			h.logger.Err.Printf("request endpoint %s with invalid URL\n", r.URL.String())
			w.WriteHeader(http.StatusNotFound)
			return
		}

		metric, err := metricPkg.CreateMetric(partsURL[idxType], partsURL[idxName])
		if err != nil {
			h.logger.Err.Printf("could not create metric: %v\n", err)
			http.Error(w, err.Error(), errs.ErrorHTTP(err))
			return
		}

		metric, err = h.store.Get(metric)
		if err != nil {
			h.logger.Err.Printf("error read metric from storage: %v\n", err)
			http.Error(w, err.Error(), errs.ErrorHTTP(err))
			return
		}

		h.CompressResponse(w, r, metric.StringValue())
	}
}

func (h Handler) GetAsJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Header.Get(ContentType) != ApplicationJSON {
			h.logger.Err.Printf("request with unsupported Content-Type: %s\n", r.Header.Get(ContentType))
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		defer func() {
			if err := r.Body.Close(); err != nil {
				h.logger.Err.Printf("error close body: %м\n", err)
			}
		}()

		w.Header().Set(ContentType, ApplicationJSON)

		reader, errReader := BodyReader(r)
		if errReader != nil {
			h.logger.Err.Printf("error get body reader: %v\n", errReader)
			http.Error(w, errReader.Error(), http.StatusBadRequest)
			return
		}
		defer func() {
			if err := reader.Close(); err != nil {
				h.logger.Err.Printf("error close reader: %v\n", err)
			}
		}()

		data, errBody := io.ReadAll(reader)
		if errBody != nil {
			h.logger.Err.Printf("error read body: %v\n", errBody)
			http.Error(w, errBody.Error(), http.StatusBadRequest)
			return
		}

		var metric metricPkg.Metric
		if err := json.Unmarshal(data, &metric); err != nil {
			h.logger.Err.Printf("error decode body to JSON: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		metric, errStorage := h.store.Get(metric)
		if errStorage != nil {
			h.logger.Err.Printf("could not get metric from storage: %v\n", errStorage)
			http.Error(w, errStorage.Error(), errs.ErrorHTTP(errStorage))
			return
		}

		encode, errEncode := json.Marshal(&metric)
		if errEncode != nil {
			h.logger.Err.Printf("error encode metric to JSON: %v\n", errStorage)
			http.Error(w, errEncode.Error(), http.StatusInternalServerError)
			return
		}

		h.CompressResponse(w, r, string(encode))
	}
}

func (h Handler) GetMetrics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set(ContentType, TextHTML)

		metrics, err := h.store.GetSlice()
		if err != nil {
			h.logger.Err.Printf("could not get all metrics from storage: %v\n", err)
			http.Error(w, err.Error(), errs.ErrorHTTP(err))
			return
		}

		html := ""
		for _, metric := range metrics {
			html += metric.ShotString() + "<br/>"
		}

		h.CompressResponse(w, r, html)
	}
}
