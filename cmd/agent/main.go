package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"metrics-and-alerting/internal/service/agent"
	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/config"
)

var cfg config.Config

func prepareConfig() {
	cfg.ReadVarsEnv()

	reportInterval := flag.Int64("r", int64(cfg.ReportInterval.Seconds()), "report interval")
	pollInterval := flag.Int64("p", int64(cfg.PollInterval.Seconds()), "poll interval")
	addr := flag.String("a", cfg.Addr, "ip address: ip:port")
	flag.Parse()

	cfg.ReportInterval = time.Duration(*reportInterval) * time.Second
	cfg.PollInterval = time.Duration(*pollInterval) * time.Second

	if addr != nil {
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

	prepareConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	agent := agent.Agent{
		Config:  &cfg,
		Storage: &storage.MemoryStorage{},
	}

	// Запуск агента сбора и отправки метрик
	agent.Start(ctx)

	<-ctx.Done()
	stop()
}
