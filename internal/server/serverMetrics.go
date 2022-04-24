package servermetrics

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"

	handler "metrics-and-alerting/internal/server/handlers"
	storage "metrics-and-alerting/internal/storage"
)

var (
	metrics = storage.MetricsData{}
)

func StartMetricsHTTPServer() *http.Server {

	r := chi.NewRouter()
	r.Get("/", handler.GetMetrics(&metrics))
	r.Get(handler.PartURLValue+"*", handler.GetMetric(&metrics))
	r.Post(handler.PartURLUpdate+"*", handler.UpdateMetric(&metrics))
	serverHTTP := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		if err := serverHTTP.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("HTTP server ListenAndServe: %v", err)
		}
	}()

	return serverHTTP
}
