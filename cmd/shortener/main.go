package main

import (
	"context"
	"embed"
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

// Глобальные переменные для версии сборки
var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

//go:embed templates/*.html
//go:embed static/*
//go:embed configs/*.json
var embeddedFiles embed.FS

func main() {
	// Вывод информации о сборке
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
	fmt.Println()

	// Пример использования встроенных файлов
	printEmbeddedFiles()

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
	defaultManager, err := audit.Init(ctx, auditCfg)
	if err != nil {
		panic(fmt.Sprintf("Ошибка инициализации аудита: %v", err))
	}
	var shortener *router.URLShortener
	if cfg.DatabaseDSN != "" {
		// Используем конструктор с БД
		shortener, err = router.NewURLShortenerWithDB(cfg.BaseURL, cfg.GetCookieSecret(), cfg.DatabaseDSN)
		if err != nil {
			panic(fmt.Sprintf("Ошибка инициализации базы данных: %v\n", err))
		}
	} else {
		// Используем конструктор без БД
		shortener = router.NewURLShortener(cfg.BaseURL, cfg.GetCookieSecret())
		shortener.FileStoragePath = cfg.FileStoragePath
		shortener.MemoryStorage.SetStorageFile(cfg.FileStoragePath)
		if err := shortener.MemoryStorage.LoadFromFile(cfg.FileStoragePath); err != nil {
			panic(fmt.Sprintf("Ошибка загрузки файла: %v\n", err))
		}
	}

	if cfg.DatabaseDSN == "" {
		shortener.MemoryStorage.SetStorageFile(cfg.FileStoragePath)
		if err := shortener.MemoryStorage.LoadFromFile(cfg.FileStoragePath); err != nil {
			panic(fmt.Sprintf("Ошибка загрузки файла: %v\n", err))
		}
	}
	if cfg.DatabaseDSN != "" {
		if err := shortener.InitDBStorage(cfg.DatabaseDSN); err != nil {
			panic(fmt.Sprintf("Ошибка инициализации базы данных: %v\n", err))
		}
	}

	r.Use(handler.GzipMiddleware)

	// Используем старую сигнатуру, но внутри она будет сохранять в файл
	r.Post("/", shortener.ShortenHandler(defaultManager, cfg.BaseURL))
	r.Post("/api/shorten", shortener.ShortenJSONHandler(defaultManager, cfg.BaseURL, cfg.FileStoragePath))
	r.Post("/api/shorten/batch", shortener.BatchShortenHandler(cfg.BaseURL, cfg.FileStoragePath))

	r.Get("/{id}", shortener.RedirectHandler(defaultManager))
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

	// Запуск сервера
	go func() {
		// Проверяем, нужно ли включить HTTPS
		if cfg.EnableHTTPS == "true" || cfg.EnableHTTPS == "1" || cfg.EnableHTTPS == "on" {
			fmt.Printf("Starting HTTPS server on %s\n", cfg.Address)

			// Используем самоподписанные сертификаты для разработки
			certFile := "server.crt"
			keyFile := "server.key"

			// Проверяем существование файлов сертификатов
			if _, statErr := os.Stat(certFile); os.IsNotExist(statErr) {
				fmt.Printf("Warning: Certificate file %s not found, please generate certificates manually\n", certFile)
				fmt.Println("Falling back to HTTP mode")
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					panic(err)
				}
			} else {
				if err := server.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
					panic(err)
				}
			}
		} else {
			fmt.Printf("Starting HTTP server on %s\n", cfg.Address)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				panic(err)
			}
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

func printEmbeddedFiles() {
	fmt.Println("📦 Embedded files:")

	// Функция для рекурсивного обхода embedded файлов
	var walkDir func(string, string)
	walkDir = func(prefix string, dir string) {
		entries, err := embeddedFiles.ReadDir(dir)
		if err != nil {
			return
		}

		for _, entry := range entries {
			fullPath := dir + "/" + entry.Name()
			if dir == "." {
				fullPath = entry.Name()
			}

			displayPath := fullPath
			if prefix != "" {
				displayPath = prefix + "/" + entry.Name()
				if prefix == "." {
					displayPath = entry.Name()
				}
			}

			if entry.IsDir() {
				fmt.Printf("  📁 %s/\n", displayPath)
				walkDir(displayPath, fullPath)
			} else {
				fmt.Printf("     📄 %s\n", displayPath)
			}
		}
	}

	// Начинаем обход с корня
	walkDir("", ".")
}
