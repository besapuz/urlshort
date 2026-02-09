package audit

import (
	"context"
	"os"
	"testing"
)

func BenchmarkAuditLogEvent(b *testing.B) {
	// Создаем временный файл для тестирования
	tmpFile, err := os.CreateTemp("", "audit-bench-*.json")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &Config{
		AuditFile: tmpFile.Name(),
	}
	defaultManager, err := Init(ctx, cfg)
	if err != nil {
		b.Fatal(err)
	}
	defer defaultManager.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogEvent(defaultManager, ActionShorten, "test-user-123", "https://example.com/long-url")
	}
}

func BenchmarkNewEvent(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewEvent(ActionShorten, "test-user-123", "https://example.com/long-url")
	}
}
