package router

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Storages struct {
	urlMap      map[string]string
	userURLsMap map[string][]string
	URLMappings []URLMapping
	mutex       sync.Mutex
	storageFile string
}

// SetStorageFile устанавливает путь к файлу хранилища
func (s *Storages) SetStorageFile(filePath string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.storageFile = filePath
}

// GetStorageFilePath возвращает путь к файлу хранилища
func (s *Storages) GetStorageFilePath() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.storageFile
}

// MutexLock блокирует мьютекс для внешнего использования
func (s *Storages) MutexLock() {
	s.mutex.Lock()
}

// MutexUnlock разблокирует мьютекс для внешнего использования
func (s *Storages) MutexUnlock() {
	s.mutex.Unlock()
}

// ==================== ПУБЛИЧНЫЕ МЕТОДЫ ДЛЯ ДОСТУПА К ДАННЫМ ====================

// GetURL возвращает оригинальный URL по короткому ID
func (s *Storages) GetURL(shortID string) (string, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	url, exists := s.urlMap[shortID]
	return url, exists
}

// SaveURL сохраняет URL в память
func (s *Storages) SaveURL(shortID, originalURL, userID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.urlMap[shortID] = originalURL

	if userURLs, exists := s.userURLsMap[userID]; exists {
		s.userURLsMap[userID] = append(userURLs, shortID)
	} else {
		s.userURLsMap[userID] = []string{shortID}
	}
}

// SaveURLWithMapping сохраняет URL с полным маппингом (для файлового хранилища)
func (s *Storages) SaveURLWithMapping(mapping URLMapping) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.URLMappings = append(s.URLMappings, mapping)
	s.urlMap[mapping.ShortURL] = mapping.OriginalURL
}

// GetUserURLs возвращает все короткие ID пользователя
func (s *Storages) GetUserURLs(userID string) []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	urls, exists := s.userURLsMap[userID]
	if !exists {
		return []string{}
	}
	// Возвращаем копию для безопасности
	result := make([]string, len(urls))
	copy(result, urls)
	return result
}

// GetURLMappings возвращает все маппинги (для файлового хранилища)
func (s *Storages) GetURLMappings() []URLMapping {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Возвращаем копию для безопасности
	result := make([]URLMapping, len(s.URLMappings))
	copy(result, s.URLMappings)
	return result
}

// DeleteURL помечает URL как удаленный (soft delete)
func (s *Storages) DeleteURL(shortID, userID string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Ищем в URLMappings для файлового хранилища
	for i := range s.URLMappings {
		if s.URLMappings[i].ShortURL == shortID && s.URLMappings[i].UserID == userID {
			s.URLMappings[i].DeletedFlag = true
			return true
		}
	}

	// Для in-memory хранилища - удаляем из urlMap
	if _, exists := s.urlMap[shortID]; exists {
		delete(s.urlMap, shortID)
		// И из userURLsMap
		if userURLs, exists := s.userURLsMap[userID]; exists {
			for i, id := range userURLs {
				if id == shortID {
					s.userURLsMap[userID] = append(userURLs[:i], userURLs[i+1:]...)
					break
				}
			}
		}
		return true
	}

	return false
}

// LoadFromFile загружает данные из файла
func (s *Storages) LoadFromFile(filePath string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		s.urlMap = make(map[string]string)
		s.URLMappings = []URLMapping{}
		return nil
	}
	if err != nil {
		return fmt.Errorf("ошибка чтения файла: %w", err)
	}

	if len(data) == 0 {
		s.urlMap = make(map[string]string)
		s.URLMappings = []URLMapping{}
		return nil
	}

	if err := json.Unmarshal(data, &s.URLMappings); err != nil {
		return fmt.Errorf("ошибка десериализации JSON: %w", err)
	}

	// Синхронизируем urlMap с URLMappings
	s.urlMap = make(map[string]string)
	for _, m := range s.URLMappings {
		s.urlMap[m.ShortURL] = m.OriginalURL
	}
	return nil
}

// SaveToFile сохраняет данные в файл
func (s *Storages) SaveToFile(filePath string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.Marshal(s.URLMappings)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return err
	}
	return nil
}

// NewStorages создает новый инициализированный экземпляр Storages
func NewStorages() *Storages {
	return &Storages{
		urlMap:      make(map[string]string),
		userURLsMap: make(map[string][]string),
		URLMappings: []URLMapping{},
		mutex:       sync.Mutex{},
	}
}
