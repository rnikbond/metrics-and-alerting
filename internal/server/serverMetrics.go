package servermetrics

import (
	"fmt"
	"log"
	"net/http"

	handler "metrics-and-alerting/internal/server/handlers"
	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/config"

	"github.com/go-chi/chi"
)

func restoreFromFile(fileStorage *storage.FileStorage, to storage.IStorage) error {
	if err := fileStorage.Read(); err != nil {
		return err
	}

	types := []string{storage.GaugeType, storage.CounterType}
	for _, typeMetric := range types {
		names := fileStorage.Names(typeMetric)
		for _, id := range names {
			if val, err := fileStorage.Get(typeMetric, id); err == nil {
				to.Set(typeMetric, id, val)
			}
		}
	}

	return nil
}

func StartMetricsHTTPServer(cfg *config.Config) *http.Server {

	memoryStorage := storage.MemoryStorage{}
	fileStorage := storage.FileStorage{
		FileName: cfg.StoreFile,
	}

	if cfg.Restore && len(cfg.StoreFile) > 0 {
		if err := restoreFromFile(&fileStorage, &memoryStorage); err != nil {
			log.Printf("error restore metrics: %s", err.Error())
		}
	}

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
