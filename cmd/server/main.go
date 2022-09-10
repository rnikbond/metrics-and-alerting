package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"metrics-and-alerting/internal/server"
	handler "metrics-and-alerting/internal/server/handlers"
	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/internal/storage/memoryStorage"
	"metrics-and-alerting/pkg/logpack"
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

	//if cfg.DatabaseDSN != "" {
	//	//store = &storage.DataBaseStorage{}
	//} else if cfg.StoreFile != "" {
	//	//store = &storage.FileStorage{}
	//	log.Println("using storage: File")
	//} else {
	//	store = memoryStorage.NewStorage()
	//	log.Println("using storage: Memory")
	//}

	store = memoryStorage.NewStorage()

	storeManager := storage.NewMetricsManager(store)
	handlers := handler.New(storeManager, logger)

	serv := server.NewServer(cfg.Addr, storeManager, handlers)
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
