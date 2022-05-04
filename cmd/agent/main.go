package main

import (
	"context"
	"os/signal"
	"syscall"

	"metrics-and-alerting/internal/service/agent"
	"metrics-and-alerting/internal/storage"
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	agent := agent.Agent{
		ServerURL:      "http://127.0.0.1:8080/update",
		PollInterval:   2,  // 2
		ReportInterval: 10, // 10
		Storage:        &storage.MemoryStorage{},
	}

	// Запуск агента сбора и отправки метрик
	agent.Start(ctx)

	<-ctx.Done()
	stop()
}
