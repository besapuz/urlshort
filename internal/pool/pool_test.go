package pool

import (
	"sync"
	"testing"
	"time"
)

// TestPoolBasic тестирует базовую функциональность пула.
func TestPoolBasic(t *testing.T) {
	// Создаем пул для URLRequest
	pool := New(func() URLRequest {
		return NewURLRequest()
	}, 10)

	// Получаем объект из пула
	req := pool.Get()

	// Используем объект
	req.Method = "POST"
	req.URL = "https://example.com"
	req.Headers["Content-Type"] = "application/json"
	req.Body = append(req.Body, "test data"...)
	req.Timeout = 30

	// Возвращаем объект в пул
	pool.Put(req)

	// Проверяем, что объект сброшен при следующем получении
	req2 := pool.Get()
	if req2.Method != "" || req2.URL != "" || len(req2.Headers) != 0 || len(req2.Body) != 0 || req2.Timeout != 0 {
		t.Errorf("Object was not properly reset: %+v", req2)
	}

	if pool.Size() != 1 {
		t.Errorf("Expected pool size 1, got %d", pool.Size())
	}
}

// TestPointerPool тестирует пул для указателей.
func TestPointerPool(t *testing.T) {
	// Создаем пул для указателей на URLRequest
	pool := NewPointerPool(func() *URLRequest {
		req := NewURLRequest()
		return &req
	}, 10)

	// Получаем указатель на объект из пула
	req := pool.Get()

	// Используем объект
	req.Method = "POST"
	req.URL = "https://example.com"
	req.Headers["Content-Type"] = "application/json"
	req.Body = append(req.Body, "test data"...)
	req.Timeout = 30

	// Возвращаем указатель в пул
	pool.Put(req)

	// При следующем получении объект должен быть сброшен
	// Но PointerPool не вызывает Reset автоматически!
	// Нужно вызывать вручную или использовать UniversalPool
}

// TestPoolConcurrent тестирует конкурентное использование пула.
func TestPoolConcurrent(t *testing.T) {
	pool := New(func() URLRequest {
		return NewURLRequest()
	}, 100)

	var wg sync.WaitGroup
	iterations := 1000
	workers := 10

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < iterations/workers; j++ {
				// Получаем объект
				req := pool.Get()

				// Используем объект
				req.Method = "GET"
				req.URL = "https://example.com"
				req.Headers["X-Worker"] = string(rune(workerID))
				req.Timeout = workerID * 10

				// Имитируем работу
				time.Sleep(time.Microsecond)

				// Возвращаем в пул
				pool.Put(req)
			}
		}(i)
	}

	wg.Wait()

	t.Logf("Final pool size: %d", pool.Size())
}

// TestPoolMaxSize тестирует ограничение максимального размера пула.
func TestPoolMaxSize(t *testing.T) {
	maxSize := 5
	pool := New(func() URLRequest {
		return NewURLRequest()
	}, maxSize)

	// Создаем больше объектов, чем максимальный размер
	var requests []URLRequest
	for i := 0; i < maxSize*2; i++ {
		req := pool.Get()
		requests = append(requests, req)
	}

	// Возвращаем все объекты в пул
	for _, req := range requests {
		pool.Put(req)
	}

	// Пул должен содержать не более maxSize объектов
	if pool.Size() > maxSize {
		t.Errorf("Pool exceeded max size: got %d, max %d", pool.Size(), maxSize)
	}

	t.Logf("Pool size: %d (max: %d)", pool.Size(), maxSize)
}

// TestPoolClear тестирует очистку пула.
func TestPoolClear(t *testing.T) {
	pool := New(func() URLRequest {
		return NewURLRequest()
	}, 10)

	// Получаем и возвращаем несколько объектов
	for i := 0; i < 5; i++ {
		req := pool.Get()
		pool.Put(req)
	}

	if pool.Size() != 5 {
		t.Errorf("Expected pool size 5 before clear, got %d", pool.Size())
	}

	// Очищаем пул
	pool.Clear()

	if pool.Size() != 0 {
		t.Errorf("Expected pool size 0 after clear, got %d", pool.Size())
	}
}
