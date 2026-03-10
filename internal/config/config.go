package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Config содержит все параметры конфигурации приложения.
type Config struct {
	// Address - адрес и порт для запуска HTTP сервера.
	// По умолчанию: "localhost:8080".
	Address string `json:"address"`

	// BaseURL - базовый URL для сокращенных ссылок.
	// По умолчанию: совпадает с Address.
	BaseURL string `json:"base_url"`

	// LogLevel - уровень логирования (debug, info, warn, error).
	LogLevel string `json:"log_level"`

	// FileStoragePath - путь к файлу для хранения URL.
	// По умолчанию: временный файл в системной директории.
	FileStoragePath string `json:"file_storage_path"`

	// DatabaseDSN - строка подключения к базе данных PostgreSQL.
	DatabaseDSN string `json:"database_dsn"`

	// CookieSecret - секретный ключ для подписи куки.
	CookieSecret []byte `json:"cookie_secret"`

	// AuditFile - путь к файлу для аудит-логов.
	AuditFile string `json:"audit_file"`

	// AuditURL - URL для отправки аудит-событий.
	AuditURL string `json:"audit_url"`

	// EnableHTTPS - флаг для включения HTTPS.
	EnableHTTPS bool `json:"enable_https"`

	// ConfigFile - путь к файлу конфигурации JSON
	ConfigFile string

	// TrustedSubnet - доверенная подсеть в формате CIDR
	TrustedSubnet string `json:"trusted_subnet"`
}

// NewConfig создает новую конфигурацию, читая параметры из:
// 1. Флагов командной строки
// 2. JSON файла конфигурации (если указан)
// 3. Переменных окружения
// 4. Значений по умолчанию
//
// Переменные окружения имеют наивысший приоритет.
func NewConfig() *Config {
	cfg := &Config{}

	// Определяем флаги
	flag.StringVar(&cfg.Address, "a", cfg.Address, "HTTP server address")
	flag.StringVar(&cfg.BaseURL, "b", "", "Base URL for shortened links")
	flag.StringVar(&cfg.LogLevel, "l", "", "log level")
	flag.StringVar(&cfg.FileStoragePath, "f", "", "path to file storage")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database connect")
	flag.StringVar(&cfg.AuditFile, "audit-file", "", "audit file")
	flag.StringVar(&cfg.AuditURL, "audit-url", "", "audit url")
	flag.BoolVar(&cfg.EnableHTTPS, "s", cfg.EnableHTTPS, "enable HTTPS")
	flag.StringVar(&cfg.ConfigFile, "c", "", "path to config file (JSON)")
	flag.StringVar(&cfg.TrustedSubnet, "t", "", "trusted subnet (CIDR)")

	flag.Parse()

	// Загружаем конфигурацию из JSON файла, если указан
	if cfg.ConfigFile != "" {
		if err := cfg.LoadFromJSONFile(cfg.ConfigFile); err != nil {
			panic(fmt.Sprintf("Ошибка загрузки конфигурации из файла %s: %v", cfg.ConfigFile, err))
		}
		fmt.Printf("Загружена конфигурация из файла: %s\n", cfg.ConfigFile)
	}

	// Переменные окружения имеют наивысший приоритет
	cfg.loadFromEnv()

	// Пост-обработка конфигурации
	cfg.processConfig()

	return cfg
}

// loadFromEnv загружает настройки из переменных окружения
func (c *Config) loadFromEnv() {
	if envRunAddres := os.Getenv("SERVER_ADDRESS"); envRunAddres != "" {
		c.Address = envRunAddres
	}
	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		c.BaseURL = envBaseURL
	}
	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		c.LogLevel = envLogLevel
	}
	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		c.FileStoragePath = envFileStoragePath
	}
	if envDatabaseDSN := os.Getenv("DATABASE_DSN"); envDatabaseDSN != "" {
		c.DatabaseDSN = envDatabaseDSN
	}
	if envAuditFile := os.Getenv("AUDIT_FILE"); envAuditFile != "" {
		c.AuditFile = envAuditFile
	}
	if envAuditURL := os.Getenv("AUDIT_URL"); envAuditURL != "" {
		c.AuditURL = envAuditURL
	}
	if envEnableHTTPS := os.Getenv("ENABLE_HTTPS"); envEnableHTTPS != "" {
		// Преобразуем строку в bool
		c.EnableHTTPS = envEnableHTTPS == "true" || envEnableHTTPS == "1" || envEnableHTTPS == "on"
	}
	if envTrustedSubnet := os.Getenv("TRUSTED_SUBNET"); envTrustedSubnet != "" {
		c.TrustedSubnet = envTrustedSubnet
	}
	if envCookieSecret := os.Getenv("COOKIE_SECRET"); envCookieSecret != "" {
		c.CookieSecret = []byte(envCookieSecret)
	} else {
		c.CookieSecret = []byte("test-secret-key-12345")
	}
}

// processConfig выполняет пост-обработку конфигурации
func (c *Config) processConfig() {
	// Если BaseURL отсутствует, формируем его из Address
	if c.BaseURL == "" {
		c.BaseURL = c.Address
	} else {
		// Проверяем, есть ли в BaseURL порт
		parsedURL, _ := url.Parse(c.BaseURL)
		if parsedURL.Port() == "" {
			// Добавляем порт из Address, если он указан
			_, port, err := net.SplitHostPort(c.Address)
			if err == nil && port != "" {
				c.BaseURL += ":" + port
			}
		}
	}

	// Добавляем протокол к BaseURL если его нет
	if !strings.HasPrefix(c.BaseURL, "http://") && !strings.HasPrefix(c.BaseURL, "https://") {
		if c.EnableHTTPS {
			c.BaseURL = "https://" + c.BaseURL
		} else {
			c.BaseURL = "http://" + c.BaseURL
		}
	}

	// Устанавливаем путь к файлу хранилища по умолчанию
	if c.FileStoragePath == "" {
		c.FileStoragePath = filepath.Join(os.TempDir(), "urls.json")
	}

	// Создаем директорию для файла хранилища
	dir := filepath.Dir(c.FileStoragePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(fmt.Sprintf("Не удалось создать директорию для хранилища: %v", err))
	}
}

// GetCookieSecret возвращает секретный ключ для подписи куки в виде []byte
func (c *Config) GetCookieSecret() []byte {
	return []byte(c.CookieSecret)
}

// LoadFromJSONFile загружает конфигурацию из JSON файла
func (c *Config) LoadFromJSONFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла: %w", err)
	}

	// Временная структура для загрузки
	var fileConfig Config
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	// Загружаем только те поля, которые не были установлены через флаги
	c.mergeFromFile(&fileConfig)

	return nil
}

// mergeFromFile объединяет текущую конфигурацию с конфигурацией из файла
func (c *Config) mergeFromFile(fileConfig *Config) {
	if c.Address == "localhost:8080" && fileConfig.Address != "" {
		c.Address = fileConfig.Address
	}
	if c.BaseURL == "" && fileConfig.BaseURL != "" {
		c.BaseURL = fileConfig.BaseURL
	}
	if c.LogLevel == "" && fileConfig.LogLevel != "" {
		c.LogLevel = fileConfig.LogLevel
	}
	if c.FileStoragePath == "" && fileConfig.FileStoragePath != "" {
		c.FileStoragePath = fileConfig.FileStoragePath
	}
	if c.DatabaseDSN == "" && fileConfig.DatabaseDSN != "" {
		c.DatabaseDSN = fileConfig.DatabaseDSN
	}
	if c.AuditFile == "" && fileConfig.AuditFile != "" {
		c.AuditFile = fileConfig.AuditFile
	}
	if c.AuditURL == "" && fileConfig.AuditURL != "" {
		c.AuditURL = fileConfig.AuditURL
	}
	if len(fileConfig.CookieSecret) > 0 {
		c.CookieSecret = fileConfig.CookieSecret
	}
	if c.TrustedSubnet == "" && fileConfig.TrustedSubnet != "" {
		c.TrustedSubnet = fileConfig.TrustedSubnet
	}
	// Для булевых значений просто используем значение из файла, если флаг не был явно установлен
	if !c.EnableHTTPS && fileConfig.EnableHTTPS {
		// Проверяем, был ли флаг -s установлен явно
		flagVisited := false
		flag.Visit(func(f *flag.Flag) {
			if f.Name == "s" {
				flagVisited = true
			}
		})
		if !flagVisited {
			c.EnableHTTPS = fileConfig.EnableHTTPS
		}
	}

}

// SaveToJSONFile сохраняет текущую конфигурацию в JSON файл
func (c *Config) SaveToJSONFile(filePath string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации в JSON: %w", err)
	}

	// Создаем директорию, если нужно
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("ошибка записи файла: %w", err)
	}

	return nil
}

// String возвращает строковое представление конфигурации
func (c *Config) String() string {
	return fmt.Sprintf(
		"Config{Address: %s, BaseURL: %s, LogLevel: %s, FileStoragePath: %s, DatabaseDSN: %s, EnableHTTPS: %v}",
		c.Address, c.BaseURL, c.LogLevel, c.FileStoragePath, c.DatabaseDSN, c.EnableHTTPS,
	)
}
