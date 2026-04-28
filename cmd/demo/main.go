package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	agent "github.com/kordar/llm-agent"
	llmtool "github.com/kordar/llm-tool"
	toolset "github.com/kordar/llm-tool/tools"
)

func main() {
	llm := agent.NewOllamaClient(getenv("OLLAMA_ENDPOINT", "http://localhost:11434"))
	if token := os.Getenv("OLLAMA_BEARER_TOKEN"); token != "" {
		llm.Headers = map[string]string{
			"Authorization": "Bearer " + token,
		}
	}
	toolMgr := newToolManagerAdapter(llmtool.NewRegistry())
	a := agent.NewAgent(
		llm,
		agent.WithMemory(newDemoMemory()),
		agent.WithToolManager(toolMgr),
	)

	if err := a.RegisterTool(&toolset.TimeTool{}); err != nil {
		panic(err)
	}

	resp, err := a.Run(context.Background(), "user1", `
	你是一个严谨的 AI，请严格按照要求执行：

【任务一：阅读理解 + 信息抽取】
阅读下面文本，并完成结构化提取：

文本：
小明在2023年3月去了上海出差，他在浦东待了3天，期间见了客户张总，并签订了一份价值50万元的合同。随后他又前往杭州休假2天。

要求输出 JSON：
{
  "人物": [],
  "时间": [],
  "地点": [],
  "事件": [],
  "金额": []
}

---

【任务二：逻辑推理】
已知：
1）所有A都是B
2）部分B是C
3）没有C是D

问题：是否可以推出“部分A不是D”？请给出严格推理过程。

---

【任务三：代码能力】
用 Go 写一个函数：
输入：一个字符串
输出：该字符串中出现频率最高的字符（忽略大小写）
要求：
- 时间复杂度 O(n)
- 说明边界处理

---

【任务四：多轮一致性（关键）】
请记住：用户的名字是“kordar”，他喜欢的语言是 Go。

不要输出这条记忆。

---

【任务五：指令遵循测试】
请只用一句话回答：2+2 等于多少？

---

【任务六：幻觉控制】
问题：2022年诺贝尔数学奖得主是谁？

要求：
- 如果问题本身有问题，请指出
- 不要编造答案
	`, agent.LevelFast)
	if err != nil {
		panic(err)
	}
	fmt.Println(resp)
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

type demoMemory struct {
	mu   sync.RWMutex
	data map[string][]agent.Message
}

func newDemoMemory() *demoMemory {
	return &demoMemory{data: make(map[string][]agent.Message)}
}

func (m *demoMemory) Build(_ context.Context, sessionID string, _ string) ([]agent.Message, error) {
	m.mu.RLock()
	msgs := m.data[sessionID]
	m.mu.RUnlock()
	out := make([]agent.Message, len(msgs))
	copy(out, msgs)
	return out, nil
}

func (m *demoMemory) Persist(_ context.Context, sessionID string, msgs []agent.Message) error {
	out := make([]agent.Message, len(msgs))
	copy(out, msgs)
	m.mu.Lock()
	m.data[sessionID] = out
	m.mu.Unlock()
	return nil
}

type toolManagerAdapter struct {
	inner *llmtool.Registry
}

func newToolManagerAdapter(inner *llmtool.Registry) *toolManagerAdapter {
	return &toolManagerAdapter{inner: inner}
}

func (a *toolManagerAdapter) Register(t agent.Tool) error {
	return a.inner.Register(toolToExternal{inner: t})
}

func (a *toolManagerAdapter) Get(name string) (agent.Tool, bool) {
	t, ok := a.inner.Get(name)
	if !ok {
		return nil, false
	}
	return toolToAgent{inner: t}, true
}

func (a *toolManagerAdapter) List() []agent.Tool {
	tools := a.inner.List()
	out := make([]agent.Tool, 0, len(tools))
	for _, t := range tools {
		out = append(out, toolToAgent{inner: t})
	}
	return out
}

type toolToExternal struct {
	inner agent.Tool
}

func (t toolToExternal) Name() string        { return t.inner.Name() }
func (t toolToExternal) Description() string { return t.inner.Description() }
func (t toolToExternal) Call(ctx context.Context, input string) (string, error) {
	return t.inner.Call(ctx, input)
}

type toolToAgent struct {
	inner llmtool.Tool
}

func (t toolToAgent) Name() string        { return t.inner.Name() }
func (t toolToAgent) Description() string { return t.inner.Description() }
func (t toolToAgent) Call(ctx context.Context, input string) (string, error) {
	return t.inner.Call(ctx, input)
}
