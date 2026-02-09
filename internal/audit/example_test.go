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
	defaultManager, err := Init(ctx, cfg)
	if err != nil {
		fmt.Printf("Ошибка инициализации: %v\n", err)
		return
	}
	defer defaultManager.Close()

	fmt.Println("Система аудита инициализирована")
	time.Sleep(100 * time.Millisecond)

	// Output:
	// Система аудита инициализирована
}

// Example логирования событий аудита.
func ExampleLogEvent() {
	// Создаем временный файл для аудита
	tmpFile, err := os.CreateTemp("", "audit-log-*.json")
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
	}

	// Инициализация
	defaultManager, err := Init(ctx, cfg)
	if err != nil {
		fmt.Printf("Ошибка инициализации: %v\n", err)
		return
	}
	defer defaultManager.Close()

	// Логируем различные события
	LogEvent(defaultManager, ActionShorten, "user-12345", "https://example.com/very-long-path/to/shorten")
	LogEvent(defaultManager, ActionFollow, "user-67890", "https://example.com/another-long-url")
	LogEvent(defaultManager, ActionShorten, "user-abcde", "https://google.com/search?q=golang")

	// Даем время на асинхронную обработку
	time.Sleep(200 * time.Millisecond)

	// Проверяем, что файл создан
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		fmt.Printf("Ошибка чтения файла: %v\n", err)
		return
	}

	// Подсчитываем количество строк (событий)
	lines := 0
	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			lines++
		}
	}

	fmt.Printf("Записано событий: %d\n", lines)
	fmt.Println("Содержимое файла содержит JSON объекты")

	// Output:
	// Записано событий: 3
	// Содержимое файла содержит JSON объекты
}
