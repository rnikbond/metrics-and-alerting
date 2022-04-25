package main

import (
	"context"
	"metrics-and-alerting/internal/service/agent"
	"metrics-and-alerting/internal/storage"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {

	var waitGroup sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	agent := agent.AgentMeticsData{
		ServerURL:      "http://127.0.0.1:8080/update",
		PollInterval:   2,
		ReportInterval: 10,
		Metrics:        &storage.MetricsData{},
	}

	// Запуск агента сбора и отправки метрик
	agent.Start(ctx, &waitGroup)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	<-sigChan

	cancel()
	waitGroup.Wait()
}
