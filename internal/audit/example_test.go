package audit

import (
	"context"
	"fmt"
	"os"
	"time"
)

// Example инициализации системы аудита.
func ExampleInit() {
	// Создаем временный файл для аудита
	tmpFile, err := os.CreateTemp("", "audit-example-*.json")
	if err != nil {
		fmt.Printf("Ошибка создания файла: %v\n", err)
		return
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Контекст для инициализации
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Конфигурация системы аудита
	cfg := &Config{
		AuditFile: tmpFile.Name(),
		AuditURL:  "http://localhost:9090/audit",
	}

	// Инициализация
	err = Init(ctx, cfg)
	if err != nil {
		fmt.Printf("Ошибка инициализации: %v\n", err)
		return
	}

	fmt.Println("Система аудита инициализирована")
	time.Sleep(100 * time.Millisecond)

	// Output:
	// Система аудита инициализирована
}

// Example логирования событий аудита.
func ExampleLogEvent() {
	// Инициализация (в реальном приложении делается один раз при старте)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpFile, _ := os.CreateTemp("", "audit-log-*.json")
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cfg := &Config{
		AuditFile: tmpFile.Name(),
	}
	Init(ctx, cfg)

	// Логируем различные события
	LogEvent(ActionShorten, "user-12345", "https://example.com/very-long-path/to/shorten")
	LogEvent(ActionFollow, "user-67890", "https://example.com/another-long-url")
	LogEvent(ActionShorten, "user-abcde", "https://google.com/search?q=golang")

	// Даем время на асинхронную обработку
	time.Sleep(200 * time.Millisecond)

	// Проверяем, что файл создан
	content, _ := os.ReadFile(tmpFile.Name())
	lines := 0
	for _, b := range content {
		if b == '\n' {
			lines++
		}
	}

	fmt.Printf("Записано событий: %d\n", lines)
	fmt.Println("Содержимое файла содержит JSON объекты")

	// Output:
	// Записано событий: 3
	// Содержимое файла содержит JSON объекты
}
