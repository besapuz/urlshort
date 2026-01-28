package router

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type URLMapping struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
	DeletedFlag bool   `json:"is_deleted"`
}

type Storages struct {
	urlMap      map[string]string
	userURLsMap map[string][]string
	URLMappings []URLMapping
	mutex       sync.Mutex
	storageFile string // Глобальная переменная для пути к файлу
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

func (s *Storages) LoadFromFile(filePath string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		// Файл не существует — инициализируем пустую карту
		s.urlMap = make(map[string]string)
		s.URLMappings = []URLMapping{}
		return nil
	}
	if err != nil {
		return fmt.Errorf("ошибка чтения файла: %w", err)
	}

	if len(data) == 0 {
		// Пустой файл — инициализируем пустую карту
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
