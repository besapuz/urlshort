// main.go
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/besapuz/urlshort/internal/config"
	"github.com/besapuz/urlshort/internal/handler"
	"github.com/besapuz/urlshort/internal/logger"
	"github.com/besapuz/urlshort/internal/router"
	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := config.NewConfig()
	r := chi.NewRouter()
	router.SetStorageFile(cfg.FileStoragePath)

	if cfg.DatabaseDSN != "" {
		if err := router.InitDBStorage(cfg.DatabaseDSN); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка инициализации базы данных: %v\n", err)
			os.Exit(1)
		}
	}

	if err := router.LoadFromFile(cfg.FileStoragePath); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка загрузки файла: %v\n", err)
		os.Exit(1)
	}
	r.Use(handler.GzipMiddleware)

	// Используем старую сигнатуру, но внутри она будет сохранять в файл
	r.Post("/", router.ShortenHandler(cfg.BaseURL))
	r.Post("/api/shorten", router.ShortenJSONHandler(cfg.BaseURL, cfg.FileStoragePath))

	r.Get("/{id}", router.RedirectHandler)
	r.Get("/ping", router.PingHandler)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Shortener service is running at %s", cfg.BaseURL)
	})
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		panic(err)
	}

	fmt.Printf("Server started on http://%s\n", cfg.Address)
	err := http.ListenAndServe(cfg.Address, logger.RequestLogger(r))

	if err != nil {
		panic(err)
	}
}
