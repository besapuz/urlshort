package config

import (
	"flag"
	"net"
	"net/url"
	"os"
	"path/filepath"
)

type Config struct {
	Address         string
	BaseURL         string
	LogLevel        string
	FileStoragePath string
}

func NewConfig() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&cfg.BaseURL, "b", "", "Base URL for shortened links")
	flag.StringVar(&cfg.LogLevel, "l", "", "log level")
	flag.StringVar(&cfg.FileStoragePath, "f", "", "path to file storage")

	flag.Parse()

	if envRunAddres := os.Getenv("SERVER_ADDRESS"); envRunAddres != "" {
		cfg.Address = envRunAddres
	}
	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	}
	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		cfg.LogLevel = envLogLevel
	}
	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		cfg.FileStoragePath = envFileStoragePath
	}

	// Если BaseURL отсутствует, формируем его из Address
	if cfg.BaseURL == "" {
		cfg.BaseURL = cfg.Address
	} else {
		// Проверяем, есть ли в BaseURL порт
		parsedURL, _ := url.Parse(cfg.BaseURL)
		if parsedURL.Port() == "" {
			// Добавляем порт из Address, если он указан
			_, port, err := net.SplitHostPort(cfg.Address)
			if err == nil && port != "" {
				cfg.BaseURL += ":" + port
			}
		}
	}
	if cfg.FileStoragePath == "" {
		cfg.FileStoragePath = filepath.Join(os.TempDir(), "./urls.json")
	}
	return cfg
}
