package main

import (
	"context"
	"fmt"
	"os"

	agent "github.com/kordar/llm-agent"
	"github.com/kordar/llm-agent/tools"
)

func main() {
	llm := agent.NewOllamaClient(getenv("OLLAMA_ENDPOINT", "http://localhost:11434"))
	if token := os.Getenv("OLLAMA_BEARER_TOKEN"); token != "" {
		llm.Headers = map[string]string{
			"Authorization": "Bearer " + token,
		}
	}
	memory := agent.NewMemory()
	a := agent.NewAgent(llm, memory)

	if err := a.RegisterTool(&tools.TimeTool{}); err != nil {
		panic(err)
	}

	resp, err := a.Run(context.Background(), "user1", "你好", agent.LevelFast)
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
