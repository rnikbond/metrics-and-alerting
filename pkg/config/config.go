package config

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env"
)

type Config struct {
	Addr           string        `env:"ADDRESS"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
	PollInterval   time.Duration `env:"POLL_INTERVAL"`
	StoreInterval  time.Duration `env:"STORE_INTERVAL"`
	StoreFile      string        `env:"STORE_FILE"`
	Restore        bool          `env:"RESTORE"`
}

// SetDefault Инициализация значений по умолчанию
func (cfg *Config) SetDefault() {

	cfg.Addr = "127.0.0.1:8080"
	cfg.PollInterval = 2 * time.Second
	cfg.ReportInterval = 10 * time.Second
	cfg.Restore = true
	cfg.StoreInterval = 300 * time.Second
	cfg.StoreFile = "/tmp/devops-metrics-db.json"
}

func (cfg *Config) String() string {
	s := "ADDRESS: " + cfg.Addr + "\n"
	s += "REPORT_INTERVAL: " + cfg.ReportInterval.String() + "\n"
	s += "POLL_INTERVAL: " + cfg.ReportInterval.String() + "\n"
	s += "STORE_INTERVAL: " + cfg.StoreInterval.String() + "\n"
	s += "STORE_FILE: " + cfg.StoreFile + "\n"
	s += "RESTORE: " + strconv.FormatBool(cfg.Restore) + "\n"

	return s
}

func (cfg *Config) ReadEnvVars() {

	// Чтение переменных среды
	if err := env.Parse(cfg); err != nil {
		log.Println(err)
	}

	// Убираем пробелы из адреса
	cfg.Addr = strings.TrimSpace(cfg.Addr)
}
