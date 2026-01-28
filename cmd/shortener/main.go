package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/besapuz/urlshort/internal/audit"
	"github.com/besapuz/urlshort/internal/config"
	"github.com/besapuz/urlshort/internal/handler"
	"github.com/besapuz/urlshort/internal/logger"
	"github.com/besapuz/urlshort/internal/router"
	"github.com/go-chi/chi/v5"
)

func main() {
	// Запускаем pprof на отдельном порту
	go func() {
		fmt.Println("Starting pprof server on :6060")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			fmt.Printf("pprof server error: %v\n", err)
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Обработка сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range sigChan {
			switch sig {
			case syscall.SIGUSR1:
				fmt.Println("Received SIGUSR1, dumping memory stats...")
				dumpMemoryStats()
			case syscall.SIGINT, syscall.SIGTERM:
				fmt.Printf("Received signal %v, shutting down...\n", sig)
				cancel()
			default:
				fmt.Printf("Received unexpected signal %v\n", sig)
			}
		}
	}()

	cfg := config.NewConfig()
	r := chi.NewRouter()

	// Инициализация аудита
	auditCfg := &audit.Config{
		AuditFile: cfg.AuditFile,
		AuditURL:  cfg.AuditURL,
	}
	if err := audit.Init(ctx, auditCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка инициализации аудита: %v\n", err)
		os.Exit(1)
	}
	shortener := &router.URLShortener{
		BaseURL:         cfg.BaseURL,
		FileStoragePath: cfg.FileStoragePath,
		CookieSecret:    cfg.GetCookieSecret(),
	}

	if cfg.DatabaseDSN == "" {
		router.SetStorageFile(cfg.FileStoragePath)
		if err := router.LoadFromFile(cfg.FileStoragePath); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка загрузки файла: %v\n", err)
			os.Exit(1)
		}
	}
	if cfg.DatabaseDSN != "" {
		if err := shortener.InitDBStorage(cfg.DatabaseDSN); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка инициализации базы данных: %v\n", err)
			os.Exit(1)
		}
	}

	r.Use(handler.GzipMiddleware)

	// Используем старую сигнатуру, но внутри она будет сохранять в файл
	r.Post("/", shortener.ShortenHandler(cfg.BaseURL))
	r.Post("/api/shorten", shortener.ShortenJSONHandler(cfg.BaseURL, cfg.FileStoragePath))
	r.Post("/api/shorten/batch", shortener.BatchShortenHandler(cfg.BaseURL, cfg.FileStoragePath))

	r.Get("/{id}", shortener.RedirectHandler)
	r.Get("/ping", shortener.PingHandler)
	r.Get("/api/user/urls", shortener.GetUserURLsHandler(cfg.BaseURL))

	r.Delete("/api/user/urls", shortener.DeleteURLsHandler())

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Shortener service is running at %s", cfg.BaseURL)
	})
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		panic(err)
	}

	server := &http.Server{
		Addr:    cfg.Address,
		Handler: logger.RequestLogger(r),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	<-ctx.Done()
	fmt.Println("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("Server shutdown error: %v\n", err)
	}
}

func dumpMemoryStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("=== Memory Stats ===\n")
	fmt.Printf("Alloc = %v MiB\n", bToMb(m.Alloc))
	fmt.Printf("TotalAlloc = %v MiB\n", bToMb(m.TotalAlloc))
	fmt.Printf("Sys = %v MiB\n", bToMb(m.Sys))
	fmt.Printf("HeapAlloc = %v MiB\n", bToMb(m.HeapAlloc))
	fmt.Printf("HeapSys = %v MiB\n", bToMb(m.HeapSys))
	fmt.Printf("HeapIdle = %v MiB\n", bToMb(m.HeapIdle))
	fmt.Printf("HeapInuse = %v MiB\n", bToMb(m.HeapInuse))
	fmt.Printf("NumGC = %v\n", m.NumGC)
	fmt.Printf("GCCPUFraction = %v\n", m.GCCPUFraction)
	fmt.Printf("=== End Memory Stats ===\n")
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
