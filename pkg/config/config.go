package config

import (
	"log"
	"os/user"
	"runtime"
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

func (cfg *Config) Read() {

	// Инициализация значений по умолчанию
	cfg.Addr = "127.0.0.1:8080"
	cfg.PollInterval = 2 * time.Second
	cfg.ReportInterval = 10 * time.Second
	cfg.StoreInterval = 300 * time.Second
	cfg.Restore = true

	// Инициализация значений по умолчанию
	// - Путь к файлу для сохранения под Windows и не Windows
	if strings.Contains(runtime.GOOS, "windows") {
		if usr, err := user.Current(); err == nil {
			cfg.StoreFile = usr.HomeDir + "\\devops-metrics-db.json"
		} else {
			log.Printf("error getting the path to the user directory: %s\n", err.Error())
		}
	} else {
		cfg.StoreFile = "/tmp/devops-metrics-db.json"
	}

	// Чтение переменных среды
	if err := env.Parse(cfg); err != nil {
		log.Println(err)
	}

	// Убираем сохранение в файл, если путь не указан
	if len(cfg.StoreFile) < 1 {
		cfg.Restore = false
	}

	// Убираем пробелы из адреса
	cfg.Addr = strings.TrimSpace(cfg.Addr)
}
