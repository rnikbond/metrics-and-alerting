package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"

	"metrics-and-alerting/internal/server"
	handler "metrics-and-alerting/internal/server/handlers"
	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/internal/storage/filestorage"
	"metrics-and-alerting/internal/storage/memorystorage"
	"metrics-and-alerting/pkg/logpack"
)

var (
	_ storage.Repository = (*server.MetricsManager)(nil)
	_ storage.Repository = (*memorystorage.MemoryStorage)(nil)
	_ storage.Repository = (*filestorage.Storage)(nil)
)

func main() {

	logger := logpack.NewLogger()
	cfg := server.DefaultConfig()

	if err := cfg.ParseFlags(); err != nil {
		logger.Fatal.Fatalf("error argv: %v\n", err)
	}

	cfg.ReadEnvVars()
	fmt.Println(cfg)

	var store storage.Repository
	if cfg.DatabaseDSN != "" {
		//store = &storage.DataBaseStorage{}
		store = memorystorage.NewStorage()
	} else if cfg.StoreFile != "" {
		fs := filestorage.New(cfg.StoreFile, cfg.StoreInterval, logger)
		if cfg.Restore {
			fs.Restore()
		}

		store = fs
		log.Println("using storage: File")
	} else {
		store = memorystorage.NewStorage()
		log.Println("using storage: Memory")
	}

	storeManager := server.NewMetricsManager(
		store,
		server.WithSignKey([]byte(cfg.SecretKey)),
	)

	handlers := handler.New(storeManager, logger)

	serv := server.NewServer(cfg.Addr, handlers)
	serv.Start()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	<-ctx.Done()
	stop()

	// TODO :: Нужно ли здесь создавать новый контекст
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := serv.Shutdown(ctx); err != nil {
		logger.Err.Printf("HTTP server Shutdown: %v\n", err)
	}
	cancel()

}
