package router

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/besapuz/urlshort/internal/config"
	"github.com/google/uuid"
)

// CookieManager управляет аутентификацией через куки
type CookieManager struct {
	cookieSecret []byte
}

// NewCookieManager создает новый менеджер кук
func NewCookieManager(cfg *config.Config) *CookieManager {
	return &CookieManager{
		cookieSecret: cfg.GetCookieSecret(),
	}
}

// authenticateUser - аутентификация пользователя и установка куки
func (cm *CookieManager) authenticateUser(w http.ResponseWriter, r *http.Request) string {
	cookieName := "user_id"

	// Пытаемся получить существующую куку
	cookie, err := r.Cookie(cookieName)
	if err == nil && cookie != nil {
		// Проверяем подпись куки
		userID, valid := cm.verifyCookie(cookie.Value)
		if valid {
			return userID
		}
	}

	// Создаем нового пользователя
	userID := uuid.New().String()
	signedCookie := cm.signUserID(userID)

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
func (cm *CookieManager) signUserID(userID string) string {
	mac := hmac.New(sha256.New, cm.cookieSecret)
	mac.Write([]byte(userID))
	signature := hex.EncodeToString(mac.Sum(nil))
	return userID + "." + signature
}

// verifyCookie - проверяет подпись куки
func (cm *CookieManager) verifyCookie(cookieValue string) (string, bool) {
	parts := strings.Split(cookieValue, ".")
	if len(parts) != 2 {
		return "", false
	}

	userID := parts[0]
	expectedSignature := parts[1]

	mac := hmac.New(sha256.New, cm.cookieSecret)
	mac.Write([]byte(userID))
	actualSignature := hex.EncodeToString(mac.Sum(nil))

	return userID, hmac.Equal([]byte(expectedSignature), []byte(actualSignature))
}
