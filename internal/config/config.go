package config

import (
	"flag"
	"fmt"
)

type Config struct {
	Url     string
	BaseURL string
}

func NewConfig() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.Url, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&cfg.BaseURL, "b", "", "Base URL for shortened links")

	flag.Parse()

	// Если BaseURL отсутствует, формируем его из Url
	if cfg.BaseURL == "" {
		cfg.BaseURL = fmt.Sprintf("%s", cfg.Url)
	}

	return cfg
}
