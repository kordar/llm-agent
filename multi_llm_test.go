package agent

import (
	"context"
	"errors"
	"testing"
)

type stubLLM struct {
	content string
	err     error
}

func (s stubLLM) Chat(context.Context, *ChatRequest) (*ChatResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &ChatResponse{Content: s.content}, nil
}

func TestMultiLLM_ExactAndFallback(t *testing.T) {
	m := NewMultiLLM()
	m.SetModelClient("model-a", stubLLM{err: errors.New("a failed")})
	m.SetDefaultClient(stubLLM{content: "from-default"})

	resp, err := m.Chat(context.Background(), &ChatRequest{Model: "model-a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || resp.Content != "from-default" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestMultiLLM_PrefixLongestFirst(t *testing.T) {
	m := NewMultiLLM()
	m.SetPrefixClient("qwen", stubLLM{content: "short-prefix"})
	m.SetPrefixClient("qwen2.5", stubLLM{content: "long-prefix"})

	resp, err := m.Chat(context.Background(), &ChatRequest{Model: "qwen2.5:7b"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || resp.Content != "long-prefix" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestMultiLLM_FallbackChain(t *testing.T) {
	m := NewMultiLLM()
	m.SetModelClient("model-a", stubLLM{err: errors.New("model failed")})
	m.SetFallbackChain(stubLLM{err: errors.New("fb1 failed")}, stubLLM{content: "fb2 ok"})

	resp, err := m.Chat(context.Background(), &ChatRequest{Model: "model-a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || resp.Content != "fb2 ok" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

