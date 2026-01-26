package router

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func BenchmarkAuthenticateUser(b *testing.B) {
	secret := []byte("test-secret-key-12345")
	userID := "test-user-id"

	b.Run("SignUserID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			signUserID(userID, secret)
		}
	})

	b.Run("VerifyCookie", func(b *testing.B) {
		cookieValue := signUserID(userID, secret)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			verifyCookie(cookieValue, secret)
		}
	})
}

// Альтернативная реализация для сравнения
func signUserIDOptimized(userID string, cookieSecret []byte) string {
	mac := hmac.New(sha256.New, cookieSecret)
	mac.Write([]byte(userID))
	signature := hex.EncodeToString(mac.Sum(nil))
	return userID + "." + signature
}

func BenchmarkSignUserID_Optimized(b *testing.B) {
	secret := []byte("test-secret-key-12345")
	userID := "test-user-id"

	for i := 0; i < b.N; i++ {
		signUserIDOptimized(userID, secret)
	}
}
