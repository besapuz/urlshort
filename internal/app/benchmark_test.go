package app

import (
	"testing"
)

func BenchmarkGenerateShortID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateShortID(8)
	}
}

func BenchmarkGenerateShortID_Length4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateShortID(4)
	}
}

func BenchmarkGenerateShortID_Length16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateShortID(16)
	}
}
