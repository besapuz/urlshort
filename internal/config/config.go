// Package config предоставляет конфигурацию для URL shortener сервиса.
// Поддерживает чтение конфигурации из флагов командной строки и переменных окружения.
package config

import (
	"flag"
	"net"
	"net/url"
	"os"
	"path/filepath"
)

// Config содержит все параметры конфигурации приложения.
type Config struct {
	// Address - адрес и порт для запуска HTTP сервера.
	// По умолчанию: "localhost:8080".
	Address string

	// BaseURL - базовый URL для сокращенных ссылок.
	// По умолчанию: совпадает с Address.
	BaseURL string

	// LogLevel - уровень логирования (debug, info, warn, error).
	LogLevel string

	// FileStoragePath - путь к файлу для хранения URL.
	// По умолчанию: временный файл в системной директории.
	FileStoragePath string

	// DatabaseDSN - строка подключения к базе данных PostgreSQL.
	DatabaseDSN string

	// CookieSecret - секретный ключ для подписи куки.
	CookieSecret []byte

	// AuditFile - путь к файлу для аудит-логов.
	AuditFile string

	// AuditURL - URL для отправки аудит-событий.
	AuditURL string
}

// NewConfig создает новую конфигурацию, читая параметры из:
// 1. Флагов командной строки
// 2. Переменных окружения
// 3. Значений по умолчанию
//
// Переменные окружения имеют приоритет над флагами командной строки.
func NewConfig() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&cfg.BaseURL, "b", "", "Base URL for shortened links")
	flag.StringVar(&cfg.LogLevel, "l", "", "log level")
	flag.StringVar(&cfg.FileStoragePath, "f", "", "path to file storage")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database connect")
	flag.StringVar(&cfg.AuditFile, "audit-file", "", "audit file")
	flag.StringVar(&cfg.AuditURL, "audit-url", "", "audit url")

	flag.Parse()

	// Чтение из переменных окружения
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
	if envDatabaseDSN := os.Getenv("DATABASE_DSN"); envDatabaseDSN != "" {
		cfg.DatabaseDSN = envDatabaseDSN
	}
	if envAuditFile := os.Getenv("AUDIT_FILE"); envAuditFile != "" {
		cfg.AuditFile = envAuditFile
	}
	if envAuditURL := os.Getenv("AUDIT_URL"); envAuditURL != "" {
		cfg.AuditURL = envAuditURL
	}
	if envCookieSecret := os.Getenv("COOKIE_SECRET"); envCookieSecret != "" {
		cfg.CookieSecret = []byte(envCookieSecret)
	} else {
		cfg.CookieSecret = []byte("test-secret-key-12345")
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
		cfg.FileStoragePath = filepath.Join(os.TempDir(), "urls.json")
	}

	// Создаем директорию для файла хранилища
	dir := filepath.Dir(cfg.FileStoragePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(err)
	}
	return cfg
}

// GetCookieSecret возвращает секретный ключ для подписи куки.
func (c *Config) GetCookieSecret() []byte {
	return c.CookieSecret
}
