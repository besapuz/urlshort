package router

import (
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
)

var (
	urlMap = make(map[string]string)
	chars  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rng    *rand.Rand
)

// initRand - инициализирует генератора случайных чисел.
func initRand() {
	rng = rand.New(rand.NewPCG(123456789, 987654321))
}

// generateShortID - генерирует случайныый идентификатор заданной длины.
func generateShortID(length int) string {
	initRand()
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[rng.IntN(len(chars))]
	}
	return string(b)
}

// ShortenHandler - обработчик POST-запросов.
func ShortenHandler(w http.ResponseWriter, r *http.Request) {
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

	shortID := generateShortID(8)
	urlMap[shortID] = url

	resp := fmt.Sprintf("http://localhost:8080/%s", shortID)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(resp))
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
