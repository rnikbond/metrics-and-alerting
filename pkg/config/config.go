package config

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
	"text/tabwriter"
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
	SecretKey      string        `env:"KEY"`
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

func (cfg Config) String() string {

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "ADDRESS\t", cfg.Addr)
	fmt.Fprintln(w, "REPORT_INTERVAL\t", cfg.ReportInterval.String())
	fmt.Fprintln(w, "POLL_INTERVAL\t", cfg.PollInterval.String())
	fmt.Fprintln(w, "STORE_INTERVAL\t", cfg.StoreInterval.String())
	fmt.Fprintln(w, "STORE_FILE\t", cfg.StoreFile)
	fmt.Fprintln(w, "RESTORE\t", strconv.FormatBool(cfg.Restore))
	fmt.Fprintln(w, "KEY\t", cfg.SecretKey)

	if err := w.Flush(); err != nil {
		return err.Error()
	}

	return buf.String()
}

func (cfg *Config) ReadEnvVars() {

	// Чтение переменных среды
	if err := env.Parse(cfg); err != nil {
		log.Println(err)
	}

	// Убираем пробелы из адреса
	cfg.Addr = strings.TrimSpace(cfg.Addr)
}
