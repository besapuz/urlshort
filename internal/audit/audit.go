// Package audit предоставляет систему аудита для логирования событий в URL shortener сервисе.
// Система поддерживает несколько способов логирования: файловое хранилище и HTTP-отправку.
package audit

import (
	"context"
	"log"
)

var (
	// defaultManager - глобальный менеджер аудита, используемый пакетом.
	defaultManager *Manager
)

// Config содержит конфигурацию системы аудита.
type Config struct {
	// AuditFile - путь к файлу для записи аудит-логов.
	// Если пустая строка, файловое логирование отключено.
	AuditFile string

	// AuditURL - URL для отправки аудит-событий по HTTP.
	// Если пустая строка, HTTP-логирование отключено.
	AuditURL string
}

// Init инициализирует систему аудита с заданной конфигурацией.
// Функция должна быть вызвана один раз при запуске приложения.
// Принимает контекст для graceful shutdown.
// Возвращает ошибку в случае проблем с инициализацией.
func Init(ctx context.Context, cfg *Config) error {
	defaultManager = NewManager()

	// Инициализируем файловый писатель, если указан путь
	if cfg.AuditFile != "" {
		fw, err := NewFileWriter(cfg.AuditFile)
		if err != nil {
			return err
		}
		defaultManager.Register(fw)
		log.Printf("File audit enabled: %s", cfg.AuditFile)
	}

	// Инициализируем HTTP отправитель, если указан URL
	if cfg.AuditURL != "" {
		hs := NewHTTPSender(cfg.AuditURL)
		defaultManager.Register(hs)
		log.Printf("HTTP audit enabled: %s", cfg.AuditURL)
	}

	// Обработка завершения приложения
	go func() {
		<-ctx.Done()
		if err := defaultManager.Close(); err != nil {
			log.Printf("Error closing audit manager: %v", err)
		}
	}()

	return nil
}

// LogEvent логирует событие аудита в систему.
// Функция потокобезопасна и может вызываться из нескольких горутин.
// Если система аудита не инициализирована, событие игнорируется.
func LogEvent(action Action, userID, url string) {
	if defaultManager == nil {
		return
	}

	event := NewEvent(action, userID, url)
	defaultManager.NotifyAll(event)
}
