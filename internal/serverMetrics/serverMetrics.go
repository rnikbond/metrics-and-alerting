package serverMetrics

import (
	"fmt"
	"net/http"

	handler "github.com/rnikbond/metrics-and-alerting/internal/serverMetrics/handlers/metricHandler"
	storage "github.com/rnikbond/metrics-and-alerting/internal/storage"
)

var (
	metrics = storage.MetricsData{}
)

func StartMetricsHttpServer() *http.Server {

	http.HandleFunc(handler.GaugeUrlPart, handler.UpdateMetricGauge(&metrics))
	http.HandleFunc(handler.CounterUrlPart, handler.UpdateMetricCounter(&metrics))

	serverHttp := &http.Server{Addr: ":8080"}

	go func() {
		if err := serverHttp.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("HTTP server ListenAndServe: %v", err)
		}
	}()

	return serverHttp
}
