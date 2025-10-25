package main

import (
	"flag"
)

var FlagRunAddr, FlagBaseURL string

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func parseFlags() {
	// регистрируем переменную flagRunAddr
	// как аргумент -a со значением :8080 по умолчанию
	flag.StringVar(&FlagRunAddr, "a", "http://localhost:8080", "address and port to run server")
	flag.StringVar(&FlagBaseURL, "b", "", "base URL for shortened links")

	// парсим переданные серверу аргументы в зарегистрированные переменные
	flag.Parse()
}
