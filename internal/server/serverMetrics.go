package servermetrics

import (
	"fmt"
	"net/http"

	handler "metrics-and-alerting/internal/server/handlers"
	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/config"

	"github.com/go-chi/chi"
)

func StartMetricsHTTPServer(cfg *config.Config) *http.Server {

	memoryStorage := storage.MemoryStorage{}

	r := chi.NewRouter()
	r.Get("/", handler.GetMetrics(&memoryStorage))
	r.Get(handler.PartURLValue+"/*", handler.GetMetric(&memoryStorage))
	r.Post(handler.PartURLValue, handler.GetMetricJSON(&memoryStorage))
	r.Post(handler.PartURLValue+"/", handler.GetMetricJSON(&memoryStorage))
	r.Post(handler.PartURLUpdate, handler.UpdateMetricJSON(&memoryStorage))
	r.Post(handler.PartURLUpdate+"/", handler.UpdateMetricJSON(&memoryStorage))
	r.Post(handler.PartURLUpdate+"/*", handler.UpdateMetricURL(&memoryStorage))

	serverHTTP := &http.Server{
		Addr:    cfg.Addr,
		Handler: r,
	}

	go func() {
		if err := serverHTTP.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("HTTP server ListenAndServe: %v\n", err)
		}
	}()

	return serverHTTP
}
