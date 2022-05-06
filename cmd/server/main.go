package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	servermetrics "metrics-and-alerting/internal/server"
	"metrics-and-alerting/pkg/config"
)

func main() {

	waitChan := make(chan struct{})

	cfg := config.Config{}
	cfg.Read()

	server := servermetrics.StartMetricsHTTPServer(&cfg)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
		<-sigChan

		log.Println("start metrics server")

		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP server Shutdown: %v\n", err)
		}
		close(waitChan)
	}()

	log.Println("server running ...")
	<-waitChan
	log.Println("stop metrics server")
}
