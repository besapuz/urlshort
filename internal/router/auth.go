package router

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// authenticateUser - аутентификация пользователя и установка куки
func authenticateUser(w http.ResponseWriter, r *http.Request, cookieSecret []byte) string {
	cookieName := "user_id"

	// Пытаемся получить существующую куку
	cookie, err := r.Cookie(cookieName)
	if err == nil && cookie != nil {
		// Проверяем подпись куки
		userID, valid := verifyCookie(cookie.Value, cookieSecret)
		if valid {
			return userID
		}
	}

	// Создаем нового пользователя
	userID := uuid.New().String()
	signedCookie := signUserID(userID, cookieSecret)

	// Устанавливаем новую куку
	newCookie := &http.Cookie{
		Name:     cookieName,
		Value:    signedCookie,
		Path:     "/",
		MaxAge:   24 * 60 * 60, // 24 часа
		HttpOnly: true,
		Secure:   false, // В продакшене должно быть true
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, newCookie)
	return userID
}

// signUserID - подписывает userID с помощью HMAC
func signUserID(userID string, cookieSecret []byte) string {
	mac := hmac.New(sha256.New, cookieSecret)
	mac.Write([]byte(userID))
	signature := hex.EncodeToString(mac.Sum(nil))
	return userID + "." + signature
}

// verifyCookie - проверяет подпись куки
func verifyCookie(cookieValue string, cookieSecret []byte) (string, bool) {
	parts := strings.Split(cookieValue, ".")
	if len(parts) != 2 {
		return "", false
	}

	userID := parts[0]
	expectedSignature := parts[1]

	mac := hmac.New(sha256.New, cookieSecret)
	mac.Write([]byte(userID))
	actualSignature := hex.EncodeToString(mac.Sum(nil))

	return userID, hmac.Equal([]byte(expectedSignature), []byte(actualSignature))
}
