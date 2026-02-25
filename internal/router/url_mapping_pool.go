package router

import (
	"github.com/besapuz/urlshort/internal/pool"
)

// URLMapping - структура, которая реализует Reset()
type URLMapping struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
	DeletedFlag bool   `json:"is_deleted"`
}

// Reset сбрасывает состояние URLMapping
func (m *URLMapping) Reset() {
	m.UUID = ""
	m.ShortURL = ""
	m.OriginalURL = ""
	m.UserID = ""
	m.DeletedFlag = false
}

// NewURLMapping создает новый URLMapping
func NewURLMapping() URLMapping {
	return URLMapping{}
}

// Создаем пул для URLMapping (не URLMappingWithReset)
var urlMappingPool = pool.New(NewURLMapping, 100)

// GetURLMappingFromPool возвращает URLMapping из пула
func GetURLMappingFromPool() URLMapping {
	return urlMappingPool.Get()
}

// PutURLMappingToPool возвращает URLMapping в пул
func PutURLMappingToPool(mapping URLMapping) {
	urlMappingPool.Put(mapping)
}
