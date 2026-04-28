package agent

import (
	"context"
)

// AgentMemory is the unified memory abstraction used by Agent runtime.
//
// Build constructs the messages used as the next turn context.
// Persist writes back conversation updates.
type AgentMemory interface {
	Build(ctx context.Context, sessionID string, userInput string) ([]Message, error)
	Persist(ctx context.Context, sessionID string, msgs []Message) error
}

// MemoryManager is kept as a backward-compatible alias.
type MemoryManager = AgentMemory
