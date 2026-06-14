package events

import (
	"encoding/json"
	"sync"
)

type Event struct {
	Name string
	Data []byte
}

type Hub struct {
	mu          sync.Mutex
	nextID      int
	subscribers map[int]chan Event
}

func NewHub() *Hub {
	return &Hub{subscribers: map[int]chan Event{}}
}

func (h *Hub) Subscribe() (int, <-chan Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	id := h.nextID
	ch := make(chan Event, 32)
	h.subscribers[id] = ch
	return id, ch
}

func (h *Hub) Unsubscribe(id int) {
	h.mu.Lock()
	ch, ok := h.subscribers[id]
	if ok {
		delete(h.subscribers, id)
		close(ch)
	}
	h.mu.Unlock()
}

func (h *Hub) Publish(name string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	event := Event{Name: name, Data: data}

	h.mu.Lock()
	defer h.mu.Unlock()
	for _, ch := range h.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}
