package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRedirectHandler - тестирование функции main/redirectHandler.
func TestRedirectHandler(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		setup          func()
		expectedStatus int
		expectedHeader string
	}{
		{
			name:           "Пустой идентификатор",
			path:           "/",
			setup:          func() { urlMap = map[string]string{} },
			expectedStatus: http.StatusBadRequest,
			expectedHeader: "",
		},
		{
			name:           "Валидный идентификатор",
			path:           "/example",
			setup:          func() { urlMap = map[string]string{"example": "https://example.com"} },
			expectedStatus: http.StatusTemporaryRedirect,
			expectedHeader: "https://example.com",
		},
		{
			name:           "Невалидный идентификатор",
			path:           "/invalid",
			setup:          func() { urlMap = map[string]string{"valid": "https://valid.com"} },
			expectedStatus: http.StatusBadRequest,
			expectedHeader: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			req, err := http.NewRequest("GET", tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			RedirectHandler(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler вернул неправильный статус: получил %v, ожидал %v", status, tt.expectedStatus)
			}

			location := rr.Header().Get("Location")
			if location != tt.expectedHeader {
				t.Errorf("Заголовок Location не совпадает: получил %q, ожидал %q", location, tt.expectedHeader)
			}
		})
	}
}

// TestShortenHandler - тестирование функции main/shortenHandler.

func TestShortenHandler(t *testing.T) {

	tests := []struct {
		name           string
		contentType    string
		body           string
		expectedStatus int
	}{
		{
			name:           "Invalid Content-Type",
			contentType:    "application/json",
			body:           "https://example.com",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty body",
			contentType:    "text/plain",
			body:           "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid URL format",
			contentType:    "text/plain",
			body:           "example.com",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Valid request",
			contentType:    "text/plain",
			body:           "https://example.com",
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Сброс глобального состояния
			urlMap = make(map[string]string)

			// Создание запроса
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)

			// Создание ответа
			rr := httptest.NewRecorder()

			// Вызов обработчика
			handler := ShortenHandler("http://localhost:8080")
			handler(rr, req)

			// Проверка статуса
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

// TestShortenJSONHandler - тестирование функции shortenJSONHandler.
func TestShortenJSONHandler(t *testing.T) {

	tests := []struct {
		name           string
		contentType    string
		body           string
		filePath       string
		expectedStatus int
	}{
		{
			name:           "Valid Content-Type",
			contentType:    "application/json",
			body:           `{"url": "https://example.com"}`,
			filePath:       "testdata/urls.json",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Empty body",
			contentType:    "text/plain",
			body:           "",
			filePath:       "testdata/urls.json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid URL format",
			contentType:    "text/plain",
			body:           `{"url": "https://example.com"}`,
			filePath:       "testdata/urls.json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid request",
			contentType:    "text/plain",
			body:           `{"url": "https://example.com"}`,
			filePath:       "testdata/urls.json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Сброс глобального состояния
			urlMap = make(map[string]string)

			// Создание запроса
			req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)

			// Создание ответа
			rr := httptest.NewRecorder()

			// Вызов обработчика
			handler := ShortenJSONHandler(tt.body, tt.filePath)
			handler(rr, req)

			// Проверка статуса
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestSignAndVerifyUserID(t *testing.T) {
	testUserID := "test-user-id-123"

	// Подписываем userID
	signedCookie := signUserID(testUserID)

	if signedCookie == "" {
		t.Error("Signed cookie should not be empty")
	}

	// Проверяем, что подпись валидна
	verifiedUserID, valid := verifyCookie(signedCookie)
	if !valid {
		t.Error("Cookie signature should be valid")
	}

	if verifiedUserID != testUserID {
		t.Errorf("Expected userID '%s', got '%s'", testUserID, verifiedUserID)
	}
}

func TestVerifyCookie_InvalidFormat(t *testing.T) {
	// Тестируем неправильный формат куки
	testCases := []string{
		"",
		"invalid",
		"userid.without.dots",
		"userid.part1.part2.part3",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			userID, valid := verifyCookie(tc)
			if valid {
				t.Error("Invalid cookie format should not be valid")
			}
			if userID != "" {
				t.Error("UserID should be empty for invalid cookie")
			}
		})
	}
}

func TestAuthenticateUser_NewUser(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	userID := authenticateUser(w, req)

	if userID == "" {
		t.Error("UserID should not be empty")
	}

	// Проверяем, что кука установлена
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Error("Cookie should be set for new user")
	}

	var userCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "user_id" {
			userCookie = cookie
			break
		}
	}

	if userCookie == nil {
		t.Error("user_id cookie should be set")
	}

	// Проверяем, что кука валидна
	verifiedUserID, valid := verifyCookie(userCookie.Value)
	if !valid {
		t.Error("Set cookie should have valid signature")
	}
	if verifiedUserID != userID {
		t.Errorf("Cookie userID '%s' should match returned userID '%s'", verifiedUserID, userID)
	}
}

func TestAuthenticateUser_ExistingUser(t *testing.T) {
	// Сначала создаем пользователя
	firstReq := httptest.NewRequest("GET", "/", nil)
	firstW := httptest.NewRecorder()
	firstUserID := authenticateUser(firstW, firstReq)

	// Получаем установленную куку
	firstCookies := firstW.Result().Cookies()
	if len(firstCookies) == 0 {
		t.Fatal("First request should set cookie")
	}

	// Создаем второй запрос с той же кукой
	secondReq := httptest.NewRequest("GET", "/", nil)
	for _, cookie := range firstCookies {
		secondReq.AddCookie(cookie)
	}
	secondW := httptest.NewRecorder()

	secondUserID := authenticateUser(secondW, secondReq)

	// UserID должен совпадать
	if secondUserID != firstUserID {
		t.Errorf("Second request userID '%s' should match first request userID '%s'", secondUserID, firstUserID)
	}

	// Новых кук устанавливаться не должно (или они должны быть такими же)
	secondCookies := secondW.Result().Cookies()
	if len(secondCookies) > 0 {
		// Если кука устанавливается, она должна быть такой же
		for _, cookie := range secondCookies {
			if cookie.Name == "user_id" && cookie.Value != firstCookies[0].Value {
				t.Error("Cookie value should not change for existing user")
			}
		}
	}
}

func TestGetAuthenticatedUserID(t *testing.T) {
	// Тест без куки
	reqWithoutCookie := httptest.NewRequest("GET", "/", nil)
	userID, valid := getAuthenticatedUserID(reqWithoutCookie)
	if valid {
		t.Error("Request without cookie should not be valid")
	}
	if userID != "" {
		t.Error("UserID should be empty for request without cookie")
	}

	// Тест с валидной кукой
	testUserID := "test-user-123"
	validCookie := signUserID(testUserID)

	reqWithValidCookie := httptest.NewRequest("GET", "/", nil)
	reqWithValidCookie.AddCookie(&http.Cookie{
		Name:  "user_id",
		Value: validCookie,
	})

	userID, valid = getAuthenticatedUserID(reqWithValidCookie)
	if !valid {
		t.Error("Request with valid cookie should be valid")
	}
	if userID != testUserID {
		t.Errorf("Expected userID '%s', got '%s'", testUserID, userID)
	}

	// Тест с невалидной кукой
	reqWithInvalidCookie := httptest.NewRequest("GET", "/", nil)
	reqWithInvalidCookie.AddCookie(&http.Cookie{
		Name:  "user_id",
		Value: "invalid.cookie",
	})

	userID, valid = getAuthenticatedUserID(reqWithInvalidCookie)
	if valid {
		t.Error("Request with invalid cookie should not be valid")
	}
	if userID != "" {
		t.Error("UserID should be empty for invalid cookie")
	}
}
