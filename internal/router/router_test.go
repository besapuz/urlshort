package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/besapuz/urlshort/internal/audit"
	"github.com/stretchr/testify/assert"
)

var defManager *audit.Manager

// TestRedirectHandler - тестирование функции main/redirectHandler.
func TestRedirectHandler(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		setup          func(*URLShortener)
		expectedStatus int
		expectedHeader string
	}{
		{
			name: "Пустой идентификатор",
			path: "/",
			setup: func(s *URLShortener) {
				s.MemoryStorage.urlMap = make(map[string]string)
			},
			expectedStatus: http.StatusBadRequest,
			expectedHeader: "",
		},
		{
			name: "Валидный идентификатор",
			path: "/example",
			setup: func(s *URLShortener) {
				s.MemoryStorage.urlMap = map[string]string{"example": "https://example.com"}
			},
			expectedStatus: http.StatusTemporaryRedirect,
			expectedHeader: "https://example.com",
		},
		{
			name: "Невалидный идентификатор",
			path: "/invalid",
			setup: func(s *URLShortener) {
				s.MemoryStorage.urlMap = map[string]string{"valid": "https://valid.com"}
			},
			expectedStatus: http.StatusBadRequest,
			expectedHeader: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ИСПОЛЬЗУЕМ КОНСТРУКТОР
			shortener := NewURLShortener("http://localhost:8080", []byte("test-secret"))
			tt.setup(shortener)

			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()

			handler := shortener.RedirectHandler(defManager)
			handler(rr, req)

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
			// ИСПОЛЬЗУЕМ КОНСТРУКТОР
			shortener := NewURLShortener("http://localhost:8080", []byte("test-secret"))

			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			rr := httptest.NewRecorder()

			handler := shortener.ShortenHandler(defManager, "http://localhost:8080")
			handler(rr, req)

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
			filePath:       "",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Invalid Content-Type",
			contentType:    "text/plain",
			body:           `{"url": "https://example.com"}`,
			filePath:       "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty body",
			contentType:    "application/json",
			body:           "",
			filePath:       "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			contentType:    "application/json",
			body:           `{invalid json}`,
			filePath:       "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid URL format",
			contentType:    "application/json",
			body:           `{"url": "example.com"}`,
			filePath:       "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ИСПОЛЬЗУЕМ КОНСТРУКТОР
			shortener := NewURLShortener("http://localhost:8080", []byte("test-secret"))

			req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			rr := httptest.NewRecorder()

			handler := shortener.ShortenJSONHandler(defManager, "http://localhost:8080", tt.filePath)
			handler(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}
