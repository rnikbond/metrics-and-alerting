package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"metrics-and-alerting/internal/storage/memstore"
	"metrics-and-alerting/pkg/logpack"
	metricPkg "metrics-and-alerting/pkg/metric"

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

func NewGaugeMetric() metricPkg.Metric {
	return metricPkg.Metric{
		ID:    "testGauge",
		MType: metricPkg.GaugeType,
		Value: randFloat64(),
	}
}

func NewCounterMetric() metricPkg.Metric {
	return metricPkg.Metric{
		ID:    "testCounter",
		MType: metricPkg.CounterType,
		Delta: randInt64(),
	}
}

// TestTrustedIP Тест Middleware для проверки IP адреса клиента через список доверительных IP адресов
func TestTrustedIP(t *testing.T) {

	logger := logpack.NewLogger()

	tests := []struct {
		name       string
		handler    *Handler
		realIP     string
		wantStatus int
	}{
		{
			name:       "Success request: SERVER with trusted ips, CLIENT with X-Real-IP",
			handler:    New(memstore.New(), logger, WithTrustedSubnet("192.168.1.1, 192.168.1.2, 127.0.0.1")),
			realIP:     "192.168.1.1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "Error request: SERVER with trusted ips, CLIENT without X-Real-IP",
			handler:    New(memstore.New(), logger, WithTrustedSubnet("192.168.1.1, 192.168.1.2, 127.0.0.1")),
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "Error request: SERVER with trusted ips, CLIENT with another X-Real-IP",
			handler:    New(memstore.New(), logger, WithTrustedSubnet("192.168.1.1, 192.168.1.2, 127.0.0.1")),
			realIP:     "192.168.1.5",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "Success request: SERVER without trusted ips, CLIENT with X-Real-IP",
			handler:    New(memstore.New(), logger),
			realIP:     "192.168.1.5",
			wantStatus: http.StatusOK,
		},
		{
			name:       "Success request: SERVER without trusted ips, CLIENT without X-Real-IP",
			handler:    New(memstore.New(), logger),
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			metric := metricPkg.Metric{
				ID:    "counter",
				MType: metricPkg.CounterType,
				Delta: randInt64(),
			}

			errUpsert := tt.handler.store.Upsert(metric)
			require.NoError(t, errUpsert)

			nextHandler := tt.handler.GetAsText()
			middleware := tt.handler.Trust(nextHandler)

			URL := fmt.Sprintf("/value/%s/%s", metric.MType, metric.ID)
			request := httptest.NewRequest(http.MethodGet, URL, nil)
			request.Header.Set(ContentType, "text/plain")
			request.Header.Set(XRealIP, tt.realIP)

			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, request)

			response := w.Result()
			defer response.Body.Close()

			assert.Equal(t, tt.wantStatus, response.StatusCode)
		})
	}
}

// TestGetJSON - Тест на получение метрики в JSON виде
func TestGetJSON(t *testing.T) {

	logger := logpack.NewLogger()
	st := memstore.New()
	handlers := New(st, logger)

	gaugeMetric := NewGaugeMetric()
	counterMetric := NewCounterMetric()

	signGauge, errSign := gaugeMetric.Sign([]byte(signKey))
	require.NoError(t, errSign)
	gaugeMetric.Hash = signGauge

	signCounter, errSign := counterMetric.Sign([]byte(signKey))
	require.NoError(t, errSign)
	counterMetric.Hash = signCounter

	require.NoError(t, st.Upsert(gaugeMetric))
	require.NoError(t, st.Upsert(counterMetric))

	tests := []struct {
		name          string
		method        string
		contentType   string
		wantStatus    int
		wantError     bool
		wantMetric    metricPkg.Metric
		requestMetric metricPkg.Metric
	}{
		{ // Тело запроса отправляется в сжатом виде и ответ должен быть в сжатом виде
			name:        "Test get gauge -> OK",
			method:      http.MethodPost,
			contentType: ApplicationJSON,
			wantStatus:  http.StatusOK,
			wantError:   false,
			wantMetric:  gaugeMetric,
			requestMetric: metricPkg.Metric{
				ID:    gaugeMetric.ID,
				MType: metricPkg.GaugeType,
			},
		},
		{ // Тело запроса отправляется без сжатия, а ответ должен быть в сжатом виде
			name:        "Test get gauge without GZIP -> OK",
			method:      http.MethodPost,
			contentType: ApplicationJSON,
			wantStatus:  http.StatusOK,
			wantError:   false,
			wantMetric:  gaugeMetric,
			requestMetric: metricPkg.Metric{
				ID:    gaugeMetric.ID,
				MType: metricPkg.GaugeType,
			},
		},
		{ // Запрос без указания заголовка Content-Type
			name:       "Test get gauge without content-type -> ERROR",
			method:     http.MethodPost,
			wantStatus: http.StatusUnsupportedMediaType,
			wantError:  true,
			wantMetric: gaugeMetric,
			requestMetric: metricPkg.Metric{
				ID:    gaugeMetric.ID,
				MType: metricPkg.GaugeType,
			},
		},
		{ // Тело запроса отправляется в сжатом виде и ответ должен быть в сжатом виде
			name:        "Test get counter -> OK",
			method:      http.MethodPost,
			contentType: ApplicationJSON,
			wantStatus:  http.StatusOK,
			wantError:   false,
			wantMetric:  counterMetric,
			requestMetric: metricPkg.Metric{
				ID:    counterMetric.ID,
				MType: metricPkg.CounterType,
			},
		},
		{ // Тело запроса отправляется без сжатия, а ответ должен быть в сжатом виде
			name:        "Test get counter without GZIP -> OK",
			method:      http.MethodPost,
			contentType: ApplicationJSON,
			wantStatus:  http.StatusOK,
			wantError:   false,
			wantMetric:  counterMetric,
			requestMetric: metricPkg.Metric{
				ID:    counterMetric.ID,
				MType: metricPkg.CounterType,
			},
		},
		{ // Запрос без указания заголовка Content-Type
			name:       "Test get gauge without content-type -> ERROR",
			method:     http.MethodPost,
			wantStatus: http.StatusUnsupportedMediaType,
			wantError:  true,
			wantMetric: counterMetric,
			requestMetric: metricPkg.Metric{
				ID:    counterMetric.ID,
				MType: metricPkg.CounterType,
			},
		},
		{ // Запрос с неизвестным типом метрики
			name:        "Test get metric with unknown type -> ERROR",
			method:      http.MethodPost,
			contentType: ApplicationJSON,
			wantStatus:  http.StatusNotFound,
			wantError:   true,
			wantMetric:  counterMetric,
			requestMetric: metricPkg.Metric{
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
			requestMetric: metricPkg.Metric{
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

			nextHandler := handlers.GetAsJSON()
			middleware := handlers.DecompressRequest(nextHandler)

			request := httptest.NewRequest(tt.method, "/value/", bytes.NewReader(encode))

			request.Header.Set(ContentType, tt.contentType)

			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, request)

			response := w.Result()
			defer response.Body.Close()

			require.Equal(t, tt.wantStatus, response.StatusCode)

			if !tt.wantError {
				require.Equal(t, response.Header.Get(ContentType), tt.contentType)

				defer response.Body.Close()

				data, errData := io.ReadAll(response.Body)
				require.NoError(t, errData)

				var metric metricPkg.Metric
				errDecode := json.Unmarshal(data, &metric)
				require.NoError(t, errDecode)

				assert.Equal(t, metric, tt.wantMetric)
			}
		})
	}
}

func TestGetMetric(t *testing.T) {

	logger := logpack.NewLogger()

	gauge, _ := metricPkg.CreateMetric(metricPkg.GaugeType, "testGauge", metricPkg.WithValueFloat(100.023))
	counter, _ := metricPkg.CreateMetric(metricPkg.CounterType, "testCounter", metricPkg.WithValueInt(100))

	st := memstore.New()
	handlers := New(st, logger)

	errUpsert := st.Upsert(gauge)
	require.NoError(t, errUpsert)

	errUpsert = st.Upsert(counter)
	require.NoError(t, errUpsert)

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
				metricType: metricPkg.GaugeType,
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
				metricType: metricPkg.GaugeType,
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
				metricType: metricPkg.CounterType,
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
				metricType: metricPkg.CounterType,
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
			target := "/value/"
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
			h := handlers.GetAsText()
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

func TestUpdateMetricURL(t *testing.T) {

	logger := logpack.NewLogger()

	tests := []struct {
		name        string
		contentType string
		httpMethod  string
		metric      metricPkg.Metric
		wantCode    int
		wantError   bool
	}{
		{
			name: "Success update metric counter -> OK",
			metric: metricPkg.Metric{
				ID:    "testCounter",
				MType: metricPkg.CounterType,
				Delta: randInt64(),
			},
			contentType: "text/plain",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusOK,
			wantError:   false,
		},
		{
			name: "Fail update metric counter - without id, delta -> ERROR",
			metric: metricPkg.Metric{
				MType: metricPkg.CounterType,
			},
			contentType: "text/plain",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusNotFound,
			wantError:   true,
		},
		{
			name: "Success update metric gauge -> OK",
			metric: metricPkg.Metric{
				ID:    "testGauge",
				MType: metricPkg.GaugeType,
				Value: randFloat64(),
			},
			contentType: "text/plain",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusOK,
			wantError:   false,
		},
		{
			name: "Fail update metric gauge - without id, value -> ERROR",
			metric: metricPkg.Metric{
				MType: metricPkg.GaugeType,
			},
			contentType: "text/plain",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusNotFound,
			wantError:   true,
		},
	}
	for _, tt := range tests {

		memoryStorage := memstore.New()
		handlers := New(memoryStorage, logger)

		t.Run(tt.name, func(t *testing.T) {

			target := "/update/"
			if len(tt.metric.MType) > 0 {
				target += tt.metric.MType
			}

			if len(tt.metric.ID) > 0 {
				if target[len(target)-1] != '/' {
					target += "/"
				}

				target += tt.metric.ID
			}

			if len(tt.metric.StringValue()) > 0 {
				if target[len(target)-1] != '/' {
					target += "/"
				}

				target += tt.metric.StringValue()
			}

			request := httptest.NewRequest(tt.httpMethod, target, nil)
			request.Header.Set("Content-Type", tt.contentType)

			w := httptest.NewRecorder()
			h := handlers.UpdateURL()
			h.ServeHTTP(w, request)

			response := w.Result()
			defer response.Body.Close()

			require.Equal(t, tt.wantCode, response.StatusCode)

			if !tt.wantError {
				require.Equal(t, tt.contentType, response.Header.Get("Content-Type"))

				metric, _ := metricPkg.CreateMetric(tt.metric.MType, tt.metric.ID)
				_, err := memoryStorage.Get(metric)
				assert.NoError(t, err)
			}
		})
	}
}
