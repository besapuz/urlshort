package db

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DBStorage struct {
	DB *sql.DB
}

// NewDBStorage - конструктор для DBStorage.
func NewDBStorage(dsn string) (*DBStorage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("filed to database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("filed to ping database: %w", err)
	}
	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("filed to run migrations: %w", err)
	}

	if err := createTable(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("filed to create table: %w", err)
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

// createTable - функция для создания таблицы в базе данных.
func createTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS url_mappings (
		id SERIAL PRIMARY KEY,
		uuid TEXT UNIQUE NOT NULL,
		short_url TEXT UNIQUE NOT NULL,
		original_url TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`
	_, err := db.Exec(query)
	return err
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

// SaveURL - сохранение URL в базу данных.
func (s *DBStorage) SaveURL(ctx context.Context, db *sql.DB, uuid, shortID, originalURL string) error {
	result, err := db.ExecContext(ctx, `INSERT INTO url_mappings (uuid, shortID, originalURL) VALEUS ($1, $2, $3)`, uuid, shortID, originalURL)
	if err != nil {
		return err
	}
	result.LastInsertId()
	return nil
}

// GetURL - получение URL из базы данных.
func (s *DBStorage) GetURL(ctx context.Context, db *sql.DB, shortID string) (string, error) {
	var originalURL string
	rows, err := db.QueryContext(ctx, `SELECT original_url FROM url_mappings WHERE short_url = $1`, shortID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	err = rows.Scan(&originalURL)
	if err != nil {
		return "", err
	}
	return originalURL, nil
}
