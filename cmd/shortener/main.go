package main

import (
	"fmt"
	"net/http"

	"github.com/besapuz/urlshort/internal/config"
	"github.com/besapuz/urlshort/internal/router"
	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := config.NewConfig()
	r := chi.NewRouter()

	// Обработка POST-запросов на корень
	r.Post("/", router.ShortenHandler(cfg.BaseURL))

	// Обработка GET-запросов к конкретному ID
	r.Get("/{id}", router.RedirectHandler)

	// Обработка главного маршрута (GET /)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Shortener service is running at %s", cfg.BaseURL)
	})

	fmt.Printf("Server started on http://%s\n", cfg.BaseURL)
	err := http.ListenAndServe(cfg.Address, r)
	if err != nil {
		panic(err)
	}
}
