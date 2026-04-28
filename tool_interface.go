package agent

import (
	"context"
	"errors"
)

// Tool is the unified abstraction used by agent runtime.
type Tool interface {
	Name() string
	Description() string
	Call(ctx context.Context, input string) (string, error)
}

// ToolManager abstracts tool registration and discovery.
// Concrete implementations are provided externally (e.g. github.com/kordar/llm-tool).
type ToolManager interface {
	Register(t Tool) error
	Get(name string) (Tool, bool)
	List() []Tool
}

type noopToolManager struct{}

func (noopToolManager) Register(_ Tool) error {
	return errors.New("agent: no tool manager configured, use WithToolManager")
}
func (noopToolManager) Get(_ string) (Tool, bool) { return nil, false }
func (noopToolManager) List() []Tool              { return nil }
