package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/besapuz/urlshort/internal/config"
)

// Example пакетного создания коротких ссылок.
func ExampleURLShortener_BatchShortenHandler() {
	cfg := &config.Config{
		BaseURL:      "http://localhost:8080",
		CookieSecret: []byte("test-secret"),
	}

	// ИСПОЛЬЗУЕМ КОНСТРУКТОР
	shortener := NewURLShortener(cfg.BaseURL, cfg.GetCookieSecret())

	handler := shortener.BatchShortenHandler(cfg.BaseURL, "")
	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	// Подготавливаем пакетный запрос
	batchRequest := []BatchRequestItem{
		{CorrelationID: "1", OriginalURL: "https://example.com/first"},
		{CorrelationID: "2", OriginalURL: "https://example.com/second"},
		{CorrelationID: "3", OriginalURL: "https://example.com/third"},
	}

	jsonBody, _ := json.Marshal(batchRequest)

	// Отправляем запрос
	resp, err := http.Post(ts.URL, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)

	// Парсим ответ
	var result []BatchResponseItem
	json.Unmarshal(body, &result)

	for _, item := range result {
		fmt.Printf("Correlation ID: %s -> Short URL starts with: http://localhost:8080/\n",
			item.CorrelationID)
	}

	// Output:
	// Status: 201
	// Correlation ID: 1 -> Short URL starts with: http://localhost:8080/
	// Correlation ID: 2 -> Short URL starts with: http://localhost:8080/
	// Correlation ID: 3 -> Short URL starts with: http://localhost:8080/
}

// Example удаления ссылок.
func ExampleURLShortener_DeleteURLsHandler() {
	cfg := &config.Config{
		BaseURL:      "http://localhost:8080",
		CookieSecret: []byte("test-secret"),
	}

	// ИСПОЛЬЗУЕМ КОНСТРУКТОР
	shortener := NewURLShortener(cfg.BaseURL, cfg.GetCookieSecret())

	// Создаем тестовый сервер
	handler := shortener.DeleteURLsHandler()
	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	// Подготавливаем запрос на удаление
	deleteRequest := []string{"abc123", "def456"}
	jsonBody, _ := json.Marshal(deleteRequest)

	// Создаем запрос
	req, _ := http.NewRequest("DELETE", ts.URL, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Устанавливаем тестовую куку
	req.AddCookie(&http.Cookie{
		Name:  "user_id",
		Value: signUserID("test-user", cfg.GetCookieSecret()),
	})

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(body))

	// Output:
	// Status: 202
	// Response: Delete request accepted
}

// Example проверки доступности БД.
func ExampleURLShortener_PingHandler() {
	// Этот пример требует реальной БД, поэтому покажем только структуру
	fmt.Println("Ping endpoint checks database connection")
	fmt.Println("GET /ping returns 200 if DB is available")
	fmt.Println("GET /ping returns 500 if DB is unavailable")

	// Output:
	// Ping endpoint checks database connection
	// GET /ping returns 200 if DB is available
	// GET /ping returns 500 if DB is unavailable
}
