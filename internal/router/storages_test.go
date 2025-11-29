package router

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSetAndGetStorageFile(t *testing.T) {
	// Сохраняем оригинальное значение
	originalStorageFile := storageFile

	// Восстанавливаем после теста
	defer func() {
		storageFile = originalStorageFile
	}()

	testPath := "/test/path/storage.json"
	SetStorageFile(testPath)

	result := GetStorageFilePath()
	if result != testPath {
		t.Errorf("Expected storage path %s, got %s", testPath, result)
	}
}

func TestLoadFromFile_FileNotExists(t *testing.T) {
	// Сохраняем оригинальные данные
	originalURLMap := urlMap
	originalMappings := URLMappings

	// Восстанавливаем после теста
	defer func() {
		urlMap = originalURLMap
		URLMappings = originalMappings
	}()

	nonExistentFile := "/tmp/nonexistent_file_12345.json"
	err := LoadFromFile(nonExistentFile)

	if err != nil {
		t.Errorf("Expected no error for non-existent file, got %v", err)
	}

	if len(urlMap) != 0 {
		t.Errorf("Expected empty urlMap, got %d elements", len(urlMap))
	}

	if len(URLMappings) != 0 {
		t.Errorf("Expected empty URLMappings, got %d elements", len(URLMappings))
	}
}

func TestLoadFromFile_EmptyFile(t *testing.T) {
	// Сохраняем оригинальные данные
	originalURLMap := urlMap
	originalMappings := URLMappings

	// Восстанавливаем после теста
	defer func() {
		urlMap = originalURLMap
		URLMappings = originalMappings
	}()

	// Создаем временный пустой файл
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.json")

	file, err := os.Create(emptyFile)
	if err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}
	file.Close()

	err = LoadFromFile(emptyFile)
	if err != nil {
		t.Errorf("Expected no error for empty file, got %v", err)
	}

	if len(urlMap) != 0 {
		t.Errorf("Expected empty urlMap, got %d elements", len(urlMap))
	}

	if len(URLMappings) != 0 {
		t.Errorf("Expected empty URLMappings, got %d elements", len(URLMappings))
	}
}

func TestLoadFromFile_ValidData(t *testing.T) {
	// Сохраняем оригинальные данные
	originalURLMap := urlMap
	originalMappings := URLMappings

	// Восстанавливаем после теста
	defer func() {
		urlMap = originalURLMap
		URLMappings = originalMappings
	}()

	// Создаем временный файл с валидными данными
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_data.json")

	testMappings := []URLMapping{
		{
			UUID:        "123",
			ShortURL:    "abc123",
			OriginalURL: "https://example.com",
			UserID:      "user1",
			DeletedFlag: false,
		},
		{
			UUID:        "456",
			ShortURL:    "def456",
			OriginalURL: "https://google.com",
			UserID:      "user2",
			DeletedFlag: true,
		},
	}

	data, err := json.Marshal(testMappings)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	err = os.WriteFile(testFile, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err = LoadFromFile(testFile)
	if err != nil {
		t.Errorf("Expected no error for valid file, got %v", err)
	}

	// Проверяем синхронизацию urlMap
	if len(urlMap) != 2 {
		t.Errorf("Expected urlMap with 2 elements, got %d", len(urlMap))
	}

	if urlMap["abc123"] != "https://example.com" {
		t.Errorf("Expected urlMap['abc123'] = 'https://example.com', got '%s'", urlMap["abc123"])
	}

	if urlMap["def456"] != "https://google.com" {
		t.Errorf("Expected urlMap['def456'] = 'https://google.com', got '%s'", urlMap["def456"])
	}

	// Проверяем URLMappings
	if len(URLMappings) != 2 {
		t.Errorf("Expected URLMappings with 2 elements, got %d", len(URLMappings))
	}
}

func TestLoadFromFile_InvalidJSON(t *testing.T) {
	// Сохраняем оригинальные данные
	originalURLMap := urlMap
	originalMappings := URLMappings

	// Восстанавливаем после теста
	defer func() {
		urlMap = originalURLMap
		URLMappings = originalMappings
	}()

	// Создаем временный файл с невалидным JSON
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.json")

	err := os.WriteFile(invalidFile, []byte("{invalid json}"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid JSON file: %v", err)
	}

	err = LoadFromFile(invalidFile)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestSaveToFile(t *testing.T) {
	// Сохраняем оригинальные данные
	originalURLMap := urlMap
	originalMappings := URLMappings

	// Восстанавливаем после теста
	defer func() {
		urlMap = originalURLMap
		URLMappings = originalMappings
	}()

	// Устанавливаем тестовые данные
	URLMappings = []URLMapping{
		{
			UUID:        "test-uuid-1",
			ShortURL:    "short1",
			OriginalURL: "https://test1.com",
			UserID:      "test-user-1",
			DeletedFlag: false,
		},
		{
			UUID:        "test-uuid-2",
			ShortURL:    "short2",
			OriginalURL: "https://test2.com",
			UserID:      "test-user-2",
			DeletedFlag: true,
		},
	}

	// Создаем временную директорию для файла
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "save_test.json")

	// Сохраняем в файл
	err := SaveToFile(testFile)
	if err != nil {
		t.Errorf("SaveToFile failed: %v", err)
	}

	// Проверяем, что файл создан
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("File was not created")
	}

	// Читаем и проверяем содержимое файла
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("Failed to read saved file: %v", err)
	}

	var loadedMappings []URLMapping
	err = json.Unmarshal(data, &loadedMappings)
	if err != nil {
		t.Errorf("Failed to unmarshal saved data: %v", err)
	}

	if len(loadedMappings) != 2 {
		t.Errorf("Expected 2 mappings in file, got %d", len(loadedMappings))
	}

	if loadedMappings[0].ShortURL != "short1" {
		t.Errorf("Expected first mapping shortURL 'short1', got '%s'", loadedMappings[0].ShortURL)
	}

	if loadedMappings[1].DeletedFlag != true {
		t.Error("Expected second mapping to have DeletedFlag = true")
	}
}

func TestSaveToFile_CreateDirectory(t *testing.T) {
	// Сохраняем оригинальные данные
	originalMappings := URLMappings
	defer func() {
		URLMappings = originalMappings
	}()

	URLMappings = []URLMapping{
		{
			UUID:        "test-uuid",
			ShortURL:    "test-short",
			OriginalURL: "https://test.com",
			UserID:      "test-user",
			DeletedFlag: false,
		},
	}

	// Пытаемся сохранить в несуществующую директорию
	tmpDir := t.TempDir()
	nestedFile := filepath.Join(tmpDir, "nonexistent", "dir", "test.json")

	err := SaveToFile(nestedFile)
	if err != nil {
		t.Errorf("SaveToFile should create directories, but got error: %v", err)
	}

	// Проверяем, что файл создан
	if _, err := os.Stat(nestedFile); os.IsNotExist(err) {
		t.Error("File was not created in nested directory")
	}
}
