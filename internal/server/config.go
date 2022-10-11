package server

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

	"github.com/caarlos0/env"
)

type Config struct {
	Addr          string   `env:"ADDRESS"        json:"address"        `
	StoreInterval Duration `env:"STORE_INTERVAL" json:"store_interval" `
	Restore       bool     `env:"RESTORE"        json:"restore"        `
	DatabaseDSN   string   `env:"DATABASE_DSN"   json:"database_dsn"   `
	StoreFile     string   `env:"STORE_FILE"     json:"store_file"     `
	SecretKey     string   `env:"KEY"            json:"secret_key"     `
	CryptoKey     string   `env:"CRYPTO_KEY"     json:"crypto_key"     `
	ConfigFile    string   `env:"CONFIG"`
}

type Duration struct {
	time.Duration
}

// DefaultConfig Конфигурация для сервиса агента со значениями по умолчанию
func DefaultConfig() *Config {

	return &Config{
		Addr:          ":8080",
		Restore:       true,
		DatabaseDSN:   "",
		StoreFile:     "",
		SecretKey:     "",
		CryptoKey:     "",
		StoreInterval: Duration{Duration: 10 * time.Second},
	}
}

func (duration *Duration) UnmarshalJSON(b []byte) error {
	var unmarshalledJson interface{}

	err := json.Unmarshal(b, &unmarshalledJson)
	if err != nil {
		return err
	}

	switch value := unmarshalledJson.(type) {
	case float64:
		duration.Duration = time.Duration(value)
	case string:
		duration.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid duration: %#v", unmarshalledJson)
	}

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

	cfgDef := DefaultConfig()
	var cfgConf Config

	if errJSON := json.Unmarshal(data, &cfgConf); errJSON != nil {
		return errJSON
	}

	if cfg.Addr == cfgDef.Addr && cfg.Addr != cfgConf.Addr {
		if len(cfgConf.Addr) != 0 {
			cfg.Addr = cfgConf.Addr
		}
	}

	if cfg.StoreInterval == cfgDef.StoreInterval &&
		cfg.StoreInterval != cfgConf.StoreInterval {
		cfg.StoreInterval = cfgConf.StoreInterval
	}

	if cfg.Restore == cfgDef.Restore && cfg.Restore != cfgConf.Restore {
		cfg.Restore = cfgConf.Restore
	}

	if cfg.DatabaseDSN == cfgDef.DatabaseDSN && cfg.DatabaseDSN != cfgConf.DatabaseDSN {
		if len(cfgConf.DatabaseDSN) != 0 {
			cfg.DatabaseDSN = cfgConf.DatabaseDSN
		}
	}

	if cfg.StoreFile == cfgDef.StoreFile && cfg.StoreFile != cfgConf.StoreFile {
		if len(cfgConf.StoreFile) != 0 {
			cfg.StoreFile = cfgConf.StoreFile
		}
	}

	if cfg.SecretKey == cfgDef.SecretKey && cfg.SecretKey != cfgConf.SecretKey {
		if len(cfgConf.SecretKey) != 0 {
			cfg.SecretKey = cfgConf.SecretKey
		}
	}

	if cfg.CryptoKey == cfgDef.CryptoKey && cfg.CryptoKey != cfgConf.CryptoKey {
		if len(cfgConf.CryptoKey) != 0 {
			cfg.CryptoKey = cfgConf.CryptoKey
		}
	}

	return nil
}

func (cfg *Config) ParseFlags() error {

	var cryptoPath string

	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "bool - restore metrics")
	flag.StringVar(&cfg.StoreFile, "f", cfg.StoreFile, "string - path to fileStorage storage")
	flag.DurationVar(&cfg.StoreInterval.Duration, "i", cfg.StoreInterval.Duration, "duration - interval store metrics")
	flag.StringVar(&cfg.SecretKey, "k", cfg.SecretKey, "string - key sign")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "string - dbstore data source name")
	flag.StringVar(&cryptoPath, "crypto-key", cfg.CryptoKey, "string - path to file with private crypto key")
	flag.StringVar(&cfg.ConfigFile, "c", cfg.ConfigFile, "string - path to config in JSON format")

	addr := flag.String("a", cfg.Addr, "string - host:port")
	flag.Parse()

	if len(cryptoPath) > 0 {

		key, err := ioutil.ReadFile(cryptoPath)
		if err != nil {
			return err
		}

		cfg.CryptoKey = string(key)
	}

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

	if err := cfg.ReadConfig(); err != nil {
		return err
	}

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

	if len(cfg.CryptoKey) != 0 {
		builder.WriteString("\t CRYPTO_KEY: USE\n")
	}

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
