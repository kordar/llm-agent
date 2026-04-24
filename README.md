# Go Agent Starter（多模型路由 + Tool + Memory + Ollama）

该模块是一个可扩展的 Go Agent 骨架，面向“工具调用 + 多模型调度 + 会话记忆”的落地场景。

- **LLM 接入**：内置 Ollama `/api/chat` 实现
- **多模型调度**：按 fast/normal/best（Q2/Q4/Q8）路由到不同模型名
- **Tool 系统**：插件化工具注册与调用（ReAct 风格）
- **Memory**：默认内存版，可替换为 Redis/DB

## 目录结构

```
agent/
  agent.go
  config.go
  executor.go
  llm.go
  memory.go
  model_router.go
  tool.go
  tools/
    time_tool.go
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

### Memory

- `Load(sessionID)` / `Save(sessionID, msgs)`

## 运行 Demo

前置：
- 本地已启动 Ollama（默认 `http://localhost:11434`）

运行：

```bash
go run ./cmd/demo
```

环境变量：
- `OLLAMA_ENDPOINT`：默认 `http://localhost:11434`

## 扩展建议

- **新增工具**：实现 `Tool` 接口并 `RegisterTool` 注册
- **接入 RAG**：把 RAG 的 `Search/Index` 封装成 Tool（`rag_search` / `rag_upsert_doc`）
- **生产化**：增加鉴权、限流、审计、超时/重试、工具白名单、租户隔离

