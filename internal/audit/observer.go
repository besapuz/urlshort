package audit

// Observer представляет интерфейс наблюдателя в паттерне Наблюдатель.
type Observer interface {
	// Notify отправляет событие наблюдателю.
	Notify(event Event) error

	// Close закрывает ресурсы наблюдателя.
	Close() error
}

// Subject представляет интерфейс субъекта в паттерне Наблюдатель.
type Subject interface {
	// Register регистрирует нового наблюдателя.
	Register(observer Observer)

	// Unregister удаляет наблюдателя.
	Unregister(observer Observer)

	// NotifyAll уведомляет всех наблюдателей о событии.
	NotifyAll(event Event)
}
