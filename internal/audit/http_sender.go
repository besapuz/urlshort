package audit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

type HTTPSender struct {
	url    string
	client *http.Client
	ch     chan Event
	done   chan struct{}
}

func NewHTTPSender(url string) *HTTPSender {
	hs := &HTTPSender{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		ch:   make(chan Event, 100),
		done: make(chan struct{}),
	}

	go hs.processEvents()
	return hs
}

func (hs *HTTPSender) Notify(event Event) error {
	select {
	case hs.ch <- event:
		return nil
	case <-hs.done:
		return http.ErrHandlerTimeout
	}
}

func (hs *HTTPSender) processEvents() {
	for {
		select {
		case event := <-hs.ch:
			hs.sendToServer(event)
		case <-hs.done:
			return
		}
	}
}

func (hs *HTTPSender) sendToServer(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", hs.url, bytes.NewReader(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := hs.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Можно проверить статус ответа
	if resp.StatusCode >= 400 {
	}
}

func (hs *HTTPSender) Close() error {
	close(hs.done)
	return nil
}
