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
	if a.mem == nil {
		return "", errors.New("agent: nil memory implementation")
	}

	model := a.router.Get(level)
	if model == "" {
		return "", errors.New("agent: no model found for level")
	}

	if a.cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.cfg.Timeout)
		defer cancel()
	}

	msgs, err := a.mem.Build(ctx, sessionID, input)
	if err != nil {
		return "", err
	}
	if a.cfg.SystemPrompt != "" {
		need := true
		if len(msgs) > 0 && msgs[0].Role == "system" && msgs[0].Content == a.cfg.SystemPrompt {
			need = false
		}
		if need {
			msgs = append([]Message{{
				Role:    "system",
				Content: a.cfg.SystemPrompt,
			}}, msgs...)
		}
	}
	msgs = append(msgs, Message{
		Role:    "user",
		Content: input,
	})
	if err := a.mem.Persist(ctx, sessionID, msgs); err != nil {
		return "", err
	}

	steps := a.cfg.MaxSteps
	if steps <= 0 {
		steps = 3
	}

	toolAttempted := false
	for i := 0; i < steps; i++ {
		decision := ToolDecision{UseTool: false}
		if !toolAttempted && a.toolRouter != nil {
			decision = a.toolRouter.Decide(ctx, model, input, msgs)
		}
		if decision.UseTool {
			toolAttempted = true
			if result, ok, err := a.tryToolCall(ctx, decision.Tool, input); err != nil {
				msgs = append(msgs, Message{
					Role:    "assistant",
					Content: "工具调用失败，降级为模型直答: " + err.Error(),
				})
			} else if ok {
				msgs = append(msgs, Message{Role: "assistant", Content: "tool:" + decision.Tool + ":" + input})
				msgs = append(msgs, Message{Role: "tool", Content: result})
				if err := a.mem.Persist(ctx, sessionID, msgs); err != nil {
					return "", err
				}
			}
		}

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
			result, used, err := a.tryToolCall(ctx, toolName, toolInput)
			if err != nil {
				msgs = append(msgs, Message{
					Role:    "assistant",
					Content: "工具调用失败，降级为模型直答: " + err.Error(),
				})
				if err := a.mem.Persist(ctx, sessionID, msgs); err != nil {
					return "", err
				}
				continue
			}
			if !used {
				msgs = append(msgs, Message{
					Role:    "assistant",
					Content: "工具不可用，已降级为模型直答",
				})
				if err := a.mem.Persist(ctx, sessionID, msgs); err != nil {
					return "", err
				}
				continue
			}

			msgs = append(msgs, Message{Role: "assistant", Content: content})
			msgs = append(msgs, Message{Role: "tool", Content: result})
			if err := a.mem.Persist(ctx, sessionID, msgs); err != nil {
				return "", err
			}
			continue
		}

		msgs = append(msgs, Message{Role: "assistant", Content: content})
		if err := a.mem.Persist(ctx, sessionID, msgs); err != nil {
			return "", err
		}
		return content, nil
	}

	msgs = append(msgs, Message{Role: "assistant", Content: "max step reached"})
	if err := a.mem.Persist(ctx, sessionID, msgs); err != nil {
		return "", err
	}
	return "max step reached", nil
}

func (a *Agent) tryToolCall(ctx context.Context, toolName string, toolInput string) (string, bool, error) {
	toolName = strings.TrimSpace(toolName)
	if toolName == "" {
		return "", false, nil
	}
	tool, found := a.tools.Get(toolName)
	if !found {
		return "", false, fmt.Errorf("agent: tool not found: %s", toolName)
	}
	result, err := tool.Call(ctx, toolInput)
	if err != nil {
		return "", true, fmt.Errorf("agent: tool call failed: %w", err)
	}
	return result, true, nil
}

func parseToolCall(content string) (name string, input string, ok bool) {
	if !strings.HasPrefix(content, "tool:") {
		return "", "", false
	}
	if strings.ContainsAny(content, "\r\n") {
		return "", "", false
	}
	parts := strings.SplitN(content, ":", 3)
	if len(parts) != 3 {
		return "", "", false
	}
	if parts[0] != "tool" {
		return "", "", false
	}
	name = strings.TrimSpace(parts[1])
	if name == "" {
		return "", "", false
	}
	if !isValidToolName(name) {
		return "", "", false
	}
	input = parts[2]
	return name, input, true
}

func isValidToolName(name string) bool {
	for _, r := range name {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}
