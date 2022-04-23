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
		// {
		// 	name: "test metric handler #1",
		// 	metricData: metricData{
		// 		name:       "RandomValue",
		// 		value:      "1.123",
		// 		metricType: storage.GuageType,
		// 	},
		// 	contentType: "text/plain; charset=utf-8",
		// 	httpMethod:  http.MethodPost,
		// 	wantCode:    http.StatusOK,
		// 	wantError:   false,
		// },
		// {
		// 	name: "test metric handler #2",
		// 	metricData: metricData{
		// 		name:       "testCounter",
		// 		value:      "5",
		// 		metricType: storage.CounterType,
		// 	},
		// 	contentType: "text/plain; charset=utf-8",
		// 	httpMethod:  http.MethodPost,
		// 	wantCode:    http.StatusOK,
		// 	wantError:   false,
		// },
		// {
		// 	name:       "test metric handler #3",
		// 	metricData: metricData{},
		// 	httpMethod: http.MethodGet,
		// 	wantCode:   http.StatusMethodNotAllowed,
		// 	wantError:  true,
		// },
		// {
		// 	name:        "test metric handler #4",
		// 	metricData:  metricData{},
		// 	contentType: "application/json",
		// 	httpMethod:  http.MethodPost,
		// 	wantCode:    http.StatusUnsupportedMediaType,
		// 	wantError:   true,
		// },
		// {
		// 	name: "test metric handler #5",
		// 	metricData: metricData{
		// 		name:       "Ololo",
		// 		value:      "5",
		// 		metricType: storage.CounterType,
		// 	},
		// 	contentType: "text/plain; charset=utf-8",
		// 	httpMethod:  http.MethodPost,
		// 	wantCode:    http.StatusOK,
		// 	wantError:   false,
		// },
		// {
		// 	name: "test metric handler #6",
		// 	metricData: metricData{
		// 		name:       "testCounter",
		// 		value:      "5.123",
		// 		metricType: storage.CounterType,
		// 	},
		// 	contentType: "text/plain; charset=utf-8",
		// 	httpMethod:  http.MethodPost,
		// 	wantCode:    http.StatusUnsupportedMediaType,
		// 	wantError:   true,
		// },
		// {
		// 	name: "test metric handler #7",
		// 	metricData: metricData{
		// 		name:       "testCounter",
		// 		value:      "aaaa",
		// 		metricType: storage.CounterType,
		// 	},
		// 	contentType: "text/plain; charset=utf-8",
		// 	httpMethod:  http.MethodPost,
		// 	wantCode:    http.StatusUnsupportedMediaType,
		// 	wantError:   true,
		// },
		// {
		// 	name: "test metric handler #8",
		// 	metricData: metricData{
		// 		name:       "HeapIdle",
		// 		value:      "4.a",
		// 		metricType: storage.GuageType,
		// 	},
		// 	contentType: "text/plain; charset=utf-8",
		// 	httpMethod:  http.MethodPost,
		// 	wantCode:    http.StatusUnsupportedMediaType,
		// 	wantError:   true,
		// },
		// {
		// 	name: "test metric handler #9",
		// 	metricData: metricData{
		// 		name:       "HeapIdle",
		// 		value:      "",
		// 		metricType: storage.GuageType,
		// 	},
		// 	contentType: "text/plain; charset=utf-8",
		// 	httpMethod:  http.MethodPost,
		// 	wantCode:    http.StatusNotFound,
		// 	wantError:   true,
		// },
		// {
		// 	name: "test metric handler #10",
		// 	metricData: metricData{
		// 		name:       "testCounter",
		// 		value:      "100",
		// 		metricType: storage.CounterType,
		// 	},
		// 	contentType: "text/plain; charset=utf-8",
		// 	httpMethod:  http.MethodPost,
		// 	wantCode:    http.StatusOK,
		// 	wantError:   false,
		// },
		// {
		// 	name: "test metric handler #11",
		// 	metricData: metricData{
		// 		name:       "",
		// 		value:      "",
		// 		metricType: storage.CounterType,
		// 	},
		// 	contentType: "text/plain; charset=utf-8",
		// 	httpMethod:  http.MethodPost,
		// 	wantCode:    http.StatusNotFound,
		// 	wantError:   true,
		// },
		// {
		// 	name: "test metric handler #12",
		// 	metricData: metricData{
		// 		name:       "testCounter",
		// 		value:      "none",
		// 		metricType: storage.CounterType,
		// 	},
		// 	contentType: "text/plain; charset=utf-8",
		// 	httpMethod:  http.MethodPost,
		// 	wantCode:    http.StatusUnsupportedMediaType,
		// 	wantError:   true,
		// },
		// {
		// 	name: "test metric handler #13",
		// 	metricData: metricData{
		// 		name:       "testGauge",
		// 		value:      "100",
		// 		metricType: storage.GuageType,
		// 	},
		// 	contentType: "text/plain; charset=utf-8",
		// 	httpMethod:  http.MethodPost,
		// 	wantCode:    http.StatusOK,
		// 	wantError:   false,
		// },
		// {
		// 	name: "test metric handler #14",
		// 	metricData: metricData{
		// 		name:       "",
		// 		value:      "100",
		// 		metricType: storage.GuageType,
		// 	},
		// 	contentType: "text/plain; charset=utf-8",
		// 	httpMethod:  http.MethodPost,
		// 	wantCode:    http.StatusNotFound,
		// 	wantError:   true,
		// },
		// {
		// 	name: "test metric handler #15",
		// 	metricData: metricData{
		// 		name:       "testGauge",
		// 		value:      "none",
		// 		metricType: storage.GuageType,
		// 	},
		// 	contentType: "text/plain; charset=utf-8",
		// 	httpMethod:  http.MethodPost,
		// 	wantCode:    http.StatusUnsupportedMediaType,
		// 	wantError:   true,
		// },
		// {
		// 	name: "test metric handler #16",
		// 	metricData: metricData{
		// 		name:       "testGauge",
		// 		value:      "100",
		// 		metricType: "unknown",
		// 	},
		// 	contentType: "text/plain; charset=utf-8",
		// 	httpMethod:  http.MethodPost,
		// 	wantCode:    http.StatusNotImplemented,
		// 	wantError:   true,
		// },
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
				metricType: storage.GuageType,
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
				metricType: storage.GuageType,
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
				metricType: storage.GuageType,
			},
			contentType: "text/plain",
			httpMethod:  http.MethodPost,
			wantCode:    http.StatusBadRequest,
			wantError:   true,
		},
	}
	for _, tt := range tests {

		storageMetrics.Clear()

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
