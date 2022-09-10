package main

import (
	"context"
	"fmt"
	"os/signal"
	"strings"
	"syscall"

	"metrics-and-alerting/internal/agent"
	"metrics-and-alerting/internal/storage/memoryStorage"
	"metrics-and-alerting/pkg/logpack"
)

func ReadyConfig(logger *logpack.LogPack) *agent.Config {

	cfg := agent.DefaultConfig()

	if err := cfg.ParseFlags(); err != nil {
		logger.Fatal.Fatalf("error argv: %v\n", err)
	}

	cfg.ReadEnvironment()
	if !strings.Contains(cfg.Addr, "http://") {
		cfg.Addr = "http://" + cfg.Addr
	}

	fmt.Println(cfg)
	return cfg
}

func main() {

	logger := logpack.NewLogger()
	cfg := ReadyConfig(logger)
	inMemory := memoryStorage.NewStorage()

	agentService := agent.NewAgent(
		inMemory,
		agent.WithPollInterval(cfg.PollInterval),
		agent.WithReportInterval(cfg.ReportInterval),
		agent.WithAddr(cfg.Addr),
		agent.WithLogger(logger),
		agent.WithReportURL(cfg.ReportURL),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	if err := agentService.Start(ctx); err != nil {
		logger.Fatal.Fatalf("could not start agent: %v\n", err)
	}

	<-ctx.Done()
	stop()
}
