package app

import (
	"math/rand/v2"
	"sync"
)

var (
	chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rng   *rand.Rand
	once  sync.Once
)

// initRand - инициализирует генератора случайных чисел.
// Идентификатор состоит из символов a-z, A-Z и 0-9.
// Функция использует криптографически безопасный генератор случайных чисел.
// Длина должна быть положительным числом.
func initRand() {
	once.Do(func() {
		rng = rand.New(rand.NewPCG(123456789, 987654321))
	})
}

// GenerateShortID - генерирует случайныый идентификатор заданной длины.
// Вызывается один раз при первом использовании GenerateShortID.
func GenerateShortID(length int) string {
	initRand()
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[rng.IntN(len(chars))]
	}
	return string(b)
}
