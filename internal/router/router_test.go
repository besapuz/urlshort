package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRedirectHandler - тестирование функции main/redirectHandler.
func TestRedirectHandler(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		setup          func()
		expectedStatus int
		expectedHeader string
	}{
		{
			name:           "Пустой идентификатор",
			path:           "/",
			setup:          func() { urlMap = map[string]string{} },
			expectedStatus: http.StatusBadRequest,
			expectedHeader: "",
		},
		{
			name:           "Валидный идентификатор",
			path:           "/example",
			setup:          func() { urlMap = map[string]string{"example": "https://example.com"} },
			expectedStatus: http.StatusTemporaryRedirect,
			expectedHeader: "https://example.com",
		},
		{
			name:           "Невалидный идентификатор",
			path:           "/invalid",
			setup:          func() { urlMap = map[string]string{"valid": "https://valid.com"} },
			expectedStatus: http.StatusBadRequest,
			expectedHeader: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			shortener := &URLShortener{}
			req, err := http.NewRequest("GET", tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			rr := httptest.NewRecorder()
			shortener.RedirectHandler(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler вернул неправильный статус: получил %v, ожидал %v", status, tt.expectedStatus)
			}

			location := rr.Header().Get("Location")
			if location != tt.expectedHeader {
				t.Errorf("Заголовок Location не совпадает: получил %q, ожидал %q", location, tt.expectedHeader)
			}
		})
	}
}

// TestShortenHandler - тестирование функции main/shortenHandler.

func TestShortenHandler(t *testing.T) {

	tests := []struct {
		name           string
		contentType    string
		body           string
		expectedStatus int
	}{
		{
			name:           "Invalid Content-Type",
			contentType:    "application/json",
			body:           "https://example.com",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty body",
			contentType:    "text/plain",
			body:           "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid URL format",
			contentType:    "text/plain",
			body:           "example.com",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Valid request",
			contentType:    "text/plain",
			body:           "https://example.com",
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortener := &URLShortener{}
			// Сброс глобального состояния
			urlMap = make(map[string]string)

			// Создание запроса
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)

			// Создание ответа
			rr := httptest.NewRecorder()

			// Вызов обработчика
			handler := shortener.ShortenHandler("http://localhost:8080")
			handler(rr, req)

			// Проверка статуса
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

// TestShortenJSONHandler - тестирование функции shortenJSONHandler.
func TestShortenJSONHandler(t *testing.T) {

	tests := []struct {
		name           string
		contentType    string
		body           string
		filePath       string
		expectedStatus int
	}{
		{
			name:           "Valid Content-Type",
			contentType:    "application/json",
			body:           `{"url": "https://example.com"}`,
			filePath:       "testdata/urls.json",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Empty body",
			contentType:    "text/plain",
			body:           "",
			filePath:       "testdata/urls.json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid URL format",
			contentType:    "text/plain",
			body:           `{"url": "https://example.com"}`,
			filePath:       "testdata/urls.json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid request",
			contentType:    "text/plain",
			body:           `{"url": "https://example.com"}`,
			filePath:       "testdata/urls.json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortener := &URLShortener{}
			// Сброс глобального состояния
			urlMap = make(map[string]string)

			// Создание запроса
			req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)

			// Создание ответа
			rr := httptest.NewRecorder()

			// Вызов обработчика
			handler := shortener.ShortenJSONHandler(tt.body, tt.filePath)
			handler(rr, req)

			// Проверка статуса
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}
