package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type DBStorage struct {
	db *sql.DB
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

	if err := createTable(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("filed to create table: %w", err)
	}
	return &DBStorage{db: db}, nil
}

// createTable - функция для создания таблицы в базе данных.
func createTable(db *sql.DB) error {
	query := `
	CREATE TABLE videos(
	"video_id" TEXT,
	"trending_date" TEXT,
	"title" TEXT,
	"channel_title" TEXT,
	"category_id" INTEGER,
	"publish_time" TEXT,
	"tags" TEXT,
	"views" INTEGER,
	"likes" INTEGER,
	"dislikes" INTEGER,
	"comment_count" INTEGER,
	"thumbnail_link" TEXT,
	"comments_disabled" BOOLEAN,
	"ratings_disabled" BOOLEAN,
	"video_error_or_removed" BOOLEAN,
	"description" TEXT
	)`
	_, err := db.Exec(query)
	return err
}

// Ping - функция для проверки соединения с базой данных.
func (s *DBStorage) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.db.PingContext(ctx)
}

// Close - функция для закрытия соединения с базой данных.
func (s *DBStorage) Close() error {
	return s.db.Close()
}
