package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	servermetrics "metrics-and-alerting/internal/server"
	"metrics-and-alerting/pkg/config"
)

var cfg config.Config

func prepareConfig() {
	cfg.ReadVarsEnv()

	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "bool - restore metrics")
	flag.StringVar(&cfg.StoreFile, "f", cfg.StoreFile, "string - path to file storage")
	flag.DurationVar(&cfg.StoreInterval, "i", cfg.StoreInterval, "duration - interval store metrics")
	addr := flag.String("a", cfg.Addr, "string - host:port")
	flag.Parse()

	if addr == nil {
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

	waitChan := make(chan struct{})
	server := servermetrics.StartMetricsHTTPServer(&cfg)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
		<-sigChan

		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP server Shutdown: %v\n", err)
		}
		close(waitChan)
	}()

	log.Println("server running ...")
	<-waitChan
	log.Println("stop metrics server")
}
