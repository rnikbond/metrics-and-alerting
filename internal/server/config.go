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
	AddrRPC       string   `env:"ADDRESS_RPC"    json:"address_rpc"    `
	StoreInterval Duration `env:"STORE_INTERVAL" json:"store_interval" `
	Restore       bool     `env:"RESTORE"        json:"restore"        `
	DatabaseDSN   string   `env:"DATABASE_DSN"   json:"database_dsn"   `
	StoreFile     string   `env:"STORE_FILE"     json:"store_file"     `
	SecretKey     string   `env:"KEY"            json:"secret_key"     `
	CryptoKey     string   `env:"CRYPTO_KEY"     json:"crypto_key"     `
	TrustedSubnet string   `env:"TRUSTED_SUBNET" json:"trusted_subnet"`
	ConfigFile    string   `env:"CONFIG"`
}

type Duration struct {
	time.Duration
}

// DefaultConfig Конфигурация для сервиса агента со значениями по умолчанию
func DefaultConfig() *Config {

	return &Config{
		Addr:          ":8080",
		AddrRPC:       ":3200",
		Restore:       true,
		DatabaseDSN:   "",
		StoreFile:     "",
		SecretKey:     "",
		CryptoKey:     "",
		StoreInterval: Duration{Duration: 10 * time.Second},
	}
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

func (cfg *Config) ParseFlags() error {

	var cryptoPath string
	var trustedSubnet string

	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "bool - restore metrics")
	flag.StringVar(&cfg.StoreFile, "f", cfg.StoreFile, "string - path to fileStorage storage")
	flag.DurationVar(&cfg.StoreInterval.Duration, "i", cfg.StoreInterval.Duration, "duration - interval store metrics")
	flag.StringVar(&cfg.SecretKey, "k", cfg.SecretKey, "string - key sign")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "string - dbstore data source name")
	flag.StringVar(&cryptoPath, "crypto-key", cfg.CryptoKey, "string - path to file with private crypto key")
	flag.StringVar(&cfg.ConfigFile, "c", cfg.ConfigFile, "string - path to config in JSON format")
	flag.StringVar(&trustedSubnet, "t", trustedSubnet, "string - CIDR")
	flag.StringVar(&cfg.AddrRPC, "rpc", cfg.AddrRPC, "string - address grpc gate")

	addr := flag.String("a", "", "string - host:port")
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

	if len(parsedAddr[0]) > 0 && parsedAddr[0] != "localhost" {
		if ip := net.ParseIP(parsedAddr[0]); ip == nil {
			return fmt.Errorf("incorrect ip: " + parsedAddr[0])
		}
	}

	if _, err := strconv.Atoi(parsedAddr[1]); err != nil {
		return fmt.Errorf("incorrect port: " + parsedAddr[1])
	}

	cfg.Addr = *addr

	if len(trustedSubnet) != 0 {
		trustedSubnet = strings.ReplaceAll(trustedSubnet, " ", "")
		ipSet := strings.Split(trustedSubnet, ",")
		for _, ip := range ipSet {
			if netIP := net.ParseIP(ip); netIP == nil {
				return fmt.Errorf("incorrect subnet ip: " + ip)
			}
		}

		cfg.TrustedSubnet = trustedSubnet
	}

	return nil
}

func (cfg Config) String() string {

	builder := strings.Builder{}

	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("\t ADDRESS: %s\n", cfg.Addr))
	builder.WriteString(fmt.Sprintf("\t ADDRESS RPC: %s\n", cfg.AddrRPC))
	builder.WriteString(fmt.Sprintf("\t STORE_INTERVAL: %s\n", cfg.StoreInterval.String()))
	builder.WriteString(fmt.Sprintf("\t RESTORE: %v\n", cfg.Restore))
	builder.WriteString(fmt.Sprintf("\t DATABASE_DSN: %s\n", cfg.DatabaseDSN))
	builder.WriteString(fmt.Sprintf("\t STORE_FILE: %s\n", cfg.StoreFile))
	builder.WriteString(fmt.Sprintf("\t KEY: %s\n", cfg.SecretKey))
	builder.WriteString(fmt.Sprintf("\t TRUSTED_SUBNET: %s\n", cfg.TrustedSubnet))

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
