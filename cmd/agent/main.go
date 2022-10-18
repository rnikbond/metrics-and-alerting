package main

import (
	"context"
	"fmt"
	"os/signal"
	"strings"
	"syscall"

	"metrics-and-alerting/internal/agent"
	"metrics-and-alerting/internal/storage/memstore"
	"metrics-and-alerting/pkg/logpack"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
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

func init() {

	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}

func main() {

	logger := logpack.NewLogger()
	cfg := ReadyConfig(logger)
	inMemory := memstore.New()

	agentService := agent.NewAgent(
		inMemory,
		agent.WithPollInterval(cfg.PollInterval.Duration),
		agent.WithReportInterval(cfg.ReportInterval.Duration),
		agent.WithAddr(cfg.Addr),
		agent.WithLogger(logger),
		agent.WithReportURL(cfg.ReportURL),
		agent.WithSignKey([]byte(cfg.SecretKey)),
		agent.WithKey([]byte(cfg.CryptoKey)),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	if err := agentService.Start(ctx); err != nil {
		logger.Fatal.Fatalf("could not start agent: %v\n", err)
	}

	<-ctx.Done()
	stop()
}
