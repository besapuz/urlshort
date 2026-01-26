package audit

import (
	"log"
	"sync"
)

type Manager struct {
	observers []Observer
	mu        sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		observers: make([]Observer, 0),
	}
}

func (m *Manager) Register(observer Observer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.observers = append(m.observers, observer)
}

func (m *Manager) NotifyAll(event Event) {
	m.mu.RLock()
	observers := make([]Observer, len(m.observers))
	copy(observers, m.observers)
	m.mu.RUnlock()

	for _, observer := range observers {
		go func(o Observer) {
			if err := o.Notify(event); err != nil {
				log.Printf("Ошибка уведомления: %v", err)
			}
		}(observer)
	}
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for _, observer := range m.observers {
		if err := observer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	m.observers = nil

	if len(errs) > 0 {
		return errs[0] // Возвращаем первую ошибку
	}
	return nil
}
