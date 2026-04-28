package agent

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

type ToolDecision struct {
	UseTool    bool
	Tool       string
	Reason     string
	Confidence float64
}

type ToolRouter struct {
	llm   LLM
	tools ToolManager
	cfg   ToolDecisionConfig
}

func NewToolRouter(llm LLM, tools ToolManager, cfg ToolDecisionConfig) *ToolRouter {
	return &ToolRouter{
		llm:   llm,
		tools: tools,
		cfg:   cfg,
	}
}

func (r *ToolRouter) Decide(ctx context.Context, model string, input string, history []Message) ToolDecision {
	if r == nil || !r.cfg.EnableRouter {
		return ToolDecision{UseTool: false, Reason: "router disabled"}
	}
	if len(r.tools.List()) == 0 {
		return ToolDecision{UseTool: false, Reason: "no tools registered", Confidence: 1}
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return ToolDecision{UseTool: false, Reason: "empty input"}
	}

	if t, ok := r.ruleSelect(input); ok {
		return ToolDecision{
			UseTool:    true,
			Tool:       t,
			Reason:     "rule matched",
			Confidence: 0.95,
		}
	}
	if hasUsefulMemory(history, input) {
		return ToolDecision{
			UseTool:    false,
			Reason:     "memory hit",
			Confidence: 0.85,
		}
	}

	llmDecision := r.llmJudge(ctx, model, input)
	if llmDecision.UseTool && llmDecision.Tool != "" {
		if _, ok := r.tools.Get(llmDecision.Tool); !ok {
			llmDecision.Tool = r.rankTool(input)
			llmDecision.Reason = "llm selected unavailable tool, ranked fallback"
		}
	}

	if llmDecision.UseTool && llmDecision.Tool == "" {
		llmDecision.Tool = r.rankTool(input)
	}
	if llmDecision.UseTool {
		if llmDecision.Tool == "" {
			return ToolDecision{UseTool: false, Reason: "no available tool", Confidence: llmDecision.Confidence}
		}
		if llmDecision.Confidence >= r.cfg.LLMConfidenceThreshold {
			return llmDecision
		}
		// Low confidence: avoid tool unless the input is clearly real-time or calculation oriented.
		if likelyRealtime(input) || likelyCalculation(input) {
			if llmDecision.Confidence < 0.5 {
				llmDecision.Confidence = 0.5
			}
			return llmDecision
		}
		return ToolDecision{
			UseTool:    false,
			Reason:     "llm confidence below threshold",
			Confidence: llmDecision.Confidence,
		}
	}
	return ToolDecision{
		UseTool:    false,
		Reason:     "llm direct answer",
		Confidence: llmDecision.Confidence,
	}
}

func (r *ToolRouter) ruleSelect(input string) (string, bool) {
	tools := r.tools.List()
	if len(tools) == 0 {
		return "", false
	}
	if likelyRealtime(input) || likelyCalculation(input) {
		return r.rankTool(input), true
	}

	lower := strings.ToLower(input)
	for _, t := range tools {
		n := strings.ToLower(t.Name())
		if n != "" && strings.Contains(lower, n) {
			return t.Name(), true
		}
	}
	return "", false
}

func (r *ToolRouter) llmJudge(ctx context.Context, model string, input string) ToolDecision {
	if r == nil || r.llm == nil {
		return ToolDecision{UseTool: false, Reason: "nil llm", Confidence: 0}
	}
	toolDesc := make([]string, 0, 8)
	for _, t := range r.tools.List() {
		toolDesc = append(toolDesc, t.Name()+":"+t.Description())
	}
	prompt := "你是工具路由器。判断是否需要调用工具。\n" +
		"只输出一行 JSON，不要输出其他文字。\n" +
		"JSON 格式: {\"use_tool\":true|false,\"tool\":\"工具名或空\",\"confidence\":0~1,\"reason\":\"简短理由\"}\n" +
		"可用工具: " + strings.Join(toolDesc, "; ") + "\n" +
		"用户问题: " + input

	resp, err := r.llm.Chat(ctx, &ChatRequest{
		Model: model,
		Messages: []Message{
			{Role: "system", Content: "你是一个严谨的工具决策器。"},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return ToolDecision{UseTool: false, Reason: "llm judge failed", Confidence: 0}
	}
	return parseRouterDecision(resp.Content)
}

func (r *ToolRouter) rankTool(query string) string {
	tools := r.tools.List()
	if len(tools) == 0 {
		return ""
	}
	qTokens := tokenizeForRoute(query)
	type candidate struct {
		name  string
		score float64
	}
	candidates := make([]candidate, 0, len(tools))
	for _, t := range tools {
		text := t.Name() + " " + t.Description()
		score := overlapScoreFloat(tokenizeForRoute(text), qTokens)
		if likelyCalculation(query) && strings.Contains(strings.ToLower(text), "calc") {
			score += 0.4
		}
		if likelyRealtime(query) && strings.Contains(strings.ToLower(text), "time") {
			score += 0.3
		}
		candidates = append(candidates, candidate{name: t.Name(), score: score})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})
	return candidates[0].name
}

func hasUsefulMemory(history []Message, query string) bool {
	queryTokens := tokensForSimilarity(query)
	if len(queryTokens) == 0 {
		return false
	}
	for i := len(history) - 1; i >= 0 && i >= len(history)-8; i-- {
		m := history[i]
		if m.Role != "assistant" && m.Role != "tool" {
			continue
		}
		memTokens := tokensForSimilarity(m.Content)
		score := overlapScoreFloat(memTokens, queryTokens)
		if score >= 0.4 {
			return true
		}
	}
	return false
}

func likelyRealtime(input string) bool {
	keywords := []string{"天气", "温度", "现在", "当前", "实时", "股价", "汇率", "时间", "日期"}
	for _, k := range keywords {
		if strings.Contains(input, k) {
			return true
		}
	}
	return false
}

func likelyCalculation(input string) bool {
	if strings.Contains(input, "计算") || strings.Contains(input, "算一下") {
		return true
	}
	digit := false
	op := false
	for _, r := range input {
		if r >= '0' && r <= '9' {
			digit = true
		}
		if r == '+' || r == '-' || r == '*' || r == '/' || r == '%' {
			op = true
		}
	}
	return digit && op
}

func parseRouterDecision(text string) ToolDecision {
	text = strings.TrimSpace(text)
	if text == "" {
		return ToolDecision{}
	}
	// Try direct JSON first.
	var raw map[string]any
	if json.Unmarshal([]byte(text), &raw) != nil {
		// Fallback: extract by first/last braces.
		l := strings.Index(text, "{")
		r := strings.LastIndex(text, "}")
		if l >= 0 && r > l {
			_ = json.Unmarshal([]byte(text[l:r+1]), &raw)
		}
	}
	if len(raw) == 0 {
		return ToolDecision{}
	}

	dec := ToolDecision{}
	if v, ok := raw["use_tool"].(bool); ok {
		dec.UseTool = v
	}
	if v, ok := raw["tool"].(string); ok {
		dec.Tool = strings.TrimSpace(v)
	}
	if v, ok := raw["reason"].(string); ok {
		dec.Reason = strings.TrimSpace(v)
	}
	switch v := raw["confidence"].(type) {
	case float64:
		dec.Confidence = v
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			dec.Confidence = f
		}
	}
	if dec.Confidence < 0 {
		dec.Confidence = 0
	}
	if dec.Confidence > 1 {
		dec.Confidence = 1
	}
	return dec
}

func tokenizeForRoute(s string) []string {
	s = strings.ToLower(strings.TrimSpace(s))
	raw := strings.Fields(s)
	out := make([]string, 0, len(raw))
	for _, t := range raw {
		t = strings.Trim(t, ".,!?;:\"'()[]{}")
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func cnBigrams(s string) []string {
	rs := []rune(strings.TrimSpace(s))
	if len(rs) < 2 {
		return nil
	}
	out := make([]string, 0, len(rs)-1)
	for i := 0; i < len(rs)-1; i++ {
		a := rs[i]
		b := rs[i+1]
		if a == ' ' || b == ' ' || a == '\n' || b == '\n' || a == '\t' || b == '\t' {
			continue
		}
		out = append(out, string([]rune{a, b}))
	}
	return out
}

func tokensForSimilarity(s string) []string {
	a := tokenizeForRoute(s)
	b := cnBigrams(s)
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	out := make([]string, 0, len(a)+len(b))
	out = append(out, a...)
	out = append(out, b...)
	return out
}

func overlapScoreFloat(a []string, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	set := make(map[string]struct{}, len(a))
	for _, x := range a {
		set[x] = struct{}{}
	}
	matched := 0
	for _, y := range b {
		if _, ok := set[y]; ok {
			matched++
		}
	}
	return float64(matched) / float64(len(b))
}
