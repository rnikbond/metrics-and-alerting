package server

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env"
)

type Config struct {
	Addr          string        `env:"ADDRESS"`
	StoreInterval time.Duration `env:"STORE_INTERVAL"`
	Restore       bool          `env:"RESTORE"`
	DatabaseDSN   string        `env:"DATABASE_URI"`
	StoreFile     string        `env:"STORE_FILE"`
	SecretKey     string        `env:"KEY"`
}

// DefaultConfig Конфигурация для сервиса агента со значениями по умолчанию
func DefaultConfig() *Config {

	return &Config{
		Addr:          ":8080",
		StoreInterval: 10 * time.Second,
		Restore:       true,
		DatabaseDSN:   "",
		StoreFile:     "",
		SecretKey:     "",
	}
}

func (cfg *Config) ParseFlags() error {

	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "bool - restore metrics")
	flag.StringVar(&cfg.StoreFile, "f", cfg.StoreFile, "string - path to fileStorage storage")
	flag.DurationVar(&cfg.StoreInterval, "i", cfg.StoreInterval, "duration - interval store metrics")
	flag.StringVar(&cfg.SecretKey, "k", cfg.SecretKey, "string - key sign")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "string - dbstore data source name")

	addr := flag.String("a", cfg.Addr, "string - host:port")
	flag.Parse()

	if addr == nil || *addr == "" {
		return fmt.Errorf("address can not be empty")
	}

	parsedAddr := strings.Split(*addr, ":")
	if len(parsedAddr) != 2 {
		return fmt.Errorf("need address in a format host:port")
	}

	if len(parsedAddr[0]) > 0 && parsedAddr[0] != "localhost" {
		if ip := net.ParseIP(parsedAddr[0]); ip == nil {
			return fmt.Errorf("incorrect ip: " + parsedAddr[0])
		}
	}

	if _, err := strconv.Atoi(parsedAddr[1]); err != nil {
		return fmt.Errorf("incorrect port: " + parsedAddr[1])
	}

	cfg.Addr = *addr
	return nil
}

func (cfg Config) String() string {

	builder := strings.Builder{}

	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("\t ADDRESS: %s\n", cfg.Addr))
	builder.WriteString(fmt.Sprintf("\t STORE_INTERVAL: %s\n", cfg.StoreInterval.String()))
	builder.WriteString(fmt.Sprintf("\t RESTORE: %v\n", cfg.Restore))
	builder.WriteString(fmt.Sprintf("\t DATABASE_DSN: %s\n", cfg.DatabaseDSN))
	builder.WriteString(fmt.Sprintf("\t STORE_FILE: %s\n", cfg.StoreFile))
	builder.WriteString(fmt.Sprintf("\t KEY: %s\n", cfg.SecretKey))

	return builder.String()
}

func (cfg *Config) ReadEnvVars() {

	// Чтение переменных среды
	if err := env.Parse(cfg); err != nil {
		log.Println(err)
	}

	// Убираем пробелы из адреса
	cfg.Addr = strings.TrimSpace(cfg.Addr)
}
