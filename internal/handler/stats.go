package handler

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/besapuz/urlshort/internal/router"
)

// StatsHandler возвращает обработчик для получения статистики
func StatsHandler(shortener *router.URLShortener, trustedSubnet string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что метод GET
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Проверяем доверенную подсеть
		if !isIPTrusted(r, trustedSubnet) {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		// Получаем статистику
		stats := shortener.GetStats()

		// Отправляем ответ
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(stats)
	}
}

// isIPTrusted проверяет, входит ли IP клиента в доверенную подсеть
func isIPTrusted(r *http.Request, trustedSubnet string) bool {
	// Если подсеть не задана, доступ запрещён
	if trustedSubnet == "" {
		return false
	}

	// Получаем IP из заголовка X-Real-IP
	ipStr := r.Header.Get("X-Real-IP")
	if ipStr == "" {
		return false
	}

	// Парсим IP клиента
	clientIP := net.ParseIP(ipStr)
	if clientIP == nil {
		return false
	}

	// Парсим доверенную подсеть
	_, subnet, err := net.ParseCIDR(trustedSubnet)
	if err != nil {
		return false
	}

	// Проверяем, входит ли IP в подсеть
	return subnet.Contains(clientIP)
}
