// internal/logger/logger.go
package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Log будет доступен всему коду как синглтон.
var Log *zap.Logger = zap.NewNop()

// Initialize инициализирует синглтон логера с необходимым уровнем логирования.
func Initialize(level string) error {
	// преобразуем текстовый уровень логирования в zap.AtomicLevel
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}
	// создаём новую конфигурацию логера
	cfg := zap.NewProductionConfig()
	// устанавливаем уровень
	cfg.Level = lvl
	// создаём логер на основе конфигурации
	zl, err := cfg.Build()
	if err != nil {
		return err
	}
	// устанавливаем синглтон
	Log = zl
	return nil
}

// LoggingResponseWriter - оборачивает оригинальный ResponseWriter для отслеживания статуса и размера ответа.
type LoggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	bodyLength int
}

func NewLoggingResponseWriter(w http.ResponseWriter) *LoggingResponseWriter {
	return &LoggingResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// GetStatusCode - возвращает код статуса ответа.
func (l *LoggingResponseWriter) GetStatusCode() int {
	return l.statusCode
}

// GetBodyLength - возвращает размер тела ответа.
func (l *LoggingResponseWriter) GetBodyLength() int {
	return l.bodyLength
}

// RequestLogger — middleware для логирования всех HTTP-запросов и ответов.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lrw := NewLoggingResponseWriter(w)
		next.ServeHTTP(lrw, r)

		duration := time.Since(start).Milliseconds()

		Log.Info("HTTP request handled",
			zap.String("method", r.Method),
			zap.String("url", r.RequestURI),
			zap.Int("status", lrw.GetStatusCode()),
			zap.Int("body_length", lrw.GetBodyLength()),
			zap.Int64("duration_ms", duration),
		)
	})
}
