package config

import (
	"flag"
	"os"
)

type Config struct {
	Address string
	BaseURL string
}

func NewConfig() *Config {
	cfg := &Config{}
	// берем алрес из переменной окружения и если env отсутствует, то берем значение флага -a
	// если и его нет то назначаем дефолт localhost:8080
	if envAdDress := os.Getenv("SERVER_ADDRESS"); envAdDress != "" {
		cfg.Address = envAdDress
	} else {
		flag.StringVar(&cfg.Address, "a", "localhost:8080", "HTTP server address")
	}

	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	} else {
		flag.StringVar(&cfg.BaseURL, "b", "", "Base URL for shortened links")
	}
	flag.Parse()

	// Если BaseURL отсутствует, формируем его из Url
	if cfg.BaseURL == "" {
		cfg.BaseURL = cfg.Address
	}

	return cfg
}
