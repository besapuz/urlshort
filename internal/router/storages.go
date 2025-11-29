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
}

var (
	urlMap      = make(map[string]string)
	userURLsMap = make(map[string][]string)
	URLMappings []URLMapping
	mutex       sync.Mutex
	storageFile string // Глобальная переменная для пути к файлу
)

// SetStorageFile устанавливает путь к файлу хранилища
func SetStorageFile(filePath string) {
	mutex.Lock()
	defer mutex.Unlock()
	storageFile = filePath
}

// GetStorageFilePath возвращает путь к файлу хранилища
func GetStorageFilePath() string {
	mutex.Lock()
	defer mutex.Unlock()
	return storageFile
}

func LoadFromFile(filePath string) error {
	mutex.Lock()
	defer mutex.Unlock()

	data, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		// Файл не существует — инициализируем пустую карту
		urlMap = make(map[string]string)
		URLMappings = []URLMapping{}
		return nil
	}
	if err != nil {
		return fmt.Errorf("ошибка чтения файла: %w", err)
	}

	if len(data) == 0 {
		// Пустой файл — инициализируем пустую карту
		urlMap = make(map[string]string)
		URLMappings = []URLMapping{}
		return nil
	}

	if err := json.Unmarshal(data, &URLMappings); err != nil {
		return fmt.Errorf("ошибка десериализации JSON: %w", err)
	}

	// Синхронизируем urlMap с URLMappings
	urlMap = make(map[string]string)
	for _, m := range URLMappings {
		urlMap[m.ShortURL] = m.OriginalURL
	}
	return nil
}

func SaveToFile(filePath string) error {
	mutex.Lock()
	defer mutex.Unlock()
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	data, err := json.Marshal(URLMappings)
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
