package servermetrics

import (
	"fmt"
	"net/http"

	handler "metrics-and-alerting/internal/server/handlers"
	storage "metrics-and-alerting/internal/storage"
)

var (
	metrics = storage.MetricsData{}
)

func StartMetricsHTTPServer() *http.Server {

	http.HandleFunc(handler.PartURLUpdate, handler.UpdateMetric(&metrics))

	serverHTTP := &http.Server{Addr: ":8080"}

	go func() {
		if err := serverHTTP.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("HTTP server ListenAndServe: %v", err)
		}
	}()

	return serverHTTP
}
