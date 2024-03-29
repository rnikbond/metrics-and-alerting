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
	"metrics-and-alerting/internal/storage/dbstore"
	"metrics-and-alerting/internal/storage/filestorage"
	"metrics-and-alerting/internal/storage/memstore"
	"metrics-and-alerting/pkg/logpack"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

var (
	_ storage.Repository = (*server.MetricsManager)(nil)
	_ storage.Repository = (*memstore.Storage)(nil)
	_ storage.Repository = (*filestorage.Storage)(nil)
	_ storage.Repository = (*dbstore.Storage)(nil)
)

func init() {

	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}

func main() {

	logger := logpack.NewLogger()
	cfg := server.DefaultConfig()

	if err := cfg.ParseFlags(); err != nil {
		logger.Fatal.Fatalf("error argv: %v\n", err)
	}

	cfg.ReadEnvVars()
	fmt.Println(cfg)

	var store storage.Repository
	if len(cfg.DatabaseDSN) != 0 {

		cfg.StoreInterval.Duration = 0
		db, err := dbstore.New(cfg.DatabaseDSN, logger)
		if err != nil {
			panic(err)
		}

		store = db
		logger.Info.Println("Using storage: Database")
	}

	if store == nil && len(cfg.StoreFile) != 0 {
		store = filestorage.New(cfg.StoreFile, logger)
		logger.Info.Println("Using storage: File")
	}

	if store == nil {
		store = memstore.New()
		logger.Info.Println("Using storage: Memory")
	}

	storeManager := server.New(
		store,
		logger,
		server.WithSignKey([]byte(cfg.SecretKey)),
		server.WithFlush(cfg.StoreInterval.Duration),
		server.WithRestore(cfg.Restore),
	)

	handlers := handler.New(storeManager,
		logger,
		handler.WithKey(cfg.CryptoKey),
		handler.WithTrustedSubnet(cfg.TrustedSubnet))

	serv := server.NewHTTPServer(cfg.Addr, handlers)
	serv.Start()
	logger.Info.Println("HTTP server started")

	if len(cfg.AddrRPC) != 0 {
		gServ, errServ := server.NewGRPCServer(cfg.AddrRPC, storeManager)
		if errServ != nil {
			logger.Err.Fatalf("failed create gRPC server: %v\n", errServ)
		}

		gServ.Start()
		logger.Info.Println("gRPC server started")

		defer gServ.Stop()
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	<-ctx.Done()
	stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := serv.Shutdown(ctx); err != nil {
		logger.Err.Printf("HTTP server Shutdown: %v\n", err)
	}
	cancel()

}
