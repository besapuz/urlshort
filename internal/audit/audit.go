package audit

import (
	"context"
	"log"
)

var (
	defaultManager *Manager
)

// Config - конфигурация аудита
type Config struct {
	AuditFile string
	AuditURL  string
}

// Init инициализирует систему аудита
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

// LogEvent логирует событие аудита
func LogEvent(action Action, userID, url string) {
	if defaultManager == nil {
		return
	}

	event := NewEvent(action, userID, url)
	defaultManager.NotifyAll(event)
}
