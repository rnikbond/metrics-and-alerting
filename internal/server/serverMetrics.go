package servermetrics

import (
	"fmt"
	"net/http"

	handler "metrics-and-alerting/internal/server/handlers"
	"metrics-and-alerting/internal/storage"

	"github.com/caarlos0/env"
	"github.com/go-chi/chi"
)

type Config struct {
	Addr string `env:"ADDRESS"`
}

func StartMetricsHTTPServer() *http.Server {

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		cfg.Addr = "127.0.0.1:8080"
	}

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
			fmt.Printf("HTTP server ListenAndServe: %v", err)
		}
	}()

	return serverHTTP
}
