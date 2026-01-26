package audit

import (
	"encoding/json"
	"os"
	"sync"
)

type FileWriter struct {
	filePath string
	file     *os.File
	mu       sync.Mutex
	ch       chan Event
	done     chan struct{}
}

func NewFileWriter(filePath string) (*FileWriter, error) {
	fw := &FileWriter{
		filePath: filePath,
		ch:       make(chan Event, 100), // Буферизованный канал
		done:     make(chan struct{}),
	}

	// Открываем файл для добавления
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	fw.file = file

	// Запускаем горутину для обработки событий
	go fw.processEvents()

	return fw, nil
}

func (fw *FileWriter) Notify(event Event) error {
	select {
	case fw.ch <- event:
		return nil
	case <-fw.done:
		return os.ErrClosed
	}
}

func (fw *FileWriter) processEvents() {
	for {
		select {
		case event := <-fw.ch:
			fw.writeToFile(event)
		case <-fw.done:
			return
		}
	}
}

func (fw *FileWriter) writeToFile(event Event) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	if _, err := fw.file.Write(append(data, '\n')); err != nil {
		// Логируем ошибку
		return
	}

	// Синхронизируем файл на диск
	fw.file.Sync()
}

func (fw *FileWriter) Close() error {
	close(fw.done)
	fw.mu.Lock()
	defer fw.mu.Unlock()
	return fw.file.Close()
}
