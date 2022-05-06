package config

import (
	"log"
	"strings"
	"time"

	"github.com/caarlos0/env"
)

type Config struct {
	Addr           string        `env:"ADDRESS"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
	PollInterval   time.Duration `env:"POLL_INTERVAL"`
}

func (cfg *Config) Read() {

	cfg.Addr = "127.0.0.1:8080"
	cfg.PollInterval = 2 * time.Second
	cfg.ReportInterval = 10 * time.Second

	if err := env.Parse(cfg); err != nil {
		log.Println(err)
	}

	cfg.Addr = strings.TrimSpace(cfg.Addr)
}
