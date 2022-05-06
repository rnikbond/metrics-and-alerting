package main

import (
	"context"
	"os/signal"
	"syscall"

	"metrics-and-alerting/internal/service/agent"
	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/config"
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	cfg := config.Config{}
	cfg.Read()

	agent := agent.Agent{
		Config:  &cfg,
		Storage: &storage.MemoryStorage{},
	}

	// Запуск агента сбора и отправки метрик
	agent.Start(ctx)

	<-ctx.Done()
	stop()
}
