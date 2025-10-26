package router

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var file *os.File

type URLMapping struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

var (
	URLMappings []URLMapping
	mutex       sync.Mutex
)

func LoadFromFile(filePath string) error {
	mutex.Lock()
	defer mutex.Unlock()
	data, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &URLMappings); err != nil {
		return err
	}
	for _, v := range URLMappings {
		urlMap[v.ShortURL] = v.OriginalURL
	}
	return nil
}

func SaveToFile(filePath string) error {
	var err error
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
	file, err = os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return err
	}
	return nil
}
