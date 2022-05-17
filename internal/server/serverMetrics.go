package servermetrics

import (
	"fmt"
	"net/http"

	handler "metrics-and-alerting/internal/server/handlers"
	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/config"

	"github.com/go-chi/chi"
)

func StartMetricsHTTPServer(st storage.IStorage, cfg *config.Config) *http.Server {

	r := chi.NewRouter()
	r.Use(handler.GZipHandle)
	r.Get("/", handler.GetMetrics(st))
	r.Get(handler.PartURLValue+"/*", handler.GetMetric(st))
	r.Post(handler.PartURLValue, handler.GetMetricJSON(st))
	r.Post(handler.PartURLValue+"/", handler.GetMetricJSON(st))
	r.Post(handler.PartURLUpdate, handler.UpdateMetricJSON(st))
	r.Post(handler.PartURLUpdate+"/", handler.UpdateMetricJSON(st))
	r.Post(handler.PartURLUpdate+"/*", handler.UpdateMetricURL(st))

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
