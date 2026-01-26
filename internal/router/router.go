package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/besapuz/urlshort/internal/app"
	"github.com/besapuz/urlshort/internal/audit"
	"github.com/besapuz/urlshort/internal/config/db"
	"github.com/google/uuid"
)

type URLShortener struct {
	DBStorage       *db.DBStorage
	UseDB           bool
	FileStoragePath string
	BaseURL         string
	CookieSecret    []byte
}

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

type UserURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// InitDBStorage - инициализация хранилища в базе данных
func (s *URLShortener) InitDBStorage(dsn string) error {
	storage, err := db.NewDBStorage(dsn)
	if err != nil {
		return err
	}
	s.DBStorage = storage
	s.UseDB = true
	return nil
}

// ShortenJSONHandler - обработчик POST-запросов в формате JSON.
func (s *URLShortener) ShortenJSONHandler(baseURL, filePath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Аутентифицируем пользователя
		userID := authenticateUser(w, r, s.CookieSecret)

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

		defer func() {
			// Аудит успешного создания
			if r.Method == http.MethodPost && err == nil && url != "" {
				audit.LogEvent(audit.ActionShorten, userID, url)
			}
		}()

		if s.UseDB {
			savedShortID, err := s.DBStorage.SaveURLWithConflictCheck(r.Context(), newUUID, shortID, url, userID)
			if err != nil {
				if errors.Is(err, db.ErrURLConflict) {
					// URL уже существует - возвращаем конфликт
					result := map[string]string{"result": fmt.Sprintf("%s/%s", baseURL, savedShortID)}
					response, _ := json.Marshal(result)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusConflict)
					w.Write(response)
					return
				}
				log.Printf("Error saving to database: %v", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			shortID = savedShortID // Используем фактически сохраненный shortID
		} else if filePath != "" {
			urlMap[shortID] = url
			URLMappings = append(URLMappings, URLMapping{
				UUID:        newUUID,
				ShortURL:    shortID,
				OriginalURL: url,
				UserID:      userID,
			})
			if err := SaveToFile(filePath); err != nil {
				log.Printf("Error saving to file: %v", err)
			}
		} else {
			urlMap[shortID] = url
			if userURLs, exists := userURLsMap[userID]; exists {
				userURLsMap[userID] = append(userURLs, shortID)
			} else {
				userURLsMap[userID] = []string{shortID}
			}
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
func (s *URLShortener) ShortenHandler(baseURL string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := authenticateUser(w, r, s.CookieSecret)
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

		defer func() {
			// Аудит успешного создания
			if r.Method == http.MethodPost && err == nil {
				audit.LogEvent(audit.ActionShorten, userID, url)
			}
		}()

		if s.UseDB {
			savedShortID, err := s.DBStorage.SaveURLWithConflictCheck(r.Context(), newUUID, shortID, url, userID)
			if err != nil {
				if errors.Is(err, db.ErrURLConflict) {
					// URL уже существует - возвращаем конфликт
					resp := fmt.Sprintf("%s/%s", baseURL, savedShortID)
					w.Header().Set("Content-Type", "text/plain")
					w.WriteHeader(http.StatusConflict)
					w.Write([]byte(resp))
					return
				}
				log.Printf("Error saving to database: %v", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			shortID = savedShortID // Используем фактически сохраненный shortID
		} else if filePath != "" {
			urlMap[shortID] = url
			URLMappings = append(URLMappings, URLMapping{
				UUID:        newUUID,
				ShortURL:    shortID,
				OriginalURL: url,
				UserID:      userID,
			})
			if err := SaveToFile(filePath); err != nil {
				log.Printf("Error saving to file: %v", err)
			}
		} else {
			urlMap[shortID] = url
			if userURLs, exists := userURLsMap[userID]; exists {
				userURLsMap[userID] = append(userURLs, shortID)
			} else {
				userURLsMap[userID] = []string{shortID}
			}
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

// GetUserURLsHandler - обработчик для получения всех URL пользователя
func (s *URLShortener) GetUserURLsHandler(baseURL string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверяем аутентификацию
		userID := authenticateUser(w, r, s.CookieSecret)

		var userURLs []UserURLResponse

		if s.UseDB {
			// Получаем URL пользователя из базы данных
			urls, err := s.DBStorage.GetUserURLs(r.Context(), userID)
			if err != nil {
				log.Printf("Error getting user URLs from database: %v", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}

			fullBaseURL := baseURL
			if !strings.HasPrefix(fullBaseURL, "http://") && !strings.HasPrefix(fullBaseURL, "https://") {
				fullBaseURL = "http://" + fullBaseURL
			}

			for _, url := range urls {
				userURLs = append(userURLs, UserURLResponse{ShortURL: fmt.Sprintf("%s/%s", fullBaseURL, url.ShortURL), // Используем fullBaseURL с протоколом
					OriginalURL: url.OriginalURL,
				})
			}
		} else {
			// Получаем URL пользователя из файлового хранилища или памяти
			mutex.Lock()
			defer mutex.Unlock()

			fullBaseURL := baseURL
			if !strings.HasPrefix(fullBaseURL, "http://") && !strings.HasPrefix(fullBaseURL, "https://") {
				fullBaseURL = "http://" + fullBaseURL
			}

			if filePath := GetStorageFilePath(); filePath != "" {
				// Ищем в файловом хранилище
				for _, mapping := range URLMappings {
					if mapping.UserID == userID && !mapping.DeletedFlag { // Добавляем проверку на удаление
						userURLs = append(userURLs, UserURLResponse{
							ShortURL:    fmt.Sprintf("%s/%s", fullBaseURL, mapping.ShortURL),
							OriginalURL: mapping.OriginalURL,
						})
					}
				}
			} else {
				// Ищем в in-memory хранилище
				if shortIDs, exists := userURLsMap[userID]; exists {
					for _, shortID := range shortIDs {
						if originalURL, exists := urlMap[shortID]; exists {
							userURLs = append(userURLs, UserURLResponse{
								ShortURL:    fmt.Sprintf("%s/%s", fullBaseURL, shortID),
								OriginalURL: originalURL,
							})
						}
					}
				}
			}
		}

		if len(userURLs) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Возвращаем список URL
		response, err := json.Marshal(userURLs)
		if err != nil {
			http.Error(w, "Error creating response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	}
}

// RedirectHandler - обработчик GET-запросов.
func (s *URLShortener) RedirectHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/")
	if id == "" {
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	var exists bool
	var url string
	var err error
	// Получаем userID из куки
	userID := authenticateUser(w, r, s.CookieSecret)
	defer func() {
		// Аудит успешного перехода по ссылке
		if r.Method == http.MethodGet && (exists || url != "") {
			audit.LogEvent(audit.ActionFollow, userID, url)
		}
	}()
	if s.UseDB {
		url, err = s.DBStorage.GetURL(r.Context(), id)
		if err != nil {
			if err.Error() == "URL was deleted" {
				http.Error(w, "URL was deleted", http.StatusGone)
				return
			}
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
func (s *URLShortener) PingHandler(w http.ResponseWriter, r *http.Request) {
	if !s.UseDB || s.DBStorage == nil {
		http.Error(w, "Database not configurated", http.StatusInternalServerError)
		return
	}
	if err := s.DBStorage.Ping(); err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// BatchShortenHandler - обработчик для пакетного сокращения URL
func (s *URLShortener) BatchShortenHandler(baseURL, filePath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := authenticateUser(w, r, s.CookieSecret)
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

		// Обработка для базы данных
		var batchResponses []BatchResponseItem

		if s.UseDB {
			// Используем отдельные вызовы SaveURLWithConflictCheck для каждого URL
			for _, item := range batchRequests {
				shortID := app.GenerateShortID(8)
				newUUID := uuid.New().String()

				savedShortID, err := s.DBStorage.SaveURLWithConflictCheck(r.Context(), newUUID, shortID, item.OriginalURL, userID)
				if err != nil {
					if errors.Is(err, db.ErrURLConflict) {
						// URL уже существует - используем существующий
						batchResponses = append(batchResponses, BatchResponseItem{
							CorrelationID: item.CorrelationID,
							ShortURL:      fmt.Sprintf("%s/%s", baseURL, savedShortID),
						})
						continue
					}
					log.Printf("Error saving URL to database: %v", err)
					http.Error(w, "Database error", http.StatusInternalServerError)
					return
				}

				batchResponses = append(batchResponses, BatchResponseItem{
					CorrelationID: item.CorrelationID,
					ShortURL:      fmt.Sprintf("%s/%s", baseURL, savedShortID),
				})
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
						UserID:      userID,
					})
				} else {
					// Для in-memory хранилища сохраняем userID
					if userURLs, exists := userURLsMap[userID]; exists {
						userURLsMap[userID] = append(userURLs, shortID)
					} else {
						userURLsMap[userID] = []string{shortID}
					}
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

// DeleteURLsHandler - улучшенный обработчик для удаления URL с fan-in паттерном
func (s *URLShortener) DeleteURLsHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверяем аутентификацию
		userID := authenticateUser(w, r, s.CookieSecret)

		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
			return
		}

		// Читаем тело запроса
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Парсим JSON с short IDs
		var shortIDs []string
		if err := json.Unmarshal(body, &shortIDs); err != nil {
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}

		if len(shortIDs) == 0 {
			http.Error(w, "Empty request", http.StatusBadRequest)
			return
		}

		// Логируем запрос на удаление
		log.Printf("Received delete request for %d URLs from user %s", len(shortIDs), userID)

		// Для базы данных - используем улучшенный метод с fan-in паттерном
		if s.UseDB {
			go func(ids []string, uid string) {
				start := time.Now()
				log.Printf("Starting async deletion of %d URLs for user %s", len(ids), uid)

				if err := s.deleteURLsBatch(ids, uid); err != nil {
					log.Printf("Error deleting URLs in batch: %v", err)
				} else {
					duration := time.Since(start)
					log.Printf("Successfully completed deletion of %d URLs for user %s in %v",
						len(ids), uid, duration)
				}
			}(shortIDs, userID)
		} else {
			// Для файлового хранилища и памяти
			go func(ids []string, uid string) {
				start := time.Now()
				log.Printf("Starting async deletion of %d URLs for user %s (file/memory)", len(ids), uid)

				if err := s.deleteURLsBatch(ids, uid); err != nil {
					log.Printf("Error deleting URLs in batch: %v", err)
				} else {
					duration := time.Since(start)
					log.Printf("Successfully completed deletion of %d URLs for user %s in %v",
						len(ids), uid, duration)
				}
			}(shortIDs, userID)
		}

		// Возвращаем Accepted, так как удаление происходит асинхронно
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Delete request accepted"))
	}
}

// deleteURLsBatch - удаляет URL в батче с использованием fan-in паттерна
func (s *URLShortener) deleteURLsBatch(shortIDs []string, userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	batchSize := 10 // Уменьшаем размер батча для лучшей параллельности
	batches := make(chan []string)
	results := make(chan error, len(shortIDs)/batchSize+1) // Буферизованный канал

	// Горутина для разбивки на батчи
	go func() {
		defer close(batches)
		for i := 0; i < len(shortIDs); i += batchSize {
			end := i + batchSize
			if end > len(shortIDs) {
				end = len(shortIDs)
			}
			batches <- shortIDs[i:end]
		}
	}()

	// Запускаем воркеры для обработки батчей
	numWorkers := 3
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for batch := range batches {
				log.Printf("Worker %d processing batch of %d URLs", workerID, len(batch))
				err := s.processBatch(ctx, batch, userID)
				results <- err
				if err != nil {
					log.Printf("Worker %d error: %v", workerID, err)
				} else {
					log.Printf("Worker %d successfully processed batch", workerID)
				}
			}
		}(i)
	}

	// Закрываем results после завершения всех воркеров
	go func() {
		wg.Wait()
		close(results)
	}()

	// Собираем результаты (fan-in)
	var finalErr error
	processedBatches := 0
	totalBatches := (len(shortIDs) + batchSize - 1) / batchSize

	for err := range results {
		processedBatches++
		if err != nil && finalErr == nil {
			finalErr = err
		}
		log.Printf("Processed batch %d/%d", processedBatches, totalBatches)
	}

	log.Printf("Completed batch deletion: %d/%d batches processed", processedBatches, totalBatches)
	return finalErr
}

// processBatch - обрабатывает один батч URL для удаления
func (s *URLShortener) processBatch(ctx context.Context, shortIDs []string, userID string) error {
	if s.UseDB {
		return s.DBStorage.DeleteURLs(ctx, shortIDs, userID)
	}

	// Для файлового хранилища и памяти
	mutex.Lock()
	defer mutex.Unlock()

	// Помечаем URL как удаленные
	for _, shortID := range shortIDs {
		// Ищем в файловом хранилище
		if filePath := GetStorageFilePath(); filePath != "" {
			for i := range URLMappings {
				if URLMappings[i].ShortURL == shortID && URLMappings[i].UserID == userID {
					URLMappings[i].DeletedFlag = true
				}
			}
		} else {
			// Для in-memory хранилища - удаляем из urlMap
			delete(urlMap, shortID)
			// И из userURLsMap
			if userURLs, exists := userURLsMap[userID]; exists {
				for i, id := range userURLs {
					if id == shortID {
						userURLsMap[userID] = append(userURLs[:i], userURLs[i+1:]...)
						break
					}
				}
			}
		}
	}

	// Сохраняем изменения в файл, если указан путь
	if filePath := GetStorageFilePath(); filePath != "" {
		if err := SaveToFile(filePath); err != nil {
			return fmt.Errorf("failed to save to file: %w", err)
		}
	}

	return nil
}
