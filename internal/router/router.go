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
		urlMap[shortID] = url
		URLMappings = append(URLMappings, URLMapping{
			UUID:        newUUID,
			ShortURL:    shortID,
			OriginalURL: url,
		})
		if err := SaveToFile(filePath); err != nil {
			log.Printf("Error saving to file: %v", err)
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

// ShortenHandler - обработчик POST-запросов
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

		shortID := app.GenerateShortID(8)
		urlMap[shortID] = url

		URLMappings = append(URLMappings, URLMapping{
			UUID:        uuid.New().String(),
			ShortURL:    shortID,
			OriginalURL: url,
		})

		filePath := GetStorageFilePath()
		if filePath != "" {
			if err := SaveToFile(filePath); err != nil {
				log.Printf("Error saving to file: %v", err)
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

// RedirectHandler - обработчик GET-запросов.
func RedirectHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/")
	if id == "" {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	url, exists := urlMap[id]
	if !exists {
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// PingHandler - обработчик для проверки соединения с БД.
func PingHandler(w http.ResponseWriter, r *http.Request) {
	if useDB || dbstorage == nil {
		http.Error(w, "Database not configurated", http.StatusInternalServerError)
		return
	}
	if err := dbstorage.Ping(); err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
