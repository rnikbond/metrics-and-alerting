package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"metrics-and-alerting/pkg/errs"
	metricPkg "metrics-and-alerting/pkg/metric"
)

func (h Handler) UpdateURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "text/plain")

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// оставляем из url только <ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>
		// затем разбиваем на массив:
		// [0] - Тип метрики
		// [1] - Название метрики
		// [2] - Значение метрики
		dataURL := strings.ReplaceAll(r.URL.String(), "/update/", "")
		partsURL := strings.Split(dataURL, "/")

		if len(partsURL) != partsUpdateURL {

			err := fmt.Errorf("invalid URL: %s", r.URL.String())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		metric, err := metricPkg.CreateMetric(
			partsURL[idxType],
			partsURL[idxName],
			metricPkg.WithValue(partsURL[idxValue]),
		)

		if err != nil {
			log.Printf("error create metric: %v\n", err)
			http.Error(w, err.Error(), errs.ErrorHTTP(err))
			return
		}

		if err := h.store.Upsert(metric); err != nil {
			log.Printf("error upsert metric: %v\n", err)
			http.Error(w, err.Error(), errs.ErrorHTTP(err))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (h Handler) UpdateJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set(ContentType, "text/plain")

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get(ContentType) != ApplicationJSON {
			w.WriteHeader(http.StatusUnsupportedMediaType)
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

		var metric metricPkg.Metric
		if err := json.Unmarshal(data, &metric); err != nil {
			log.Printf("error decode JSON body: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := h.store.Upsert(metric); err != nil {
			log.Printf("error update metric: %v\n", err)
			http.Error(w, err.Error(), errs.ErrorHTTP(err))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (h Handler) UpdateDataJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set(ContentType, "text/plain")

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get(ContentType) != ApplicationJSON {
			w.WriteHeader(http.StatusUnsupportedMediaType)
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

		var metrics []metricPkg.Metric
		if err := json.Unmarshal(data, &metrics); err != nil {
			log.Printf("error decode JSON body: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := h.store.UpsertSlice(metrics); err != nil {
			log.Printf("error update metric: %v\n", err)
			http.Error(w, err.Error(), errs.ErrorHTTP(err))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
