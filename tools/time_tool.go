package tools

import (
	"context"
	"time"
)

type TimeTool struct{}

func (t *TimeTool) Name() string {
	return "time"
}

func (t *TimeTool) Description() string {
	return "获取当前时间"
}

func (t *TimeTool) Call(ctx context.Context, input string) (string, error) {
	return time.Now().Format(time.RFC3339), nil
}

