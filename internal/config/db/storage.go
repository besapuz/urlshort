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

// SaveURLWithConflictCheck - обновите для работы с user_id
func (s *DBStorage) SaveURLWithConflictCheck(ctx context.Context, uuid, shortID, originalURL, userID string) (string, error) {
	// Сначала проверяем, существует ли уже такой URL
	existingShortURL, err := s.GetShortURLByOriginalURL(ctx, originalURL)
	if err == nil && existingShortURL != "" {
		// URL уже существует - возвращаем существующий shortURL
		return existingShortURL, ErrURLConflict
	}

	// Если URL не существует, вставляем новую запись
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO url_mappings (uuid, short_url, original_url, user_id) VALUES ($1, $2, $3, $4)`,
		uuid, shortID, originalURL, userID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			// Если все же произошел конфликт (параллельный запрос), получаем существующий short_url
			existingShortURL, err := s.GetShortURLByOriginalURL(ctx, originalURL)
			if err != nil {
				return "", fmt.Errorf("failed to get existing short URL: %w", err)
			}
			return existingShortURL, ErrURLConflict
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

// GetURL - обновите метод получения URL для проверки флага deleted
func (s *DBStorage) GetURL(ctx context.Context, shortID string) (string, error) {
	var originalURL string
	var deleted bool

	err := s.DB.QueryRowContext(ctx,
		`SELECT original_url, deleted FROM url_mappings WHERE short_url = $1`,
		shortID).Scan(&originalURL, &deleted)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("URL not found for shortID: %s", shortID)
		}
		return "", fmt.Errorf("failed to get URL: %w", err)
	}

	if deleted {
		return "", fmt.Errorf("URL was deleted")
	}

	return originalURL, nil
}

type UserURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// GetUserURLs - обновите для исключения удаленных URL
func (s *DBStorage) GetUserURLs(ctx context.Context, userID string) ([]UserURL, error) {
	var urls []UserURL

	rows, err := s.DB.QueryContext(ctx,
		"SELECT short_url, original_url FROM url_mappings WHERE user_id = $1 AND deleted = false",
		userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user URLs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var url UserURL
		if err := rows.Scan(&url.ShortURL, &url.OriginalURL); err != nil {
			return nil, fmt.Errorf("failed to scan user URL: %w", err)
		}
		urls = append(urls, url)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	return urls, nil
}

// DeleteURLs - помечает URL как удаленные (soft delete)
func (s *DBStorage) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	// Начинаем транзакцию
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Подготавливаем запрос
	stmt, err := tx.PrepareContext(ctx,
		`UPDATE url_mappings SET deleted = true WHERE short_url = $1 AND user_id = $2`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Выполняем для каждого shortID
	for _, shortID := range shortIDs {
		_, err := stmt.ExecContext(ctx, shortID, userID)
		if err != nil {
			return fmt.Errorf("failed to delete URL %s: %w", shortID, err)
		}
	}

	// Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetStats возвращает статистику: количество URL и количество пользователей
func (s *DBStorage) GetStats() (int, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var urlsCount, usersCount int

	// Получаем количество URL (неудаленных)
	err := s.DB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM url_mappings WHERE deleted = false").Scan(&urlsCount)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get URLs count: %w", err)
	}

	// Получаем количество уникальных пользователей
	err = s.DB.QueryRowContext(ctx,
		"SELECT COUNT(DISTINCT user_id) FROM url_mappings WHERE user_id IS NOT NULL AND user_id != ''").Scan(&usersCount)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get users count: %w", err)
	}

	return urlsCount, usersCount, nil
}
