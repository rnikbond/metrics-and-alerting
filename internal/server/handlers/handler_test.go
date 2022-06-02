package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"metrics-and-alerting/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//const (
//	signKey = "KeySignMetric"
//)

//func StorageWithData() (storage.Storager, error) {
//
//	gauge, err := storage.CreateMetric(storage.GaugeType, "testGauge", 111.111)
//	if err != nil {
//		return nil, err
//	}
//
//	counter, err := storage.CreateMetric(storage.CounterType, "testCounter", 222)
//	if err != nil {
//		return nil, err
//	}
//
//	store := storage.InMemoryStorage{}
//
//	if err := store.Update(gauge); err != nil {
//		return nil, err
//	}
//	if err := store.Update(counter); err != nil {
//		return nil, err
//	}
//
//	return &store, nil
//}
//
//func randFloat64() *float64 {
//	rand.Seed(time.Now().UnixNano())
//	val := rand.Float64()
//	return &val
//}

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

// TestGetMetricGZip - Тест на получение метрики, сжатой GZip
//func TestGetMetricGZip(t *testing.T) {
//
//	st, err := StorageWithData()
//	require.NoError(t, err)
//
//	gaugeMetric := storage.Metric{
//		ID:    "testGauge",
//		MType: storage.GaugeType,
//	}
//
//	counterMetric := storage.Metric{
//		ID:    "testCounter",
//		MType: storage.CounterType,
//	}
//
//	gaugeMetric, err = st.Get(gaugeMetric)
//	require.NoError(t, err)
//
//	counterMetric, err = st.Get(counterMetric)
//	require.NoError(t, err)
//
//	// cases
//
//	tests := []struct {
//		name           string
//		acceptEncoding string
//		method         string
//		metric         storage.Metric
//	}{
//		{
//			name:           "Test without errors -> OK",
//			acceptEncoding: GZip,
//			method:         http.MethodGet,
//			metric:         gaugeMetric,
//		},
//	}
//
//	for _, tt := range tests {
//
//		t.Run(tt.name, func(t *testing.T) {
//			log.Println("")
//		})
//	}
//}

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
