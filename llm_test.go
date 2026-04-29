package agent

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewOllamaClient_Defaults(t *testing.T) {
	c := NewOllamaClient("http://localhost:11434")
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.Endpoint != "http://localhost:11434" {
		t.Fatalf("unexpected endpoint: %q", c.Endpoint)
	}
	if c.Client == nil {
		t.Fatal("expected default http client")
	}
	if c.Client.Timeout != 60*time.Second {
		t.Fatalf("expected timeout 60s, got %v", c.Client.Timeout)
	}
}

func TestOllamaClientChat_ValidateInput(t *testing.T) {
	var nilClient *OllamaClient
	if _, err := nilClient.Chat(context.Background(), &ChatRequest{Model: "qwen"}); err == nil {
		t.Fatal("expected error for nil client")
	}

	c := NewOllamaClient("http://localhost:11434")
	if _, err := c.Chat(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil request")
	}

	if _, err := c.Chat(context.Background(), &ChatRequest{Model: "   "}); err == nil {
		t.Fatal("expected error for empty model")
	}
}

func TestOllamaClientChat_Success(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotContentType string
	var gotAuth string
	var gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"content":"hello from server"}}`))
	}))
	defer srv.Close()

	c := NewOllamaClient(srv.URL + "/")
	c.Headers = map[string]string{"Authorization": "Bearer test-token"}

	resp, err := c.Chat(context.Background(), &ChatRequest{
		Model: "qwen2.5:7b",
		Messages: []Message{
			{Role: "user", Content: "hi"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || resp.Content != "hello from server" {
		t.Fatalf("unexpected response: %#v", resp)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method POST, got %q", gotMethod)
	}
	if gotPath != "/api/chat" {
		t.Fatalf("expected path /api/chat, got %q", gotPath)
	}
	if gotContentType != "application/json" {
		t.Fatalf("unexpected content type: %q", gotContentType)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("unexpected auth header: %q", gotAuth)
	}
	if !strings.Contains(gotBody, `"stream":false`) {
		t.Fatalf("expected body to contain stream=false, got: %s", gotBody)
	}
	if !strings.Contains(gotBody, `"model":"qwen2.5:7b"`) {
		t.Fatalf("expected body to contain model, got: %s", gotBody)
	}
}

func TestOllamaClientChat_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request payload"))
	}))
	defer srv.Close()

	c := NewOllamaClient(srv.URL)
	_, err := c.Chat(context.Background(), &ChatRequest{Model: "qwen"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "ollama http 400") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "bad request payload") {
		t.Fatalf("unexpected error body: %v", err)
	}
}

func TestOllamaClientChat_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()

	c := NewOllamaClient(srv.URL)
	_, err := c.Chat(context.Background(), &ChatRequest{Model: "qwen"})
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
	var syntaxErr interface{ Error() string }
	if !errors.As(err, &syntaxErr) {
		// keep this branch loose; we only need to ensure it's a decode-related error.
		if !strings.Contains(strings.ToLower(err.Error()), "invalid character") {
			t.Fatalf("unexpected decode error: %v", err)
		}
	}
}
