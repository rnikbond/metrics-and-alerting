package handler

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	signKey = "KeySignMetric"
)

func randFloat64() *float64 {
	rand.Seed(time.Now().UnixNano())
	val := rand.Float64()
	return &val
}

func randInt64() *int64 {
	rand.Seed(time.Now().UnixNano())
	val := rand.Int63()
	return &val
}

func NewGaugeMetric() storage.Metric {
	return storage.Metric{
		ID:    "testGauge",
		MType: storage.GaugeType,
		Value: randFloat64(),
	}
}

func NewCounterMetric() storage.Metric {
	return storage.Metric{
		ID:    "testCounter",
		MType: storage.CounterType,
		Delta: randInt64(),
	}
}

// TestGetJSON - Тест на получение метрики в JSON виде
func TestGetJSON(t *testing.T) {

	cfg := config.Config{}
	cfg.SetDefault()
	cfg.VerifyOnUpdate = false
	cfg.SecretKey = signKey

	st := storage.InMemoryStorage{}
	errInit := st.Init(cfg)
	require.NoError(t, errInit)

	gaugeMetric := NewGaugeMetric()
	counterMetric := NewCounterMetric()

	signGauge, errSign := storage.Sign(gaugeMetric, []byte(signKey))
	require.NoError(t, errSign)
	gaugeMetric.Hash = signGauge

	signCounter, errSign := storage.Sign(counterMetric, []byte(signKey))
	require.NoError(t, errSign)
	counterMetric.Hash = signCounter

	require.NoError(t, st.Upset(gaugeMetric))
	require.NoError(t, st.Upset(counterMetric))

	tests := []struct {
		name            string
		acceptEncoding  string
		contentEncoding string
		method          string
		contentType     string
		wantStatus      int
		wantError       bool
		wantMetric      storage.Metric
		requestMetric   storage.Metric
	}{
		{ // Тело запроса отправляется в сжатом виде и ответ должен быть в сжатом виде
			name:            "Test get gauge -> OK",
			acceptEncoding:  GZip,
			contentEncoding: GZip,
			method:          http.MethodPost,
			contentType:     ApplicationJSON,
			wantStatus:      http.StatusOK,
			wantError:       false,
			wantMetric:      gaugeMetric,
			requestMetric: storage.Metric{
				ID:    gaugeMetric.ID,
				MType: storage.GaugeType,
			},
		},
		{ // Тело запроса отправляется без сжатия, а ответ должен быть в сжатом виде
			name:        "Test get gauge without GZIP -> OK",
			method:      http.MethodPost,
			contentType: ApplicationJSON,
			wantStatus:  http.StatusOK,
			wantError:   false,
			wantMetric:  gaugeMetric,
			requestMetric: storage.Metric{
				ID:    gaugeMetric.ID,
				MType: storage.GaugeType,
			},
		},
		{ // Запрос с некорректным HTTP методом
			name:        "Test get gauge by http.Get -> ERROR",
			method:      http.MethodGet,
			contentType: ApplicationJSON,
			wantStatus:  http.StatusMethodNotAllowed,
			wantError:   true,
			wantMetric:  gaugeMetric,
			requestMetric: storage.Metric{
				ID:    gaugeMetric.ID,
				MType: storage.GaugeType,
			},
		},
		{ // Запрос без указания заголовка Content-Type
			name:       "Test get gauge without content-type -> ERROR",
			method:     http.MethodPost,
			wantStatus: http.StatusUnsupportedMediaType,
			wantError:  true,
			wantMetric: gaugeMetric,
			requestMetric: storage.Metric{
				ID:    gaugeMetric.ID,
				MType: storage.GaugeType,
			},
		},
		{ // Тело запроса отправляется в сжатом виде и ответ должен быть в сжатом виде
			name:            "Test get counter -> OK",
			acceptEncoding:  GZip,
			contentEncoding: GZip,
			method:          http.MethodPost,
			contentType:     ApplicationJSON,
			wantStatus:      http.StatusOK,
			wantError:       false,
			wantMetric:      counterMetric,
			requestMetric: storage.Metric{
				ID:    counterMetric.ID,
				MType: storage.CounterType,
			},
		},
		{ // Тело запроса отправляется без сжатия, а ответ должен быть в сжатом виде
			name:        "Test get counter without GZIP -> OK",
			method:      http.MethodPost,
			contentType: ApplicationJSON,
			wantStatus:  http.StatusOK,
			wantError:   false,
			wantMetric:  counterMetric,
			requestMetric: storage.Metric{
				ID:    counterMetric.ID,
				MType: storage.CounterType,
			},
		},
		{ // Запрос с некорректным HTTP методом
			name:        "Test get counter by http.Get -> ERROR",
			method:      http.MethodGet,
			contentType: ApplicationJSON,
			wantStatus:  http.StatusMethodNotAllowed,
			wantError:   true,
			wantMetric:  counterMetric,
			requestMetric: storage.Metric{
				ID:    counterMetric.ID,
				MType: storage.CounterType,
			},
		},
		{ // Запрос без указания заголовка Content-Type
			name:       "Test get gauge without content-type -> ERROR",
			method:     http.MethodPost,
			wantStatus: http.StatusUnsupportedMediaType,
			wantError:  true,
			wantMetric: counterMetric,
			requestMetric: storage.Metric{
				ID:    counterMetric.ID,
				MType: storage.CounterType,
			},
		},
		{ // Запрос с неизвестным типом метрики
			name:        "Test get metric with unknown type -> ERROR",
			method:      http.MethodPost,
			contentType: ApplicationJSON,
			wantStatus:  http.StatusNotFound,
			wantError:   true,
			wantMetric:  counterMetric,
			requestMetric: storage.Metric{
				ID:    counterMetric.ID,
				MType: "ololo",
			},
		},
		{ // Запрос без указания типа метрики
			name:        "Test get metric with invalid type -> ERROR",
			method:      http.MethodPost,
			contentType: ApplicationJSON,
			wantStatus:  http.StatusNotFound,
			wantError:   true,
			wantMetric:  counterMetric,
			requestMetric: storage.Metric{
				ID: counterMetric.ID,
			},
		},
		{ // Запрос без указания названия метрики
			name:        "Test get metric with invalid id -> ERROR",
			method:      http.MethodPost,
			contentType: ApplicationJSON,
			wantStatus:  http.StatusNotFound,
			wantError:   true,
			wantMetric:  counterMetric,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encode, _ := json.Marshal(tt.requestMetric)

			nextHandler := GetJSON(&st)
			middleware := GZipHandle(nextHandler)

			var request *http.Request
			switch tt.contentEncoding {

			case GZip:
				var compress bytes.Buffer
				gzipWriter := gzip.NewWriter(&compress)
				_, errGZip := gzipWriter.Write(encode)
				require.NoError(t, errGZip)

				errGZip = gzipWriter.Close()
				require.NoError(t, errGZip)

				request = httptest.NewRequest(tt.method, PartURLValue, &compress)
				request.Header.Set(ContentEncoding, tt.contentEncoding)

			default:
				request = httptest.NewRequest(tt.method, PartURLValue, bytes.NewReader(encode))
			}

			request.Header.Set(ContentType, tt.contentType)
			request.Header.Set(AcceptEncoding, tt.acceptEncoding)

			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, request)

			response := w.Result()
			defer response.Body.Close()

			require.Equal(t, tt.wantStatus, response.StatusCode)

			if !tt.wantError {
				require.Equal(t, response.Header.Get(ContentType), tt.contentType)
				require.Equal(t, response.Header.Get(ContentEncoding), tt.acceptEncoding)

				var reader io.ReadCloser
				var errReader error

				switch tt.contentEncoding {
				case GZip:
					reader, errReader = gzip.NewReader(response.Body)
				default:
					reader = response.Body
				}

				require.NoError(t, errReader)
				defer reader.Close()

				data, errData := io.ReadAll(reader)
				require.NoError(t, errData)

				var metric storage.Metric
				errDecode := json.Unmarshal(data, &metric)
				require.NoError(t, errDecode)

				assert.Equal(t, metric, tt.wantMetric)
			}
		})
	}
}

func TestUpdateJSON(t *testing.T) {

	cfg := config.Config{}
	cfg.SetDefault()
	cfg.VerifyOnUpdate = false
	cfg.SecretKey = signKey

	st := storage.InMemoryStorage{}
	errInit := st.Init(cfg)
	require.NoError(t, errInit)

	gaugeMetric := NewGaugeMetric()
	counterMetric := NewCounterMetric()

	signGauge, errSign := storage.Sign(gaugeMetric, []byte(signKey))
	require.NoError(t, errSign)
	gaugeMetric.Hash = signGauge

	signCounter, errSign := storage.Sign(counterMetric, []byte(signKey))
	require.NoError(t, errSign)
	counterMetric.Hash = signCounter

	tests := []struct {
		name            string
		contentEncoding string
		method          string
		contentType     string
		wantStatus      int
		wantError       bool
		requestMetric   storage.Metric
	}{
		{ // Запрос без указания заголовка Content-Type
			name:            "Test update without content-type -> ERROR",
			contentEncoding: GZip,
			method:          http.MethodPost,
			wantStatus:      http.StatusUnsupportedMediaType,
			wantError:       true,
			requestMetric: storage.Metric{
				ID:    "testGauge",
				MType: storage.GaugeType,
				Value: gaugeMetric.Value,
			},
		},
		{ // Запрос с некорректным HTTP методом
			name:            "Test update with http.Get -> ERROR",
			contentEncoding: GZip,
			contentType:     ApplicationJSON,
			method:          http.MethodGet,
			wantStatus:      http.StatusMethodNotAllowed,
			wantError:       true,
			requestMetric: storage.Metric{
				ID:    "testGauge",
				MType: storage.GaugeType,
				Value: gaugeMetric.Value,
				Hash:  gaugeMetric.Hash,
			},
		},
		{ // Корректный запрос обновления gauge метрики
			name:            "Test update gauge -> OK",
			contentEncoding: GZip,
			method:          http.MethodPost,
			contentType:     ApplicationJSON,
			wantStatus:      http.StatusOK,
			wantError:       false,
			requestMetric: storage.Metric{
				ID:    "testGauge",
				MType: storage.GaugeType,
				Value: gaugeMetric.Value,
				Hash:  gaugeMetric.Hash,
			},
		},
		{ // Запрос без указания значения метрики
			name:            "Test update gauge without value -> ERROR",
			contentEncoding: GZip,
			method:          http.MethodPost,
			contentType:     ApplicationJSON,
			wantStatus:      http.StatusBadRequest,
			wantError:       true,
			requestMetric: storage.Metric{
				ID:    "testGauge",
				MType: storage.GaugeType,
				Hash:  gaugeMetric.Hash,
			},
		},
		{ // Запрос без указания подписи метрики
			name:            "Test update gauge without hash -> ERROR",
			contentEncoding: GZip,
			method:          http.MethodPost,
			contentType:     ApplicationJSON,
			wantStatus:      http.StatusBadRequest,
			wantError:       true,
			requestMetric: storage.Metric{
				ID:    "testGauge",
				MType: storage.GaugeType,
			},
		},
		{ // Запрос с некорректной подписью метрики
			name:            "Test update gauge with invalid hash -> ERROR",
			contentEncoding: GZip,
			method:          http.MethodPost,
			contentType:     ApplicationJSON,
			wantStatus:      http.StatusBadRequest,
			wantError:       true,
			requestMetric: storage.Metric{
				ID:    "testGauge",
				MType: storage.GaugeType,
				Hash:  "hash_ololo",
			},
		},
		{ // Запрос метрики без типа
			name:            "Test update gauge without type -> ERROR",
			contentEncoding: GZip,
			method:          http.MethodPost,
			contentType:     ApplicationJSON,
			wantStatus:      http.StatusNotImplemented,
			wantError:       true,
			requestMetric: storage.Metric{
				ID:    "testGauge",
				Value: gaugeMetric.Value,
				Hash:  gaugeMetric.Hash,
			},
		},
		{ // Запрос метрики без названия
			name:            "Test update gauge without id -> ERROR",
			contentEncoding: GZip,
			method:          http.MethodPost,
			contentType:     ApplicationJSON,
			wantStatus:      http.StatusBadRequest,
			wantError:       true,
			requestMetric: storage.Metric{
				MType: storage.GaugeType,
				Value: gaugeMetric.Value,
				Hash:  gaugeMetric.Hash,
			},
		},
		{ // Корректный запрос обновления counter метрики с сжатием тела запроса
			name:            "Test update counter -> OK",
			contentEncoding: GZip,
			method:          http.MethodPost,
			contentType:     ApplicationJSON,
			wantStatus:      http.StatusOK,
			wantError:       false,
			requestMetric: storage.Metric{
				ID:    "testCounter",
				MType: storage.CounterType,
				Delta: counterMetric.Delta,
				Hash:  counterMetric.Hash,
			},
		},
		{ // Корректный запрос обновления counter метрики, но без сжатия тела запроса
			name:        "Test update counter (unused gzip)-> OK",
			method:      http.MethodPost,
			contentType: ApplicationJSON,
			wantStatus:  http.StatusOK,
			wantError:   false,
			requestMetric: storage.Metric{
				ID:    "testCounter",
				MType: storage.CounterType,
				Delta: counterMetric.Delta,
				Hash:  counterMetric.Hash,
			},
		},
		{ // Запрос без указания подписи метрики
			name:            "Test update counter without hash -> ERROR",
			contentEncoding: GZip,
			method:          http.MethodPost,
			contentType:     ApplicationJSON,
			wantStatus:      http.StatusBadRequest,
			wantError:       true,
			requestMetric: storage.Metric{
				ID:    "testCounter",
				MType: storage.CounterType,
			},
		},
		{ // Запрос с некорректной подписью метрики
			name:            "Test update counter with invalid hash -> ERROR",
			contentEncoding: GZip,
			method:          http.MethodPost,
			contentType:     ApplicationJSON,
			wantStatus:      http.StatusBadRequest,
			wantError:       true,
			requestMetric: storage.Metric{
				ID:    "testCounter",
				MType: storage.CounterType,
				Hash:  "ololo_hash",
			},
		},
		{ // Запрос без указания значения метрики
			name:            "Test update counter without delta -> ERROR",
			contentEncoding: GZip,
			method:          http.MethodPost,
			contentType:     ApplicationJSON,
			wantStatus:      http.StatusBadRequest,
			wantError:       true,
			requestMetric: storage.Metric{
				ID:    "testCounter",
				MType: storage.CounterType,
				Hash:  counterMetric.Hash,
			},
		},
		{ // Запрос без указания типа метрики
			name:            "Test update counter without type -> ERROR",
			contentEncoding: GZip,
			method:          http.MethodPost,
			contentType:     ApplicationJSON,
			wantStatus:      http.StatusNotImplemented,
			wantError:       true,
			requestMetric: storage.Metric{
				ID:    "testCounter",
				Delta: counterMetric.Delta,
				Hash:  counterMetric.Hash,
			},
		},
		{ // Запрос без указания названия метрики
			name:            "Test update counter without id -> ERROR",
			contentEncoding: GZip,
			method:          http.MethodPost,
			contentType:     ApplicationJSON,
			wantStatus:      http.StatusBadRequest,
			wantError:       true,
			requestMetric: storage.Metric{
				MType: storage.CounterType,
				Delta: counterMetric.Delta,
				Hash:  counterMetric.Hash,
			},
		},
	}

	for _, tt := range tests {
		errReset := st.Reset()
		require.NoError(t, errReset)

		t.Run(tt.name, func(t *testing.T) {

			encode, errEncode := json.Marshal(tt.requestMetric)
			require.NoError(t, errEncode)

			var request *http.Request
			switch tt.contentEncoding {

			case GZip:
				var compress bytes.Buffer
				gzipWriter := gzip.NewWriter(&compress)
				_, errGZip := gzipWriter.Write(encode)
				require.NoError(t, errGZip)

				errGZip = gzipWriter.Close()
				require.NoError(t, errGZip)

				request = httptest.NewRequest(tt.method, PartURLUpdate, &compress)
				request.Header.Set(ContentEncoding, tt.contentEncoding)

			default:
				request = httptest.NewRequest(tt.method, PartURLUpdate, bytes.NewReader(encode))
			}

			request.Header.Set(ContentType, tt.contentType)

			nextHandler := UpdateJSON(&st)
			middleware := GZipHandle(nextHandler)

			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, request)

			response := w.Result()
			defer response.Body.Close()

			require.Equal(t, tt.wantStatus, response.StatusCode)

			if !tt.wantError {

				metricInStore := storage.Metric{
					ID:    tt.requestMetric.ID,
					MType: tt.requestMetric.MType,
				}

				var errStore error
				metricInStore, errStore = st.Get(metricInStore)
				require.NoError(t, errStore)

				switch metricInStore.MType {
				case storage.GaugeType:
					require.NotNil(t, metricInStore.Value)
					assert.Equal(t, *metricInStore.Value, *tt.requestMetric.Value)
				case storage.CounterType:
					require.NotNil(t, metricInStore.Delta)
					assert.Equal(t, *metricInStore.Delta, *tt.requestMetric.Delta)
				}
			}
		})
	}
}

/*
func TestUpdateMetricURL(t *testing.T) {

	memoryStorage := storage.InMemoryStorage{}

	type metricData struct {
		name       string
		value      string
		metricType string
	}

	tests := []struct {
		name string

		metricData metricData

		contentType string
		httpMethod  string
		wantCode    int
		wantError   bool
	}{
		{
			name: "TestUpdateMetric - Type {Counter} => [OK]",
			metricData: metricData{
				name:       "testCounter",
				value:      "100",
				metricType: storage.CounterType,
			},
			contentType: "text/plain",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusOK,
			wantError:   false,
		},
		{
			name: "TestUpdateMetric - Type {Counter}, Without {Name, Value} => [Error]",
			metricData: metricData{
				name:       "",
				value:      "",
				metricType: storage.CounterType,
			},
			contentType: "text/plain",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusNotFound,
			wantError:   true,
		},
		{
			name: "TestUpdateMetric - Type {Counter}, Without {Name, Value} => [Error]",
			metricData: metricData{
				name:       "testCounter",
				value:      "none",
				metricType: storage.CounterType,
			},
			contentType: "text/plain",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusBadRequest,
			wantError:   true,
		},
		{
			name: "TestUpdateMetric - Type {Counter}, Invalid {Value} => [Error]",
			metricData: metricData{
				name:       "testCounter",
				value:      "none",
				metricType: storage.CounterType,
			},
			contentType: "text/plain",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusBadRequest,
			wantError:   true,
		},
		{
			name: "TestUpdateMetric - Type {Gauge} => [OK]",
			metricData: metricData{
				name:       "testGauge",
				value:      "100",
				metricType: storage.GaugeType,
			},
			contentType: "text/plain",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusOK,
			wantError:   false,
		},
		{
			name: "TestUpdateMetric - Type {Gauge}, Without {Name, Value} => [Error]",
			metricData: metricData{
				name:       "",
				value:      "",
				metricType: storage.GaugeType,
			},
			contentType: "text/plain",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusNotFound,
			wantError:   true,
		},
		{
			name: "TestUpdateMetric - Type {Gauge}, Invalid {Value} => [Error]",
			metricData: metricData{
				name:       "testGauge",
				value:      "none",
				metricType: storage.GaugeType,
			},
			contentType: "text/plain",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusBadRequest,
			wantError:   true,
		},
	}
	for _, tt := range tests {

		memoryStorage.Reset()

		t.Run(tt.name, func(t *testing.T) {

			target := PartURLUpdate + "/"
			if len(tt.metricData.metricType) > 0 {
				target += tt.metricData.metricType
			}

			if len(tt.metricData.name) > 0 {
				if target[len(target)-1] != '/' {
					target += "/"
				}

				target += tt.metricData.name
			}

			if len(tt.metricData.value) > 0 {
				if target[len(target)-1] != '/' {
					target += "/"
				}

				target += tt.metricData.value
			}

			request := httptest.NewRequest(tt.httpMethod, target, nil)
			request.Header.Set("Content-Type", tt.contentType)

			w := httptest.NewRecorder()
			h := UpdateURL(&memoryStorage)
			h.ServeHTTP(w, request)

			response := w.Result()
			defer response.Body.Close()

			require.Equal(t, tt.wantCode, response.StatusCode)

			if !tt.wantError {
				require.Equal(t, tt.contentType, response.Header.Get("Content-Type"))

				metric, _ := storage.CreateMetric(tt.metricData.metricType, tt.metricData.name)
				_, err := memoryStorage.Get(metric)
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetMetric(t *testing.T) {

	gauge, _ := storage.CreateMetric(storage.GaugeType, "testGauge", 100.023)
	counter, _ := storage.CreateMetric(storage.CounterType, "testCounter", 100)

	st := storage.InMemoryStorage{}
	st.Update(gauge)
	st.Update(counter)

	type metricData struct {
		name       string
		metricType string
	}

	tests := []struct {
		name       string
		metricData metricData

		contentType string
		httpMethod  string
		wantCode    int
		wantValue   string
		wantError   bool
	}{
		{
			name: "TestGetMetric - Type {Gauge} => [OK]",
			metricData: metricData{
				name:       "testGauge",
				metricType: storage.GaugeType,
			},
			contentType: "text/plain",
			httpMethod:  http.MethodGet,
			wantCode:    http.StatusOK,
			wantValue:   "100.023",

			wantError: false,
		},
		{
			name: "TestGetMetric - Type {Gauge}, Without {Name} => [Error]",
			metricData: metricData{
				metricType: storage.GaugeType,
			},
			contentType: "text/plain",
			httpMethod:  http.MethodGet,
			wantCode:    http.StatusNotFound,

			wantError: true,
		},
		{
			name: "TestGetMetric - Without {Type} => [Error]",
			metricData: metricData{
				name: "testGauge",
			},
			contentType: "text/plain",
			httpMethod:  http.MethodGet,
			wantCode:    http.StatusNotFound,

			wantError: true,
		},
		{
			name:        "TestGetMetric - Without {Type, Name} => [Error]",
			metricData:  metricData{},
			contentType: "text/plain",
			httpMethod:  http.MethodGet,
			wantCode:    http.StatusNotFound,

			wantError: true,
		},
		{
			name: "TestGetMetric - Type {Counter} => [OK]",
			metricData: metricData{
				name:       "testCounter",
				metricType: storage.CounterType,
			},
			contentType: "text/plain",
			httpMethod:  http.MethodGet,
			wantCode:    http.StatusOK,
			wantValue:   "100",

			wantError: false,
		},
		{
			name: "TestGetMetric - Type {Counter}, Without {Name} => [Error]",
			metricData: metricData{
				metricType: storage.CounterType,
			},
			contentType: "text/plain",
			httpMethod:  http.MethodGet,
			wantCode:    http.StatusNotFound,

			wantError: true,
		},
		{
			name: "TestGetMetric - Without {Type} => [Error]",
			metricData: metricData{
				name: "testCounter",
			},
			contentType: "text/plain",
			httpMethod:  http.MethodGet,
			wantCode:    http.StatusNotFound,

			wantError: true,
		},
		{
			name:        "TestGetMetric - Without {Type, Name} => [Error]",
			metricData:  metricData{},
			contentType: "text/plain",
			httpMethod:  http.MethodGet,
			wantCode:    http.StatusNotFound,

			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := PartURLValue + "/"
			if len(tt.metricData.metricType) > 0 {
				target += tt.metricData.metricType
			}

			if len(tt.metricData.name) > 0 {
				if target[len(target)-1] != '/' {
					target += "/"
				}

				target += tt.metricData.name
			}

			request := httptest.NewRequest(tt.httpMethod, target, nil)
			request.Header.Set("Content-Type", tt.contentType)

			w := httptest.NewRecorder()
			h := Get(&st)
			h.ServeHTTP(w, request)

			response := w.Result()
			defer response.Body.Close()

			require.Equal(t, tt.wantCode, response.StatusCode)

			got, err := io.ReadAll(response.Body)

			if !tt.wantError {
				require.NoError(t, err)
				require.Equal(t, tt.contentType, response.Header.Get("Content-Type"))
				assert.Equal(t, string(got), tt.wantValue)
			}

		})
	}
}

func TestGetMetrics(t *testing.T) {

	gauge, _ := storage.CreateMetric(storage.GaugeType, "testGauge1", 100.023)

	st := storage.InMemoryStorage{}
	st.Update(gauge)

	//type metricData struct {
	//	name       string
	//	value      string
	//	metricType string
	//}

	f := 100.023

	tests := []struct {
		name   string
		metric storage.Metric

		contentType string
		httpMethod  string
		wantCode    int
		wantValue   string
		wantError   bool
	}{
		{
			name: "TestGetMetrics => [OK]",
			metric: storage.Metric{
				ID:    "testGauge1",
				MType: storage.GaugeType,
				Value: &f,
			},
			contentType: "text/html",
			httpMethod:  http.MethodGet,
			wantCode:    http.StatusOK,
			wantError:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.httpMethod, "/", nil)
			request.Header.Set("Content-Type", tt.contentType)

			w := httptest.NewRecorder()
			h := http.HandlerFunc(GetMetrics(&st))
			h.ServeHTTP(w, request)

			response := w.Result()
			defer response.Body.Close()

			require.Equal(t, tt.wantCode, response.StatusCode)

			got, err := io.ReadAll(response.Body)
			answer := strings.Replace(string(got), "<br/>", "", -1)

			if !tt.wantError {
				require.NoError(t, err)
				require.Equal(t, tt.contentType, response.Header.Get("Content-Type"))
				assert.Equal(t, answer, tt.metric.ShotString())
			}

		})
	}
}

func TestUpdateMetricJSON(t *testing.T) {

	st := storage.InMemoryStorage{}

	value := 123.123
	var delta int64 = 123

	tests := []struct {
		name        string
		httpMethod  string
		contentType string
		metric      storage.Metric
		wantStatus  int
	}{
		{
			name:        "Update gauge Post/JSON => [OK]",
			httpMethod:  http.MethodPost,
			contentType: "application/json",
			metric: storage.Metric{
				ID:    "testGauge",
				MType: storage.GaugeType,
				Value: &value,
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "Update gauge Get/JSON => [ERROR]",
			httpMethod:  http.MethodGet,
			contentType: "application/json",
			wantStatus:  http.StatusMethodNotAllowed,
		},
		{
			name:        "Update gauge Post/Text => [ERROR]",
			httpMethod:  http.MethodPost,
			contentType: "text/plain",
			wantStatus:  http.StatusUnsupportedMediaType,
		},
		{
			name:        "Update gauge Post/JSON Without{ID} => [ERROR]",
			httpMethod:  http.MethodPost,
			contentType: "application/json",
			metric: storage.Metric{
				MType: storage.GaugeType,
				Value: &value,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:        "Update gauge Post/JSON Without{Value} => [ERROR]",
			httpMethod:  http.MethodPost,
			contentType: "application/json",
			metric: storage.Metric{
				ID:    "testGauge",
				MType: storage.GaugeType,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:        "Update gauge Post/JSON Without{Type,Value} => [ERROR]",
			httpMethod:  http.MethodPost,
			contentType: "application/json",
			metric: storage.Metric{
				ID: "testGauge",
			},
			wantStatus: http.StatusNotImplemented,
		},

		{
			name:        "Update counter Post/JSON => [OK]",
			httpMethod:  http.MethodPost,
			contentType: "application/json",
			metric: storage.Metric{
				ID:    "testCounter",
				MType: storage.CounterType,
				Delta: &delta,
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "Update counter Get/JSON => [ERROR]",
			httpMethod:  http.MethodGet,
			contentType: "application/json",
			wantStatus:  http.StatusMethodNotAllowed,
		},
		{
			name:        "Update counter Post/Text => [ERROR]",
			httpMethod:  http.MethodPost,
			contentType: "text/plain",
			wantStatus:  http.StatusUnsupportedMediaType,
		},
		{
			name:        "Update counter Post/JSON Without{ID} => [ERROR]",
			httpMethod:  http.MethodPost,
			contentType: "application/json",
			metric: storage.Metric{
				MType: storage.CounterType,
				Delta: &delta,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:        "Update counter Post/JSON Without{Delta} => [ERROR]",
			httpMethod:  http.MethodPost,
			contentType: "application/json",
			metric: storage.Metric{
				ID:    "testCounter",
				MType: storage.CounterType,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:        "Update counter Post/JSON Without{Type,Delta} => [ERROR]",
			httpMethod:  http.MethodPost,
			contentType: "application/json",
			metric: storage.Metric{
				ID: "testCounter",
			},
			wantStatus: http.StatusNotImplemented,
		},
		{
			name:        "Update counter Post/JSON (YandexTest) => [OK]",
			httpMethod:  http.MethodPost,
			contentType: "application/json",
			metric: storage.Metric{
				ID:    "GetSet87",
				MType: storage.CounterType,
				Delta: &delta,
			},
			wantStatus: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			data, _ := json.Marshal(tt.metric)
			request := httptest.NewRequest(tt.httpMethod, PartURLUpdate, bytes.NewReader(data))
			request.Header.Set("Content-Type", tt.contentType)

			w := httptest.NewRecorder()
			h := UpdateJSON(&st)
			h.ServeHTTP(w, request)

			response := w.Result()
			defer response.Body.Close()

			assert.Equal(t, tt.wantStatus, response.StatusCode)
		})
	}
}

func TestGetMetricJSON(t *testing.T) {

	value := 123.123
	var delta int64 = 123

	gauge, _ := storage.CreateMetric(storage.GaugeType, "testGauge", value)
	counter, _ := storage.CreateMetric(storage.CounterType, "testCounter", delta)

	st := storage.InMemoryStorage{}
	st.Update(gauge)
	st.Update(counter)

	tests := []struct {
		name        string
		httpMethod  string
		contentType string
		reqMetric   storage.Metric
		wantMetric  storage.Metric
		wantStatus  int
		wantErr     bool
	}{
		{
			name:        "Get gauge metric => [OK]",
			httpMethod:  http.MethodPost,
			contentType: "application/json",
			reqMetric: storage.Metric{
				ID:    "testGauge",
				MType: storage.GaugeType,
			},
			wantMetric: storage.Metric{
				ID:    "testGauge",
				MType: storage.GaugeType,
				Value: &value,
			},
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:        "Get gauge metric Without{ID} => [ERROR]",
			httpMethod:  http.MethodPost,
			contentType: "application/json",
			reqMetric: storage.Metric{
				MType: storage.GaugeType,
			},
			wantStatus: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:        "Get gauge metric Without{Type} => [ERROR]",
			httpMethod:  http.MethodPost,
			contentType: "application/json",
			reqMetric: storage.Metric{
				ID: "testGauge",
			},
			wantStatus: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:        "Get gauge metric Without{ID,Type} => [ERROR]",
			httpMethod:  http.MethodPost,
			contentType: "application/json",
			wantStatus:  http.StatusNotFound,
			wantErr:     true,
		},
		{
			name:        "Get counter metric => [OK]",
			httpMethod:  http.MethodPost,
			contentType: "application/json",
			reqMetric: storage.Metric{
				ID:    "testCounter",
				MType: storage.CounterType,
			},
			wantMetric: storage.Metric{
				ID:    "testCounter",
				MType: storage.CounterType,
				Delta: &delta,
			},
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:        "Get counter metric Bad{Gauge} => [ERROR]",
			httpMethod:  http.MethodPost,
			contentType: "application/json",
			reqMetric: storage.Metric{
				ID:    "testCounter",
				MType: storage.GaugeType,
			},
			wantMetric: storage.Metric{
				ID:    "testCounter",
				MType: storage.CounterType,
				Delta: &delta,
			},
			wantStatus: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			data, _ := json.Marshal(tt.reqMetric)

			request := httptest.NewRequest(tt.httpMethod, PartURLValue, bytes.NewReader(data))
			request.Header.Set("Content-Type", tt.contentType)

			w := httptest.NewRecorder()
			h := GetJSON(&st)
			h.ServeHTTP(w, request)

			response := w.Result()
			defer response.Body.Close()

			require.Equal(t, tt.wantStatus, response.StatusCode)

			if !tt.wantErr {
				body, err := io.ReadAll(response.Body)
				require.NoError(t, err)

				var metric storage.Metric
				err = json.Unmarshal(body, &metric)
				require.NoError(t, err)

				assert.Equal(t, tt.wantMetric, metric)
			}
		})
	}
}
*/
