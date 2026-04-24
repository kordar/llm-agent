package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type LLM interface {
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Content string
}

type OllamaClient struct {
	Endpoint string
	Client   *http.Client
	Headers  map[string]string
}

func NewOllamaClient(endpoint string) *OllamaClient {
	return &OllamaClient{
		Endpoint: endpoint,
		Client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (o *OllamaClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if o == nil {
		return nil, errors.New("agent: nil ollama client")
	}
	if req == nil {
		return nil, errors.New("agent: nil chat request")
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, errors.New("agent: empty model")
	}

	body := map[string]any{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   false,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := strings.TrimRight(o.Endpoint, "/") + "/api/chat"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range o.Headers {
		httpReq.Header.Set(k, v)
	}

	client := o.Client
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var raw bytes.Buffer
		_, _ = raw.ReadFrom(resp.Body)
		return nil, fmt.Errorf("agent: ollama http %d: %s", resp.StatusCode, raw.String())
	}

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &ChatResponse{Content: result.Message.Content}, nil
}

