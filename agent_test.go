package agent

import (
	"context"
	"errors"
	"testing"
)

type agentTestLLM struct{}

func (agentTestLLM) Chat(context.Context, *ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{Content: "ok"}, nil
}

type agentTestTool struct {
	name string
}

func (t agentTestTool) Name() string        { return t.name }
func (t agentTestTool) Description() string { return "test tool" }
func (t agentTestTool) Call(context.Context, string) (string, error) {
	return "ok", nil
}

type agentTestToolManager struct {
	tools map[string]Tool
}

func newAgentTestToolManager() *agentTestToolManager {
	return &agentTestToolManager{tools: map[string]Tool{}}
}

func (m *agentTestToolManager) Register(t Tool) error {
	if t == nil {
		return errors.New("nil tool")
	}
	m.tools[t.Name()] = t
	return nil
}

func (m *agentTestToolManager) Get(name string) (Tool, bool) {
	t, ok := m.tools[name]
	return t, ok
}

func (m *agentTestToolManager) List() []Tool {
	out := make([]Tool, 0, len(m.tools))
	for _, t := range m.tools {
		out = append(out, t)
	}
	return out
}

type agentTestMemory struct{}

func (agentTestMemory) Build(context.Context, string, string) ([]Message, error) {
	return nil, nil
}

func (agentTestMemory) Persist(context.Context, string, []Message) error {
	return nil
}

func TestNewAgent_Defaults(t *testing.T) {
	a := NewAgent(agentTestLLM{})
	if a == nil {
		t.Fatal("expected non-nil agent")
	}
	if a.router == nil {
		t.Fatal("expected default router")
	}
	if a.tools == nil {
		t.Fatal("expected default tool manager")
	}
	if a.toolRouter == nil {
		t.Fatal("expected tool router to be initialized")
	}
	if a.cfg.MaxSteps != DefaultConfig().MaxSteps {
		t.Fatalf("unexpected default config: %#v", a.cfg)
	}
}

func TestAgentRegisterTool_WithoutToolManager(t *testing.T) {
	a := NewAgent(agentTestLLM{})
	err := a.RegisterTool(agentTestTool{name: "time"})
	if err == nil {
		t.Fatal("expected error without external tool manager")
	}
}

func TestWithToolManager_EnablesRegister(t *testing.T) {
	mgr := newAgentTestToolManager()
	a := NewAgent(agentTestLLM{}, WithToolManager(mgr))
	if err := a.RegisterTool(agentTestTool{name: "time"}); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}
	if _, ok := mgr.Get("time"); !ok {
		t.Fatal("expected tool registered in custom manager")
	}
}

func TestWithConfig_UpdatesConfigAndToolRouter(t *testing.T) {
	a := NewAgent(agentTestLLM{})
	oldRouter := a.toolRouter

	cfg := DefaultConfig()
	cfg.MaxSteps = 9
	cfg.ToolDecision.EnableRouter = false

	WithConfig(cfg)(a)

	if a.cfg.MaxSteps != 9 {
		t.Fatalf("expected cfg.MaxSteps=9, got %d", a.cfg.MaxSteps)
	}
	if a.toolRouter == nil {
		t.Fatal("expected toolRouter to be initialized")
	}
	if a.toolRouter == oldRouter {
		t.Fatal("expected toolRouter to be rebuilt after WithConfig")
	}
}

func TestWithRouter_AndMemoryOptions(t *testing.T) {
	a := NewAgent(agentTestLLM{})
	originalRouter := a.router

	WithRouter(nil)(a)
	if a.router != originalRouter {
		t.Fatal("expected nil router option to be ignored")
	}

	r := NewModelRouter()
	WithRouter(r)(a)
	if a.router != r {
		t.Fatal("expected router to be replaced")
	}

	mem1 := agentTestMemory{}
	WithMemory(mem1)(a)
	if a.mem == nil {
		t.Fatal("expected memory set by WithMemory")
	}

	mem2 := agentTestMemory{}
	WithMemory(mem2)(a)
	if a.mem == nil {
		t.Fatal("expected memory set by WithMemoryManager")
	}
}
