package app

import "math/rand/v2"

var (
	chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rng   *rand.Rand
)

// initRand - инициализирует генератора случайных чисел.
func initRand() {
	rng = rand.New(rand.NewPCG(123456789, 987654321))
}

// GenerateShortID - генерирует случайныый идентификатор заданной длины.
func GenerateShortID(length int) string {
	initRand()
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[rng.IntN(len(chars))]
	}
	return string(b)
}
