package agent

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
)

// MultiLLM routes requests to different LLM clients and supports ordered fallback.
type MultiLLM struct {
	mu sync.RWMutex

	byModel map[string]LLM
	byPref  map[string]LLM

	defaultClient LLM
	fallback      []LLM
}

func NewMultiLLM() *MultiLLM {
	return &MultiLLM{
		byModel: make(map[string]LLM),
		byPref:  make(map[string]LLM),
	}
}

func (m *MultiLLM) SetModelClient(model string, client LLM) {
	if m == nil {
		return
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if client == nil {
		delete(m.byModel, model)
		return
	}
	m.byModel[model] = client
}

func (m *MultiLLM) SetPrefixClient(prefix string, client LLM) {
	if m == nil {
		return
	}
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if client == nil {
		delete(m.byPref, prefix)
		return
	}
	m.byPref[prefix] = client
}

func (m *MultiLLM) SetDefaultClient(client LLM) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.defaultClient = client
	m.mu.Unlock()
}

func (m *MultiLLM) SetFallbackChain(clients ...LLM) {
	if m == nil {
		return
	}
	filtered := make([]LLM, 0, len(clients))
	for _, c := range clients {
		if c != nil {
			filtered = append(filtered, c)
		}
	}
	m.mu.Lock()
	m.fallback = filtered
	m.mu.Unlock()
}

func (m *MultiLLM) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if m == nil {
		return nil, errors.New("agent: nil multi llm")
	}
	if req == nil {
		return nil, errors.New("agent: nil chat request")
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		return nil, errors.New("agent: empty model")
	}

	candidates := m.resolveCandidates(model)
	if len(candidates) == 0 {
		return nil, errors.New("agent: no llm client found for model: " + model)
	}

	var errs []error
	for _, c := range candidates {
		resp, err := c.Chat(ctx, req)
		if err == nil {
			return resp, nil
		}
		errs = append(errs, err)
	}
	return nil, errors.Join(errs...)
}

func (m *MultiLLM) resolveCandidates(model string) []LLM {
	m.mu.RLock()
	defer m.mu.RUnlock()

	candidates := make([]LLM, 0, 1+len(m.byPref)+1+len(m.fallback))
	seen := make(map[LLM]struct{})
	push := func(c LLM) {
		if c == nil {
			return
		}
		if _, ok := seen[c]; ok {
			return
		}
		seen[c] = struct{}{}
		candidates = append(candidates, c)
	}

	// 1) exact model match
	push(m.byModel[model])

	// 2) longest-prefix match first
	prefixes := make([]string, 0, len(m.byPref))
	for p := range m.byPref {
		prefixes = append(prefixes, p)
	}
	sort.SliceStable(prefixes, func(i, j int) bool { return len(prefixes[i]) > len(prefixes[j]) })
	for _, p := range prefixes {
		if strings.HasPrefix(model, p) {
			push(m.byPref[p])
		}
	}

	// 3) default client
	push(m.defaultClient)

	// 4) explicit fallback chain
	for _, c := range m.fallback {
		push(c)
	}
	return candidates
}

