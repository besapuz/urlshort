package audit

import "time"

// Action представляет тип аудит-события.
type Action string

const (
	// ActionShorten - событие сокращения URL.
	ActionShorten Action = "shorten"

	// ActionFollow - событие перехода по сокращенной ссылке.
	ActionFollow Action = "follow"
)

// Event представляет аудит-событие.
type Event struct {
	// Timestamp - временная метка события в формате Unix timestamp.
	Timestamp int64 `json:"ts"`

	// Action - тип события.
	Action Action `json:"action"`

	// UserID - идентификатор пользователя, совершившего действие.
	UserID string `json:"user_id"`

	// URL - URL, над которым выполнено действие.
	URL string `json:"url"`
}

// NewEvent создает новое аудит-событие.
// Автоматически устанавливает текущую временную метку.
func NewEvent(action Action, userID, url string) Event {
	return Event{
		Timestamp: time.Now().Unix(),
		Action:    action,
		UserID:    userID,
		URL:       url,
	}
}
