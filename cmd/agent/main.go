package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"metrics-and-alerting/internal/service/agent"
	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/config"
)

var cfg config.Config

func parseFlags() {

	flag.DurationVar(&cfg.ReportInterval, "r", cfg.ReportInterval, "duration - report interval")
	flag.DurationVar(&cfg.PollInterval, "p", cfg.PollInterval, "duration - poll interval")
	flag.StringVar(&cfg.SecretKey, "k", cfg.SecretKey, "string - key crypto")
	flag.BoolVar(&cfg.VerifyOnUpdate, "vu", cfg.VerifyOnUpdate, "bool - verify changes")
	flag.StringVar(&cfg.ReportType, "rt", cfg.ReportType, fmt.Sprint("support types: ",
		config.ReportURL, "/", config.ReportJSON, "/", config.ReportBatchJSON))
	addr := flag.String("a", cfg.Addr, "ip address: ip:port")
	flag.Parse()

	if addr == nil || *addr == "" {
		return
	}

	parsedAddr := strings.Split(*addr, ":")
	if len(parsedAddr) != 2 {
		log.Println("need address in a form host:port")
		return
	}

	if parsedAddr[0] != "localhost" {
		if ip := net.ParseIP(parsedAddr[0]); ip == nil {
			log.Println("incorrect ip: " + parsedAddr[0])
			return
		}
	}

	if _, err := strconv.Atoi(parsedAddr[1]); err != nil {
		log.Println("incorrect port: " + parsedAddr[1])
		return
	}

	cfg.Addr = *addr
}

func main() {

	cfg.SetDefault()
	parseFlags()
	cfg.ReadEnvVars()

	cfg.VerifyOnUpdate = false

	fmt.Println(cfg)

	memoryStore := storage.InMemoryStorage{}
	if err := memoryStore.Init(cfg); err != nil {
		log.Fatalf("can not run agent service: %v\n", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	agentService := agent.Agent{}
	agentService.Init(cfg, &memoryStore)

	if err := agentService.Start(ctx); err != nil {
		log.Fatalf("can not run agent service: %v\n", err)
	}

	<-ctx.Done()
	stop()
}
