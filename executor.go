package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

func (a *Agent) Run(ctx context.Context, sessionID string, input string, level ModelLevel) (string, error) {
	if a == nil {
		return "", errors.New("agent: nil agent")
	}
	if a.llm == nil {
		return "", errors.New("agent: nil llm")
	}
	if a.memory == nil {
		return "", errors.New("agent: nil memory")
	}

	model := a.router.Get(level)
	if model == "" {
		return "", errors.New("agent: no model found for level")
	}

	msgs := a.memory.Load(sessionID)
	if a.cfg.SystemPrompt != "" && len(msgs) == 0 {
		msgs = append(msgs, Message{
			Role:    "system",
			Content: a.cfg.SystemPrompt,
		})
	}
	msgs = append(msgs, Message{
		Role:    "user",
		Content: input,
	})

	steps := a.cfg.MaxSteps
	if steps <= 0 {
		steps = 3
	}

	for i := 0; i < steps; i++ {
		resp, err := a.llm.Chat(ctx, &ChatRequest{
			Model:    model,
			Messages: msgs,
		})
		if err != nil {
			return "", err
		}
		content := strings.TrimSpace(resp.Content)

		toolName, toolInput, ok := parseToolCall(content)
		if ok {
			tool, found := a.tools.Get(toolName)
			if !found {
				return "", fmt.Errorf("agent: tool not found: %s", toolName)
			}
			result, err := tool.Call(ctx, toolInput)
			if err != nil {
				return "", fmt.Errorf("agent: tool call failed: %w", err)
			}

			msgs = append(msgs, Message{Role: "assistant", Content: content})
			msgs = append(msgs, Message{Role: "tool", Content: result})
			continue
		}

		msgs = append(msgs, Message{Role: "assistant", Content: content})
		a.memory.Save(sessionID, msgs)
		return content, nil
	}

	msgs = append(msgs, Message{Role: "assistant", Content: "max step reached"})
	a.memory.Save(sessionID, msgs)
	return "max step reached", nil
}

func parseToolCall(content string) (name string, input string, ok bool) {
	if !strings.HasPrefix(content, "tool:") {
		return "", "", false
	}
	parts := strings.SplitN(content, ":", 3)
	if len(parts) < 2 {
		return "", "", false
	}
	name = strings.TrimSpace(parts[1])
	if name == "" {
		return "", "", false
	}
	if len(parts) == 3 {
		input = parts[2]
	}
	return name, input, true
}

