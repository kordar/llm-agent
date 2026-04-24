package agent

import (
	"sync"
)

type Memory interface {
	Load(sessionID string) []Message
	Save(sessionID string, msgs []Message)
}

type InMemory struct {
	mu   sync.RWMutex
	data map[string][]Message
}

func NewMemory() *InMemory {
	return &InMemory{
		data: make(map[string][]Message),
	}
}

func (m *InMemory) Load(sessionID string) []Message {
	m.mu.RLock()
	msgs := m.data[sessionID]
	m.mu.RUnlock()

	out := make([]Message, len(msgs))
	copy(out, msgs)
	return out
}

func (m *InMemory) Save(sessionID string, msgs []Message) {
	out := make([]Message, len(msgs))
	copy(out, msgs)

	m.mu.Lock()
	m.data[sessionID] = out
	m.mu.Unlock()
}

