package main

import (
	"flag"
)

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func parseFlags() {
	// регистрируем переменную flagRunAddr
	// как аргумент -a со значением :8080 по умолчанию
	var flagRunAddr string
	flag.StringVar(&flagRunAddr, "a", ":8080", "address and port to run server")
	var flagBaseURL string
	flag.StringVar(&flagBaseURL, "b", "http://localhost:8080", "base URL for shortened links")

	// парсим переданные серверу аргументы в зарегистрированные переменные
	flag.Parse()
}
