package router

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/besapuz/urlshort/internal/app"
	"github.com/besapuz/urlshort/internal/config/db"
	"github.com/google/uuid"
)

var (
	dbstorage    *db.DBStorage
	useDB        bool
	cookieSecret = []byte("ncklsj8s9c8ysgjc-scishb")
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

type UserURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// authenticateUser - аутентификация пользователя и установка куки
func authenticateUser(w http.ResponseWriter, r *http.Request) string {
	cookieName := "user_id"

	// Пытаемся получить существующую куку
	cookie, err := r.Cookie(cookieName)
	if err == nil && cookie != nil {
		// Проверяем подпись куки
		userID, valid := verifyCookie(cookie.Value)
		if valid {
			return userID
		}
	}

	// Создаем нового пользователя
	userID := uuid.New().String()
	signedCookie := signUserID(userID)

	// Устанавливаем новую куку
	newCookie := &http.Cookie{
		Name:     cookieName,
		Value:    signedCookie,
		Path:     "/",
		MaxAge:   24 * 60 * 60, // 24 часа
		HttpOnly: true,
		Secure:   false, // В продакшене должно быть true
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, newCookie)
	return userID
}

// signUserID - подписывает userID с помощью HMAC
func signUserID(userID string) string {
	mac := hmac.New(sha256.New, cookieSecret)
	mac.Write([]byte(userID))
	signature := hex.EncodeToString(mac.Sum(nil))
	return userID + "." + signature
}

// verifyCookie - проверяет подпись куки
func verifyCookie(cookieValue string) (string, bool) {
	parts := strings.Split(cookieValue, ".")
	if len(parts) != 2 {
		return "", false
	}

	userID := parts[0]
	expectedSignature := parts[1]

	mac := hmac.New(sha256.New, cookieSecret)
	mac.Write([]byte(userID))
	actualSignature := hex.EncodeToString(mac.Sum(nil))

	return userID, hmac.Equal([]byte(expectedSignature), []byte(actualSignature))
}

// getAuthenticatedUserID - получает аутентифицированный userID из куки
func getAuthenticatedUserID(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("user_id")
	if err != nil {
		return "", false
	}

	userID, valid := verifyCookie(cookie.Value)
	if !valid {
		return "", false
	}

	return userID, true
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
		// Аутентифицируем пользователя
		userID := authenticateUser(w, r)

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

		if useDB {
			savedShortID, err := dbstorage.SaveURLWithConflictCheck(r.Context(), newUUID, shortID, url, userID)
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
func ShortenHandler(baseURL string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := authenticateUser(w, r)
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
			savedShortID, err := dbstorage.SaveURLWithConflictCheck(r.Context(), newUUID, shortID, url, userID)
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
func GetUserURLsHandler(baseURL string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверяем аутентификацию
		userID := authenticateUser(w, r)

		var userURLs []UserURLResponse

		if useDB {
			// Получаем URL пользователя из базы данных
			urls, err := dbstorage.GetUserURLs(r.Context(), userID)
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
					if mapping.UserID == userID && !mapping.Deleted { // Добавляем проверку на удаление
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
		userID := authenticateUser(w, r)
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

		if useDB {
			// Используем отдельные вызовы SaveURLWithConflictCheck для каждого URL
			for _, item := range batchRequests {
				shortID := app.GenerateShortID(8)
				newUUID := uuid.New().String()

				savedShortID, err := dbstorage.SaveURLWithConflictCheck(r.Context(), newUUID, shortID, item.OriginalURL, userID)
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

// DeleteURLsHandler - обработчик для удаления URL
func DeleteURLsHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверяем аутентификацию
		userID := authenticateUser(w, r)

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

		// Для базы данных
		if useDB {
			// Запускаем удаление в отдельной горутине (асинхронно)
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				if err := dbstorage.DeleteURLs(ctx, shortIDs, userID); err != nil {
					log.Printf("Error deleting URLs: %v", err)
				}
			}()
		} else {
			// Для файлового хранилища и памяти
			go func() {
				mutex.Lock()
				defer mutex.Unlock()

				// Помечаем URL как удаленные в памяти
				for _, shortID := range shortIDs {
					// Ищем в файловом хранилище
					if filePath := GetStorageFilePath(); filePath != "" {
						for i := range URLMappings {
							if URLMappings[i].ShortURL == shortID && URLMappings[i].UserID == userID {
								URLMappings[i].Deleted = true
							}
						}
						// Сохраняем изменения в файл
						if err := SaveToFile(filePath); err != nil {
							log.Printf("Error saving to file: %v", err)
						}
					} else {
						// Для in-memory хранилища просто удаляем из userURLsMap
						if userURLs, exists := userURLsMap[userID]; exists {
							for i, id := range userURLs {
								if id == shortID {
									// Удаляем из списка пользователя
									userURLsMap[userID] = append(userURLs[:i], userURLs[i+1:]...)
									break
								}
							}
						}
					}
				}
			}()
		}

		// Возвращаем Accepted, так как удаление происходит асинхронно
		w.WriteHeader(http.StatusAccepted)
	}
}
