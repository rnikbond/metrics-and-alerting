package server

import (
	"context"
	"fmt"
	"net/http"

	handler "metrics-and-alerting/internal/server/handlers"
	"metrics-and-alerting/internal/storage"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

type MetricsServer struct {
	HTTP *http.Server
}

func NewServer(addr string, store storage.Repository, h *handler.Handler) *MetricsServer {

	r := chi.NewRouter()
	r.Use(h.DecompressRequest)
	r.Use(middleware.Logger)

	r.Get("/ping", h.Ping())
	r.Get("/ping/", h.Ping())

	r.Get("/", h.GetMetrics())
	r.Get("/value", h.Get())
	r.Get("/value/", h.Get())
	r.Get("/value/*", h.Get())

	r.Post("/update/*", handler.UpdateURL(store))
	r.Post("/update", handler.UpdateJSON(store))
	r.Post("/updates", handler.UpdateDataJSON(store))
	r.Post("/updates/", handler.UpdateDataJSON(store))

	serv := &MetricsServer{
		HTTP: &http.Server{
			Addr:    addr,
			Handler: r,
		},
	}

	return serv
}

func (serv *MetricsServer) Start() {
	go func() {
		if err := serv.HTTP.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("HTTP server ListenAndServe: %v\n", err)
		}
	}()
}

func (serv *MetricsServer) Shutdown(ctx context.Context) error {
	return serv.HTTP.Shutdown(ctx)
}
