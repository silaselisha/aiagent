package suggest

import (
	"context"
	"fmt"
	"strings"

	"starseed/internal/config"
)

// DraftWithLLM optionally upgrades a heuristic draft using an LLM provider.
func DraftWithLLM(ctx context.Context, cfg config.LLMConfig, tweetText, heuristic string) (string, error) {
	if strings.ToLower(cfg.Provider) != "openai" || cfg.APIKey == "" {
		return heuristic, nil
	}
	// Minimal inlined client to avoid heavy deps; replace with official SDK if desired.
	// OpenAI Responses API (JSON). We keep prompt small and grounded.
	payload := fmt.Sprintf(`{"model":"%s","input":[{"role":"user","content":[{"type":"text","text":"Tweet: %s\nDraft a concise, wise, kind, on-topic reply (max 220 chars)."}]}]}`, cfg.Model, escapeJSON(tweetText))
	req, err := httpNewRequest(ctx, "https://api.openai.com/v1/responses", "POST", payload)
	if err != nil { return heuristic, err }
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpDo(req)
	if err != nil { return heuristic, err }
	defer resp.Body.Close()
	if resp.StatusCode >= 400 { return heuristic, fmt.Errorf("llm status %d", resp.StatusCode) }
	text, err := parseOpenAIResponse(resp)
	if err != nil || strings.TrimSpace(text) == "" {
		return heuristic, err
	}
	return text, nil
}

// --- light http helpers (decoupled for testability) ---

var httpNewRequest = defaultNewRequest
var httpDo = defaultDo

// escapeJSON is minimal, for controlled prompts
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
