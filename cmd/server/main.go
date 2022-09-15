package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	servermetrics "metrics-and-alerting/internal/server"
	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/config"
)

var cfg config.Config

func parseFlags() {

	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "bool - restore metrics")
	flag.StringVar(&cfg.StoreFile, "f", cfg.StoreFile, "string - path to file storage")
	flag.DurationVar(&cfg.StoreInterval, "i", cfg.StoreInterval, "duration - interval store metrics")
	flag.StringVar(&cfg.SecretKey, "k", cfg.SecretKey, "string - key sign")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "string - database data source name")
	flag.BoolVar(&cfg.VerifyOnUpdate, "vu", cfg.VerifyOnUpdate, "bool - verify changes")
	flag.StringVar(&cfg.PprofAddr, "pa", cfg.PprofAddr, "pprof address - for run profiler")

	addr := flag.String("a", cfg.Addr, "string - host:port")
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

func prepareConfig() {
	cfg.SetDefault()
	parseFlags()
	cfg.ReadEnvVars()
}

func runProfiler(addr string) {
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Printf("error start profiler server: %v\n", err)
		}
	}()
}

func main() {

	prepareConfig()
	fmt.Println(cfg)

	runProfiler(cfg.PprofAddr)

	var store storage.Storager

	if cfg.DatabaseDSN != "" {
		log.Println("using storage: DataBase")
		store = &storage.DataBaseStorage{}
	} else if cfg.StoreFile != "" {
		store = &storage.FileStorage{}
		log.Println("using storage: File")
	} else {
		store = &storage.InMemoryStorage{}
		log.Println("using storage: Memory")
	}

	if err := store.Init(cfg); err != nil {
		log.Fatalf("can not init storage: %v", err)
	}

	waitChan := make(chan struct{})
	server := servermetrics.StartMetricsHTTPServer(store, cfg)

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
