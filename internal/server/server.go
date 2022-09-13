package server

import (
	"context"
	"fmt"
	"net/http"

	handler "metrics-and-alerting/internal/server/handlers"

	"github.com/go-chi/chi"
)

type MetricsServer struct {
	HTTP *http.Server
}

func NewServer(addr string, h *handler.Handler) *MetricsServer {

	r := chi.NewRouter()
	//r.Use(h.DecompressRequest)
	//r.Use(middleware.Logger)

	r.Get("/ping", h.Ping())
	r.Get("/ping/", h.Ping())

	r.Get("/", h.GetMetrics())
	r.Get("/value/*", h.GetAsText())
	r.Post("/value", h.GetAsJSON())
	r.Post("/value/", h.GetAsJSON())

	r.Post("/update/*", h.UpdateURL())
	r.Post("/update", h.UpdateJSON())
	r.Post("/update/", h.UpdateJSON())
	//r.Post("/updates", h.UpdateDataJSON())
	//r.Post("/updates/", h.UpdateDataJSON())

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
