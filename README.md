# Go Agent Starter（多模型路由 + Tool + Memory + Ollama）

该模块是一个可扩展的 Go Agent 骨架，面向“工具调用 + 多模型调度 + 会话记忆”的落地场景。

- **LLM 接入**：内置 Ollama `/api/chat` 实现
- **多模型调度**：按 fast/normal/best（Q2/Q4/Q8）路由到不同模型名
- **Tool 系统**：`agent` 仅保留 `ToolManager` 抽象，具体实现建议使用独立库 `llm-tool`
- **Tool Router**：多信号决策（规则 + Memory 命中 + LLM 判断 + 工具可用性）
- **Memory**：仅保留 `AgentMemory` 接口，具体实现由外部注入

## 目录结构

```
agent/
  agent.go
  config.go
  executor.go
  llm.go
  memory_manager.go
  model_router.go
  tool_interface.go
  tool_router.go
  tool_router_test.go
  cmd/
    demo/
      main.go
```

## 核心接口

### LLM

- `LLM.Chat(ctx, req)`：输入 `model + messages`，返回 assistant 内容

### Tool

- `Tool.Call(ctx, input string) (string, error)`：输入/输出统一为字符串（推荐 JSON）
- ReAct 触发格式：`tool:<tool_name>:<tool_input>`
- Router 优先决策：先判断“是否该调工具”，再进入 LLM 主回答环节
- 工具管理：通过 `WithToolManager(...)` 注入外部实现（推荐 `github.com/kordar/llm-tool`）
- 本地开发：当前仓库目录名为 `tool/`，模块名仍是 `github.com/kordar/llm-tool`，通常用 `replace` 指向 `../tool`

### Memory
- 统一抽象：`AgentMemory.Build(...)` + `AgentMemory.Persist(...)`
- 注入方式：`WithMemory(...)` 或 `WithMemoryManager(...)`
- 说明：agent 包内不再提供任何 memory 默认实现

## 运行 Demo

前置：
- 本地已启动 Ollama（默认 `http://localhost:11434`）

运行：

```bash
go run ./cmd/demo
```

说明：
- demo 使用 `llm-tool` 的 `Registry`，并通过一个轻量适配器对接 `agent.ToolManager`
- 这样可以保持 `agent` 只依赖抽象接口，不绑定具体工具库实现

环境变量：
- `OLLAMA_ENDPOINT`：默认 `http://localhost:11434`
- `OLLAMA_BEARER_TOKEN`：可选，为 Ollama 请求注入 `Authorization: Bearer <token>`

## 配置

- `MaxSteps`：单轮对话内最多推理/工具迭代次数
- `Timeout`：单次 `Run` 超时时间
- `SystemPrompt`：系统提示词
- `ToolDecision.EnableRouter`：是否启用 Router 决策层
- `ToolDecision.LLMConfidenceThreshold`：LLM 低置信度阈值（低于阈值偏向不调工具）

## 扩展建议

- **新增工具**：实现 `Tool` 接口并 `RegisterTool` 注册
- **工具库解耦**：将 `Registry` 与通用工具实现放在 `llm-tool`，agent 只依赖接口
- **接入 RAG**：把 RAG 的 `Search/Index` 封装成 Tool（`rag_search` / `rag_upsert_doc`）
- **生产化**：增加鉴权、限流、审计、超时/重试、工具白名单、租户隔离
- **工具治理**：增加工具级限流、缓存、fallback 链路与调用观测
