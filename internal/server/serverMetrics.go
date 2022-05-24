package servermetrics

import (
	"fmt"
	"net/http"

	handler "metrics-and-alerting/internal/server/handlers"
	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/config"

	"github.com/go-chi/chi"
)

func StartMetricsHTTPServer(memStore *storage.MemoryStorage, cfg *config.Config) *http.Server {

	r := chi.NewRouter()
	r.Use(handler.GZipHandle)
	r.Get("/", handler.GetMetrics(memStore))
	r.Get(handler.PartURLPing, handler.CheckHealthStorage(memStore))
	r.Get(handler.PartURLValue+"/*", handler.GetMetric(memStore))
	r.Post(handler.PartURLValue, handler.GetMetricJSON(memStore))
	r.Post(handler.PartURLValue+"/", handler.GetMetricJSON(memStore))
	r.Post(handler.PartURLUpdate, handler.UpdateMetricJSON(memStore))
	r.Post(handler.PartURLUpdate+"/", handler.UpdateMetricJSON(memStore))
	r.Post(handler.PartURLUpdate+"/*", handler.UpdateMetricURL(memStore))

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
