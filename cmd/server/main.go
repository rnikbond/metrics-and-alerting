package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	serverMetrics "metrics-and-alerting/internal/server"
)

func main() {

	waitChan := make(chan struct{})
	server := serverMetrics.StartMetricsHttpServer()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
		<-sigChan

		if err := server.Shutdown(context.Background()); err != nil {
			fmt.Printf("HTTP server Shutdown: %v\n", err)
		}
		close(waitChan)
	}()

	<-waitChan
}
