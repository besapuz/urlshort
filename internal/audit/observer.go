package audit

type Observer interface {
	Notify(event Event) error
	Close() error
}

type Subject interface {
	Register(observer Observer)
	Unregister(observer Observer)
	NotifyAll(event Event)
}
