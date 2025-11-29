package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DBStorage struct {
	DB *sql.DB
}

// ErrURLConflict - ошибка конфликта URL
var ErrURLConflict = errors.New("URL already exists")

// NewDBStorage - конструктор для DBStorage.
func NewDBStorage(dsn string) (*DBStorage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	return &DBStorage{DB: db}, nil
}

// runMigrations - выполняет миграции БД.
func runMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Получаем путь к миграциям относительно этого файла
	_, filename, _, _ := runtime.Caller(0)
	migrationsPath := filepath.Join(filepath.Dir(filename), "../../../migrations")

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	// Применяем миграции
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

// Ping - функция для проверки соединения с базой данных.
func (s *DBStorage) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.DB.PingContext(ctx)
}

// Close - функция для закрытия соединения с базой данных.
func (s *DBStorage) Close() error {
	return s.DB.Close()
}

// SaveURL - сохранение URL в базу данных
func (s *DBStorage) SaveURL(ctx context.Context, uuid, shortID, originalURL string) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO url_mappings (uuid, short_url, original_url) VALUES ($1, $2, $3)`,
		uuid, shortID, originalURL)
	if err != nil {
		return fmt.Errorf("failed to save URL: %w", err)
	}
	return nil
}

// SaveURLWithConflictCheck - сохранение URL с проверкой конфликта
func (s *DBStorage) SaveURLWithConflictCheck(ctx context.Context, uuid, shortID, originalURL, userID string) (string, error) {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO url_mappings (uuid, short_url, original_url) VALUES ($1, $2, $3, $4)`,
		uuid, shortID, originalURL, userID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			// Если это нарушение уникальности по original_url, получаем существующий short_url
			if pgErr.ConstraintName == "idx_original_url" {
				existingShortURL, err := s.GetShortURLByOriginalURL(ctx, originalURL)
				if err != nil {
					return "", fmt.Errorf("failed to get existing short URL: %w", err)
				}
				return existingShortURL, ErrURLConflict
			}
		}
		return "", fmt.Errorf("failed to save URL: %w", err)
	}
	return shortID, nil
}

// GetShortURLByOriginalURL - получение short_url по original_url
func (s *DBStorage) GetShortURLByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	var shortURL string
	err := s.DB.QueryRowContext(ctx,
		`SELECT short_url FROM url_mappings WHERE original_url = $1`,
		originalURL).Scan(&shortURL)
	if err != nil {
		return "", fmt.Errorf("failed to get short URL: %w", err)
	}
	return shortURL, nil
}

// GetURL - получение URL из базы данных.
func (s *DBStorage) GetURL(ctx context.Context, shortID string) (string, error) {
	var originalURL string
	err := s.DB.QueryRowContext(ctx,
		`SELECT original_url FROM url_mappings WHERE short_url = $1`,
		shortID).Scan(&originalURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("URL not found for shortID: %s", shortID)
		}
		return "", fmt.Errorf("failed to get URL: %w", err)
	}
	return originalURL, nil
}

type URLMapping struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// GetUserURLs - получение всех URL пользователя
func (s *DBStorage) GetUserURLs(ctx context.Context, userID string) ([]URLMapping, error) {
	var urls []URLMapping

	rows, err := s.DB.QueryContext(ctx,
		"SELECT short_url, original_url FROM url_mappings WHERE user_id = $1", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user URLs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var url URLMapping
		if err := rows.Scan(&url.ShortURL, &url.OriginalURL); err != nil {
			return nil, fmt.Errorf("failed to scan user URL: %w", err)
		}
		urls = append(urls, url)
	}

	// Проверяем ошибки после итерации по rows
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	return urls, nil
}
