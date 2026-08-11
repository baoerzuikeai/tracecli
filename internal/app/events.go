package app

import (
	"sync"
	"time"
)

type EventType string

const (
	EventStateChanged EventType = "state_changed"
)

type Event struct {
	Type     EventType
	Previous State
	State    State
	Err      error
	Attempt  int
	At       time.Time
}

type EventBus struct {
	mu          sync.RWMutex
	subscribers map[uint64]chan Event
	nextID      uint64
	buffer      int
	closed      bool
}

func NewEventBus() *EventBus {
	return NewEventBusWithBuffer(32)
}

func NewEventBusWithBuffer(buffer int) *EventBus {
	if buffer <= 0 {
		buffer = 1
	}
	return &EventBus{
		subscribers: make(map[uint64]chan Event),
		buffer:      buffer,
	}
}

func (b *EventBus) Subscribe() <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.subscribers == nil {
		b.subscribers = make(map[uint64]chan Event)
	}
	if b.buffer <= 0 {
		b.buffer = 1
	}

	channel := make(chan Event, b.buffer)
	if b.closed {
		close(channel)
		return channel
	}

	b.nextID++
	b.subscribers[b.nextID] = channel
	return channel
}

func (b *EventBus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return
	}
	for _, channel := range b.subscribers {
		select {
		case channel <- event:
		default:
		}
	}
}

func (b *EventBus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}
	b.closed = true
	for id, channel := range b.subscribers {
		close(channel)
		delete(b.subscribers, id)
	}
}
