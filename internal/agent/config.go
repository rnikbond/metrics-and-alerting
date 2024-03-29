package agent

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"metrics-and-alerting/internal/agent/services/reporter"

	"github.com/caarlos0/env"
)

type Config struct {
	Addr           string   `env:"ADDRESS"         json:"address"        `
	ReportInterval Duration `env:"REPORT_INTERVAL" json:"report_interval"`
	PollInterval   Duration `env:"POLL_INTERVAL"   json:"poll_interval"  `
	ReportType     string   `env:"REPORT_TYPE"     json:"report_type"    `
	SecretKey      string   `env:"KEY"             json:"key"            `
	CryptoKey      string   `env:"CRYPTO_KEY"      json:"crypto_key"     `
	ConfigFile     string   `env:"CONFIG"`
}

// DefaultConfig Конфигурация для сервиса агента со значениями по умолчанию
func DefaultConfig() *Config {

	return &Config{
		Addr:           ":8080",
		ReportInterval: Duration{Duration: 10 * time.Second},
		PollInterval:   Duration{Duration: 2 * time.Second},
		ReportType:     reporter.ReportAsBatchJSON,
		SecretKey:      "",
		CryptoKey:      "",
	}
}

type Duration struct {
	time.Duration
}

func (duration *Duration) UnmarshalJSON(b []byte) error {
	var unmarshalledJSON interface{}

	err := json.Unmarshal(b, &unmarshalledJSON)
	if err != nil {
		return err
	}

	switch value := unmarshalledJSON.(type) {
	case string:
		duration.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid duration: %#v", unmarshalledJSON)
	}

	return nil
}

func (cfg *Config) ParseFlags() error {

	var cryptoPath string

	flag.DurationVar(&cfg.ReportInterval.Duration, "r", cfg.ReportInterval.Duration, "report interval (duration)")
	flag.DurationVar(&cfg.PollInterval.Duration, "p", cfg.PollInterval.Duration, "poll interval (duration)")
	flag.StringVar(&cfg.SecretKey, "k", cfg.SecretKey, "string - secret key for sign metrics")
	flag.StringVar(&cryptoPath, "crypto-key", cfg.CryptoKey, "string - path to file with public crypto key")
	flag.StringVar(&cfg.ReportType, "rt", cfg.ReportType, fmt.Sprint("support types: ",
		reporter.ReportAsURL, "|", reporter.ReportAsJSON, "|", reporter.ReportAsBatchJSON, "|", reporter.ReportAsGRPC))
	flag.StringVar(&cfg.ConfigFile, "c", cfg.ConfigFile, "string - path to config in JSON format")
	addr := flag.String("a", "", "ip address: ip:port")
	flag.Parse()

	if err := cfg.ReadConfig(); err != nil {
		return err
	}

	if len(cryptoPath) == 0 {
		cryptoPath = cfg.CryptoKey
	}

	if len(cryptoPath) > 0 {

		key, err := ioutil.ReadFile(cryptoPath)
		if err != nil {
			return err
		}

		cfg.CryptoKey = string(key)
	}

	if addr == nil || *addr == "" {
		*addr = cfg.Addr
	}

	parsedAddr := strings.Split(*addr, ":")
	if len(parsedAddr) != 2 {
		return fmt.Errorf("need address in a format host:port")
	}

	if len(parsedAddr[0]) > 0 {
		if parsedAddr[0] != "localhost" {
			if ip := net.ParseIP(parsedAddr[0]); ip == nil {
				return fmt.Errorf("incorrect ip: " + parsedAddr[0])
			}
		}
	} else {
		*addr = "localhost" + *addr
	}

	if _, err := strconv.Atoi(parsedAddr[1]); err != nil {
		return fmt.Errorf("incorrect port: " + parsedAddr[1])
	}

	cfg.Addr = *addr
	return nil
}

func (cfg *Config) ReadConfig() error {

	if len(cfg.ConfigFile) == 0 {
		return nil
	}

	data, errRead := ioutil.ReadFile(cfg.ConfigFile)
	if errRead != nil {
		return errRead
	}

	return json.Unmarshal(data, cfg)
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
	builder.WriteString(fmt.Sprintf("\t REPORT_TYPE: %s\n", cfg.ReportType))
	builder.WriteString(fmt.Sprintf("\t KEY: %s\n", cfg.SecretKey))

	if len(cfg.CryptoKey) != 0 {
		builder.WriteString("\t CRYPTO_KEY: USE\n")
	}

	return builder.String()
}
