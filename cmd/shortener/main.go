package main

import (
	"fmt"
	"net/http"

	"github.com/besapuz/urlshort/internal/router"
	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	// Обработка POST-запросов на корень
	r.Post("/", router.ShortenHandler)

	// Обработка GET-запросов к конкретному ID
	r.Get("/{id}", router.RedirectHandler)

	// Обработка главного маршрута (GET /)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Shortener service is running")
	})

	fmt.Println("Server started on http://localhost:8080")
	err := http.ListenAndServe("localhost:8080", r)
	if err != nil {
		panic(err)
	}
}
