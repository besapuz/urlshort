package router

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/besapuz/urlshort/internal/app"
)

var urlMap = make(map[string]string)

// ShortenHandler - обработчик POST-запросов.
func ShortenHandler(baseURL string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка типа контента
		if r.Header.Get("Content-Type") != "text/plain" {
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		// Чтение тела запроса
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Очистка от пробелом и валидация URL
		url := strings.TrimSpace(string(body))
		if !strings.HasPrefix(string(body), "http://") && !strings.HasPrefix(string(body), "https://") {
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		shortID := app.GenerateShortID(8)
		urlMap[shortID] = url
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
