package handler

import (
	"metrics-and-alerting/internal/storage"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateMetric(t *testing.T) {

	storageMetrics := storage.MetricsData{}

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
			name: "test metric handler #1",
			metricData: metricData{
				name:       "RandomValue",
				value:      "1.123",
				metricType: storage.GuageType,
			},
			contentType: "text/plain; charset=utf-8",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusOK,
			wantError:   false,
		},
		{
			name: "test metric handler #2",
			metricData: metricData{
				name:       storage.CounterName,
				value:      "5",
				metricType: storage.CounterType,
			},
			contentType: "text/plain; charset=utf-8",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusOK,
			wantError:   false,
		},
		{
			name:       "test metric handler #3",
			metricData: metricData{},
			httpMethod: http.MethodGet,
			wantCode:   http.StatusMethodNotAllowed,
			wantError:  true,
		},
		{
			name:        "test metric handler #4",
			metricData:  metricData{},
			contentType: "application/json",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusUnsupportedMediaType,
			wantError:   true,
		},
		{
			name: "test metric handler #5",
			metricData: metricData{
				name:       "Ololo",
				value:      "5",
				metricType: storage.CounterType,
			},
			contentType: "text/plain; charset=utf-8",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusBadRequest,
			wantError:   true,
		},
		{
			name: "test metric handler #6",
			metricData: metricData{
				name:       storage.CounterName,
				value:      "5.123",
				metricType: storage.CounterType,
			},
			contentType: "text/plain; charset=utf-8",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusBadRequest,
			wantError:   true,
		},
		{
			name: "test metric handler #7",
			metricData: metricData{
				name:       storage.CounterName,
				value:      "aaaa",
				metricType: storage.CounterType,
			},
			contentType: "text/plain; charset=utf-8",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusBadRequest,
			wantError:   true,
		},
		{
			name: "test metric handler #8",
			metricData: metricData{
				name:       "HeapIdle",
				value:      "4.a",
				metricType: storage.GuageType,
			},
			contentType: "text/plain; charset=utf-8",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusBadRequest,
			wantError:   true,
		},
		{
			name: "test metric handler #9",
			metricData: metricData{
				name:       "HeapIdle",
				value:      "",
				metricType: storage.GuageType,
			},
			contentType: "text/plain; charset=utf-8",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusBadRequest,
			wantError:   true,
		},
		{
			name: "test metric handler #10",
			metricData: metricData{
				name:       "",
				value:      "1.123",
				metricType: storage.GuageType,
			},
			contentType: "text/plain; charset=utf-8",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusBadRequest,
			wantError:   true,
		},
	}
	for _, tt := range tests {

		storageMetrics.Clear()

		t.Run(tt.name, func(t *testing.T) {

			request := httptest.NewRequest(tt.httpMethod, PartURLUpdate+tt.metricData.metricType+"/"+tt.metricData.name+"/"+tt.metricData.value, nil)
			request.Header.Set("Content-Type", tt.contentType)

			w := httptest.NewRecorder()
			h := http.HandlerFunc(UpdateMetric(&storageMetrics))
			h.ServeHTTP(w, request)

			response := w.Result()
			defer response.Body.Close()

			require.Equal(t, tt.wantCode, response.StatusCode)

			if !tt.wantError {
				require.Equal(t, tt.contentType, response.Header.Get("Content-Type"))

				if tt.metricData.metricType == storage.CounterType {
					assert.Contains(t, storageMetrics.GetCounters(), tt.metricData.name)
				} else {
					assert.Contains(t, storageMetrics.GetGauges(), tt.metricData.name)
				}

			}
		})
	}
}
