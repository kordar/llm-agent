package agent

import (
	"context"
	"testing"
)

type fakeTool struct {
	name string
	desc string
}

func (f fakeTool) Name() string        { return f.name }
func (f fakeTool) Description() string { return f.desc }
func (f fakeTool) Call(_ context.Context, _ string) (string, error) {
	return "ok", nil
}

type fakeLLM struct {
	content string
}

func (f fakeLLM) Chat(_ context.Context, _ *ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{Content: f.content}, nil
}

type fakeToolManager struct {
	items map[string]Tool
}

func newFakeToolManager() *fakeToolManager {
	return &fakeToolManager{items: make(map[string]Tool)}
}

func (m *fakeToolManager) Register(t Tool) error {
	m.items[t.Name()] = t
	return nil
}

func (m *fakeToolManager) Get(name string) (Tool, bool) {
	t, ok := m.items[name]
	return t, ok
}

func (m *fakeToolManager) List() []Tool {
	out := make([]Tool, 0, len(m.items))
	for _, t := range m.items {
		out = append(out, t)
	}
	return out
}

func TestToolRouter_RuleFirst(t *testing.T) {
	reg := newFakeToolManager()
	_ = reg.Register(fakeTool{name: "time", desc: "获取当前时间"})
	r := NewToolRouter(fakeLLM{content: `{"use_tool":false,"confidence":0.9}`}, reg, DefaultConfig().ToolDecision)

	dec := r.Decide(context.Background(), "qwen", "现在北京时间是多少", nil)
	if !dec.UseTool {
		t.Fatalf("expected tool decision")
	}
}

func TestToolRouter_MemoryHitPreferDirect(t *testing.T) {
	reg := newFakeToolManager()
	_ = reg.Register(fakeTool{name: "time", desc: "获取当前时间"})
	r := NewToolRouter(fakeLLM{content: `{"use_tool":true,"tool":"time","confidence":0.9}`}, reg, DefaultConfig().ToolDecision)

	history := []Message{
		{Role: "assistant", Content: "报销流程是提交申请后审批打款"},
	}
	dec := r.Decide(context.Background(), "qwen", "报销流程是什么", history)
	if dec.UseTool {
		t.Fatalf("expected direct answer due to memory hit")
	}
}
