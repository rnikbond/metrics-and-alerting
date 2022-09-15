package servermetrics

import (
	"fmt"
	"net/http"

	handler "metrics-and-alerting/internal/server/handlers"
	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/config"

	"github.com/go-chi/chi"
)

// StartMetricsHTTPServer Запуск HTTP сервера.
// Настраивается роутер для обработки запросов.
//
// Обрабатываются следующие запросы для получения данных:
// • GET /ping - Возвращает признак работоспособности storage.
// • GET /value/<type>/<id> - Возвращает в теле значение метрики.
// • POST /value | /value/ - Возвращает данные метрики в виде JSON.
//
// Обрабатываются следующие запросы для изменения данных:
// • POST /update/<type>/<id>/<value> - Обновление значения одной метрики, данные передаеются в URL запроса.
// • POST /update | /update/ - Обновление значения одной метрики, данные передаеются в теле запроса в виде JSON.
// • POST /updates | /updates/ - Обновление значений метрик, данные передаеются в теле запроса в виде JSON.
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
