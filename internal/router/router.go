package router

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/besapuz/urlshort/internal/app"
	"github.com/besapuz/urlshort/internal/config/db"
	"github.com/google/uuid"
)

var (
	dbstorage *db.DBStorage
	useDB     bool
)

var req struct {
	URL string `json:"url"`
}

type BatchRequestItem struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type BatchResponseItem struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// InitDBStorage - инициализация хранилища в базе данных
func InitDBStorage(dsn string) error {
	storage, err := db.NewDBStorage(dsn)
	if err != nil {
		return err
	}
	dbstorage = storage
	useDB = true
	return nil
}

// ShortenJSONHandler - обработчик POST-запросов в формате JSON.
func ShortenJSONHandler(baseURL, filePath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		// Дессирализация JSON
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		url := req.URL
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		shortID := app.GenerateShortID(8)
		newUUID := uuid.New().String()

		if useDB {
			if err := dbstorage.SaveURL(r.Context(), newUUID, shortID, url); err != nil {
				log.Printf("Error saving to database: %v", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
		} else if filePath != "" {
			urlMap[shortID] = url
			URLMappings = append(URLMappings, URLMapping{
				UUID:        newUUID,
				ShortURL:    shortID,
				OriginalURL: url,
			})
			if err := SaveToFile(filePath); err != nil {
				log.Printf("Error saving to file: %v", err)
			}
		} else {
			urlMap[shortID] = url
		}

		if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
			baseURL = "http://" + baseURL
		}

		result := map[string]string{"result": fmt.Sprintf("%s/%s", baseURL, shortID)}
		response, err := json.Marshal(result)
		if err != nil {
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(response)
	}
}

// ShortenHandler - обработчик POST-запросов.
func ShortenHandler(baseURL string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "text/plain" {
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		url := strings.TrimSpace(string(body))
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		filePath := GetStorageFilePath()

		shortID := app.GenerateShortID(8)
		newUUID := uuid.New().String()

		if useDB {
			if err := dbstorage.SaveURL(r.Context(), newUUID, shortID, url); err != nil {
				log.Printf("Error saving to database: %v", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
		} else if filePath != "" {
			urlMap[shortID] = url
			URLMappings = append(URLMappings, URLMapping{
				UUID:        newUUID,
				ShortURL:    shortID,
				OriginalURL: url,
			})
			if err := SaveToFile(filePath); err != nil {
				log.Printf("Error saving to file: %v", err)
			}
		} else {
			urlMap[shortID] = url
		}

		if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
			baseURL = "http://" + baseURL
		}
		resp := fmt.Sprintf("%s/%s", baseURL, shortID)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(resp))
	}
}

// RedirectHandler - обработчик GET-запросов.
func RedirectHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/")
	if id == "" {
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	var exists bool
	var url string
	var err error
	if useDB {
		url, err = dbstorage.GetURL(r.Context(), id)
		if err != nil {
			http.Error(w, "", http.StatusBadRequest)
			return
		}
	} else {
		url, exists = urlMap[id]
		if !exists {
			http.Error(w, "", http.StatusBadRequest)
			return
		}
	}
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// PingHandler - обработчик для проверки соединения с БД.
func PingHandler(w http.ResponseWriter, r *http.Request) {
	if !useDB || dbstorage == nil {
		http.Error(w, "Database not configurated", http.StatusInternalServerError)
		return
	}
	if err := dbstorage.Ping(); err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// BatchShortenHandler - обработчик для пакетного сокращения URL
func BatchShortenHandler(baseURL, filePath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			http.Error(w, "Empty or invalid request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var batchRequests []BatchRequestItem
		if err := json.Unmarshal(body, &batchRequests); err != nil {
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}

		if len(batchRequests) == 0 {
			http.Error(w, "Empty batch", http.StatusBadRequest)
			return
		}

		// Валидация URL
		for _, item := range batchRequests {
			if !strings.HasPrefix(item.OriginalURL, "http://") &&
				!strings.HasPrefix(item.OriginalURL, "https://") {
				http.Error(w, fmt.Sprintf("Invalid URL in item: %s", item.CorrelationID), http.StatusBadRequest)
				return
			}
		}

		// Формируем базовый URL
		if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
			baseURL = "http://" + baseURL
		}

		var batchResponses []BatchResponseItem

		// Обработка для базы данных
		if useDB {
			// Сохраняем все URL в одной транзакции
			tx, err := dbstorage.DB.Begin()
			if err != nil {
				log.Printf("Error starting transaction: %v", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			defer tx.Rollback()

			for _, item := range batchRequests {
				shortID := app.GenerateShortID(8)
				newUUID := uuid.New().String()

				_, err := tx.ExecContext(r.Context(),
					`INSERT INTO url_mappings (uuid, short_url, original_url) VALUES ($1, $2, $3)`,
					newUUID, shortID, item.OriginalURL)

				if err != nil {
					log.Printf("Error saving URL to database: %v", err)
					http.Error(w, "Database error", http.StatusInternalServerError)
					return
				}

				batchResponses = append(batchResponses, BatchResponseItem{
					CorrelationID: item.CorrelationID,
					ShortURL:      fmt.Sprintf("%s/%s", baseURL, shortID),
				})
			}

			if err := tx.Commit(); err != nil {
				log.Printf("Error committing transaction: %v", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
		} else {
			// Обработка для файлового хранилища и памяти
			mutex.Lock()
			defer mutex.Unlock()

			for _, item := range batchRequests {
				shortID := app.GenerateShortID(8)
				newUUID := uuid.New().String()

				// Сохраняем в память
				urlMap[shortID] = item.OriginalURL

				// Сохраняем в файловое хранилище, если указан путь
				if filePath != "" {
					URLMappings = append(URLMappings, URLMapping{
						UUID:        newUUID,
						ShortURL:    shortID,
						OriginalURL: item.OriginalURL,
					})
				}

				batchResponses = append(batchResponses, BatchResponseItem{
					CorrelationID: item.CorrelationID,
					ShortURL:      fmt.Sprintf("%s/%s", baseURL, shortID),
				})
			}

			// Сохраняем в файл, если указан путь
			if filePath != "" {
				if err := SaveToFile(filePath); err != nil {
					log.Printf("Error saving to file: %v", err)
					// Не прерываем выполнение, только логируем ошибку
				}
			}
		}

		response, err := json.Marshal(batchResponses)
		if err != nil {
			http.Error(w, "Error creating response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(response)
	}
}
