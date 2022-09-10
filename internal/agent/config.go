package agent

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"metrics-and-alerting/internal/agent/services/reporter"

	"github.com/caarlos0/env"
)

type Config struct {
	Addr           string        `env:"ADDRESS"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
	PollInterval   time.Duration `env:"POLL_INTERVAL"`
	ReportURL      string        `env:"REPORT_TYPE"`
	SecretKey      string        `env:"KEY"`
}

// DefaultConfig Конфигурация для сервиса агента со значениями по умолчанию
func DefaultConfig() *Config {

	return &Config{
		Addr:           ":8080",
		ReportInterval: 10 * time.Second,
		PollInterval:   2 * time.Second,
		ReportURL:      reporter.ReportAsURL,
		SecretKey:      "",
	}
}

func (cfg *Config) ParseFlags() error {

	flag.DurationVar(&cfg.ReportInterval, "r", cfg.ReportInterval, "report interval (duration)")
	flag.DurationVar(&cfg.PollInterval, "p", cfg.PollInterval, "poll interval (duration)")
	flag.StringVar(&cfg.SecretKey, "k", cfg.SecretKey, "string - secret key for sign metrics")
	flag.StringVar(&cfg.ReportURL, "rt", cfg.ReportURL, fmt.Sprint("support types: ",
		reporter.ReportAsURL, "|", reporter.ReportAsJSON, "|", reporter.ReportAsJsonBatch))
	addr := flag.String("a", cfg.Addr, "ip address: ip:port")
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

// ReadEnvironment Получение параметров конфигурации из переменных окружения
func (cfg *Config) ReadEnvironment() {

	// Чтение переменных среды
	if err := env.Parse(cfg); err != nil {
		log.Println(err)
	}

	// Удаление пробелов из адреса
	cfg.Addr = strings.TrimSpace(cfg.Addr)
}

func (cfg Config) String() string {

	builder := strings.Builder{}

	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("\t ADDRESS: %s\n", cfg.Addr))
	builder.WriteString(fmt.Sprintf("\t REPORT_INTERVAL: %s\n", cfg.ReportInterval.String()))
	builder.WriteString(fmt.Sprintf("\t POLL_INTERVAL: %s\n", cfg.PollInterval.String()))
	builder.WriteString(fmt.Sprintf("\t REPORT_TYPE: %s\n", cfg.ReportURL))
	builder.WriteString(fmt.Sprintf("\t KEY: %s\n", cfg.SecretKey))

	return builder.String()
}
