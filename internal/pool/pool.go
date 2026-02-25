// Package pool предоставляет типизированный пул объектов с поддержкой Reset.
package pool

import (
	"sync"
)

// Resetable определяет интерфейс для типов, которые могут сбрасывать свое состояние.
// Используем указатель в интерфейсе, чтобы работать с методами, имеющими pointer receiver.
type Resetable interface {
	Reset()
}

// Pool представляет собой типизированный пул объектов с поддержкой Reset.
// T должен быть типом, который реализует метод Reset() через pointer receiver.
type Pool[T any] struct {
	pool    *sync.Pool
	mu      sync.RWMutex
	size    int
	maxSize int
}

// New создает новый пул для типа T.
// T должен быть типом, у которого есть метод Reset() с pointer receiver.
func New[T any](newFunc func() T, maxSize int) *Pool[T] {
	p := &Pool[T]{
		pool: &sync.Pool{
			New: func() any {
				return newFunc()
			},
		},
		maxSize: maxSize,
	}

	return p
}

// Get возвращает объект из пула.
func (p *Pool[T]) Get() T {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.pool.Get().(T)
}

// Put помещает объект обратно в пул.
// Используем type assertion для вызова Reset().
func (p *Pool[T]) Put(x T) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Пытаемся вызвать Reset() через interface{}
	if resetter, ok := any(&x).(interface{ Reset() }); ok {
		resetter.Reset()
	}

	// Проверяем ограничение по размеру
	if p.maxSize > 0 && p.size >= p.maxSize {
		return
	}

	p.pool.Put(x)
	p.size++
}

// Clear очищает пул.
func (p *Pool[T]) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.pool = &sync.Pool{
		New: p.pool.New,
	}
	p.size = 0
}

// Size возвращает текущий размер пула.
func (p *Pool[T]) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.size
}

// MaxSize возвращает максимальный размер пула.
func (p *Pool[T]) MaxSize() int {
	return p.maxSize
}
