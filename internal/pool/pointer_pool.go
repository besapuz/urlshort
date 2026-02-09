package pool

import (
	"sync"
)

// PointerPool представляет собой пул для указателей на объекты с поддержкой Reset.
type PointerPool[T any] struct {
	pool    *sync.Pool
	mu      sync.RWMutex
	size    int
	maxSize int
}

// NewPointerPool создает новый пул для указателей на тип T.
func NewPointerPool[T any](newFunc func() *T, maxSize int) *PointerPool[T] {
	p := &PointerPool[T]{
		pool: &sync.Pool{
			New: func() any {
				return newFunc()
			},
		},
		maxSize: maxSize,
	}

	return p
}

// Get возвращает указатель на объект из пула.
func (p *PointerPool[T]) Get() *T {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.pool.Get().(*T)
}

// Put помещает указатель на объект обратно в пул.
func (p *PointerPool[T]) Put(x *T) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Проверяем ограничение по размеру
	if p.maxSize > 0 && p.size >= p.maxSize {
		return
	}

	p.pool.Put(x)
	p.size++
}

// Size возвращает текущее количество объектов в пуле.
func (p *PointerPool[T]) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.size
}
