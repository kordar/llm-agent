package agent

import "time"

type Config struct {
	MaxSteps     int
	SystemPrompt string
	Timeout      time.Duration
}

func DefaultConfig() Config {
	return Config{
		MaxSteps: 3,
		SystemPrompt: `你是一个可调用工具的智能助手。
当你需要调用工具时，严格返回：tool:<tool_name>:<tool_input>
当你不需要调用工具时，直接返回最终答案。`,
		Timeout: 30 * time.Second,
	}
}

