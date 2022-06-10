package servermetrics

import (
	"fmt"
	"net/http"

	handler "metrics-and-alerting/internal/server/handlers"
	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/config"

	"github.com/go-chi/chi"
)

func StartMetricsHTTPServer(memStore storage.Storager, cfg config.Config) *http.Server {

	r := chi.NewRouter()
	r.Use(handler.GZipHandle)
	r.With()
	r.Get("/", handler.GetMetrics(memStore))
	r.Get(handler.PartURLPing, handler.Ping(memStore))
	r.Get(handler.PartURLValue+"/*", handler.Get(memStore))
	r.Post(handler.PartURLValue, handler.GetJSON(memStore))
	r.Post(handler.PartURLValue+"/", handler.GetJSON(memStore))
	r.Post(handler.PartURLUpdate, handler.UpdateJSON(memStore))
	r.Post(handler.PartURLUpdate+"/", handler.UpdateJSON(memStore))
	r.Post(handler.PartURLUpdate+"/*", handler.UpdateURL(memStore))
	r.Post(handler.PartURLUpdates, handler.UpdateDataJSON(memStore))
	r.Post(handler.PartURLUpdates+"/", handler.UpdateDataJSON(memStore))

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
