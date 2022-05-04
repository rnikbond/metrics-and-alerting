package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"metrics-and-alerting/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateMetric(t *testing.T) {

	st := storage.MemoryStorage{}

	type metricData struct {
		name       string
		value      string
		metricType string
	}

	tests := []struct {
		name       string
		metricData metricData

		contentType string
		httpMethod  string
		wantCode    int
		wantError   bool
	}{
		{
			name: "TestIteration2/TestCounterHandlers/update",
			metricData: metricData{
				name:       "testGauge",
				value:      "100",
				metricType: storage.CounterType,
			},
			contentType: "text/plain",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusOK,
			wantError:   false,
		},
		{
			name: "TestIteration2/TestCounterHandlers/without_id",
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
			name: "TestIteration2/TestCounterHandlers/invalid_value",
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
			name: "TestIteration2/TestCounterHandlers/invalid_value",
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
			name: "TestIteration2/TestGaugeHandlers/update",
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
			name: "TestIteration2/TestGaugeHandlers/without_id",
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
			name: "TestIteration2/TestGaugeHandlers/invalid_value",
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

		st.Clear()

		t.Run(tt.name, func(t *testing.T) {

			target := PartURLUpdate
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
			h := http.HandlerFunc(UpdateMetric(&st))
			h.ServeHTTP(w, request)

			response := w.Result()
			defer response.Body.Close()

			require.Equal(t, tt.wantCode, response.StatusCode)

			if !tt.wantError {
				require.Equal(t, tt.contentType, response.Header.Get("Content-Type"))

				_, err := st.Get(tt.metricData.metricType, tt.metricData.name)
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetMetric(t *testing.T) {

	st := storage.MemoryStorage{}
	st.Set(storage.GaugeType, "testGauge", 100.023)
	st.Set(storage.CounterType, "testCounter", 100)

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
			name: "TestIteration2/TestGaugeGetHandlers/value",
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
			name: "TestIteration2/TestGaugeGetHandlers/without_id",
			metricData: metricData{
				metricType: storage.GaugeType,
			},
			contentType: "text/plain",
			httpMethod:  http.MethodGet,
			wantCode:    http.StatusNotFound,

			wantError: true,
		},
		{
			name: "TestIteration2/TestGaugeGetHandlers/uncorrect_type",
			metricData: metricData{
				name: "testGauge",
			},
			contentType: "text/plain",
			httpMethod:  http.MethodGet,
			wantCode:    http.StatusNotFound,

			wantError: true,
		},
		{
			name:        "TestIteration2/TestGaugeGetHandlers/without_id_type",
			metricData:  metricData{},
			contentType: "text/plain",
			httpMethod:  http.MethodGet,
			wantCode:    http.StatusNotFound,

			wantError: true,
		},
		{
			name: "TestIteration2/TestCounterGetHandlers/value",
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
			name: "TestIteration2/TestCounterGetHandlers/without_id",
			metricData: metricData{
				metricType: storage.CounterType,
			},
			contentType: "text/plain",
			httpMethod:  http.MethodGet,
			wantCode:    http.StatusNotFound,

			wantError: true,
		},
		{
			name: "TestIteration2/TestCounterGetHandlers/without_type",
			metricData: metricData{
				name: "testCounter",
			},
			contentType: "text/plain",
			httpMethod:  http.MethodGet,
			wantCode:    http.StatusNotFound,

			wantError: true,
		},
		{
			name:        "TestIteration2/TestCounterGetHandlers/without_type_id",
			metricData:  metricData{},
			contentType: "text/plain",
			httpMethod:  http.MethodGet,
			wantCode:    http.StatusNotFound,

			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := PartURLValue
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
			h := http.HandlerFunc(GetMetric(&st))
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
	st := storage.MemoryStorage{}
	st.Set(storage.GaugeType, "testGauge1", 100.023)

	type metricData struct {
		name       string
		value      string
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
			name: "TestIteration2/TestIndexPage/",
			metricData: metricData{
				name:       "testGauge1",
				value:      "100.023",
				metricType: storage.GaugeType,
			},
			contentType: "text/html",
			httpMethod:  http.MethodGet,
			wantCode:    http.StatusOK,
			wantValue:   "testGauge1 100.023 <br/>",
			wantError:   true,
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

			if !tt.wantError {
				require.NoError(t, err)
				require.Equal(t, tt.contentType, response.Header.Get("Content-Type"))
				assert.Equal(t, string(got), tt.wantValue)
			}

		})
	}
}
