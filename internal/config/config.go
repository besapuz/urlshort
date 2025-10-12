package config

import (
	"flag"
)

type Config struct {
	Address string
	BaseURL string
}

func NewConfig() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&cfg.BaseURL, "b", "", "Base URL for shortened links")

	flag.Parse()

	// Если BaseURL отсутствует, формируем его из Url
	if cfg.BaseURL == "" {
		cfg.BaseURL = cfg.Address
	}

	return cfg
}
