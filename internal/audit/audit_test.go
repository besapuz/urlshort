package audit

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestAuditSystem(t *testing.T) {
	// Создаем временный файл для тестирования
	tmpFile, err := os.CreateTemp("", "audit-test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := &Config{
		AuditFile: tmpFile.Name(),
		AuditURL:  "", // Не тестируем HTTP в юнит-тестах
	}

	defaultManager, err := Init(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to init audit: %v", err)
	}

	// Логируем событие
	LogEvent(defaultManager, ActionShorten, "test-user-123", "https://example.com/long-url")

	// Даем время на асинхронную обработку
	time.Sleep(100 * time.Millisecond)

	// Проверяем, что файл создан и содержит данные
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read audit file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Audit file is empty")
	}

	// Проверяем структуру JSON
	var event Event
	err = json.Unmarshal(data[:len(data)-1], &event) // Убираем последний \n
	if err != nil {
		t.Fatalf("Failed to unmarshal audit event: %v", err)
	}

	if event.Action != ActionShorten {
		t.Errorf("Expected action %s, got %s", ActionShorten, event.Action)
	}

	if event.UserID != "test-user-123" {
		t.Errorf("Expected user ID test-user-123, got %s", event.UserID)
	}
}
