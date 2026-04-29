# llm-agent

`llm-agent` 是一个轻量级 Go Agent 运行时，提供以下能力：

- 统一的 `LLM` 抽象
- 可插拔工具系统（`Tool` + `ToolManager`）
- 可插拔记忆系统（`AgentMemory`）
- 工具路由决策（规则 + 记忆命中 + LLM 判定）
- 模型分级路由（`fast/normal/best`）
- 内置 `OllamaClient`（对接 Ollama `/api/chat`）

## 安装

```bash
go get github.com/kordar/llm-agent
```

## 核心概念

- `Agent`：运行入口，负责多轮消息编排、工具调用循环与消息持久化
- `LLM`：模型接口，方法为 `Chat(ctx, *ChatRequest) (*ChatResponse, error)`
- `Tool`：工具接口，统一 `Call(ctx, input string) (string, error)`
- `ToolManager`：工具注册与查询抽象
- `AgentMemory`：会话记忆抽象（`Build` / `Persist`）
- `ModelRouter`：将 `ModelLevel` 映射到具体模型名

## 最小可用示例

下面示例展示如何组装一个可运行的 Agent。为了便于复制，示例使用内存版 `ToolManager` 和 `AgentMemory`。

```go
package main

import (
	"context"
	"fmt"
	"sync"

	agent "github.com/kordar/llm-agent"
)

type memory struct {
	mu   sync.RWMutex
	data map[string][]agent.Message
}

func newMemory() *memory { return &memory{data: map[string][]agent.Message{}} }

func (m *memory) Build(_ context.Context, sessionID string, _ string) ([]agent.Message, error) {
	m.mu.RLock()
	msgs := m.data[sessionID]
	m.mu.RUnlock()
	out := make([]agent.Message, len(msgs))
	copy(out, msgs)
	return out, nil
}

func (m *memory) Persist(_ context.Context, sessionID string, msgs []agent.Message) error {
	out := make([]agent.Message, len(msgs))
	copy(out, msgs)
	m.mu.Lock()
	m.data[sessionID] = out
	m.mu.Unlock()
	return nil
}

type timeTool struct{}

func (timeTool) Name() string        { return "time" }
func (timeTool) Description() string { return "返回当前时间" }
func (timeTool) Call(_ context.Context, _ string) (string, error) {
	return "2026-01-01 12:00:00", nil
}

type toolMgr struct{ items map[string]agent.Tool }

func newToolMgr() *toolMgr { return &toolMgr{items: map[string]agent.Tool{}} }
func (m *toolMgr) Register(t agent.Tool) error {
	m.items[t.Name()] = t
	return nil
}
func (m *toolMgr) Get(name string) (agent.Tool, bool) { t, ok := m.items[name]; return t, ok }
func (m *toolMgr) List() []agent.Tool {
	out := make([]agent.Tool, 0, len(m.items))
	for _, t := range m.items {
		out = append(out, t)
	}
	return out
}

func main() {
	llm := agent.NewOllamaClient("http://localhost:11434")
	a := agent.NewAgent(
		llm,
		agent.WithMemory(newMemory()),
		agent.WithToolManager(newToolMgr()),
	)

	_ = a.RegisterTool(timeTool{})
	resp, err := a.Run(context.Background(), "session-1", "现在几点？", agent.LevelFast)
	if err != nil {
		panic(err)
	}
	fmt.Println(resp)
}
```

## `Run` 行为说明

`Run(ctx, sessionID, input, level)` 的主流程：

1. 读取会话历史（`AgentMemory.Build`）
2. 注入系统提示词（如未重复）
3. 追加用户输入并持久化
4. 进入最多 `MaxSteps` 次循环：
5. 工具路由器先判断是否要调工具（可关闭）
6. 调用 LLM；若返回 `tool:<name>:<input>` 则触发工具调用
7. 将 assistant/tool 消息写回记忆
8. 得到最终答案或达到步数上限

工具调用格式要求为单行：

```text
tool:<tool_name>:<tool_input>
```

## 配置项

`DefaultConfig()` 默认值：

- `MaxSteps = 3`
- `Timeout = 30s`
- `SystemPrompt` 为内置工具调用规则提示词
- `ToolDecision.EnableRouter = true`
- `ToolDecision.LLMConfidenceThreshold = 0.6`

可通过 `WithConfig(cfg)` 覆盖。

## Ollama 客户端

`NewOllamaClient(endpoint)` 默认行为：

- 请求路径：`POST {endpoint}/api/chat`
- 自动发送 `Content-Type: application/json`
- 请求体固定 `stream=false`
- 默认 `http.Client.Timeout = 60s`
- 支持通过 `client.Headers` 注入额外请求头（例如 Bearer Token）

## 测试

```bash
go test ./...
```

当前已覆盖：

- 多模型回退链路
- 工具路由关键决策路径
- `OllamaClient` 请求与响应处理
