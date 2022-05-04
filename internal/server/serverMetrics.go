package servermetrics

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"

	handler "metrics-and-alerting/internal/server/handlers"
	storage "metrics-and-alerting/internal/storage"
)

func StartMetricsHTTPServer() *http.Server {

	memoryStorage := storage.MemoryStorage{}

	r := chi.NewRouter()
	r.Get("/", handler.GetMetrics(&memoryStorage))
	r.Get(handler.PartURLValue+"*", handler.GetMetric(&memoryStorage))
	r.Post(handler.PartURLUpdate+"*", handler.UpdateMetric(&memoryStorage))

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
