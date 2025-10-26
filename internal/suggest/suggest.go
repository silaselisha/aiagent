package suggest

import (
	"fmt"
	"strings"
	"time"

	"starseed/internal/model"
)

// Suggestion contains a proposed reply/comment and timing.
type Suggestion struct {
	Tweet model.Tweet
	When  time.Time
	Text  string
	Why   string
}

// HeuristicSuggest generates simple rule-based suggestions.
func HeuristicSuggest(tweets []model.Tweet, now time.Time) []Suggestion {
	out := make([]Suggestion, 0)
	for _, t := range tweets {
		org := model.OrganicContentScore(t)
		if org < 0.55 { // aim for organic content most of the time
			continue
		}
		if t.Language != "" && t.Language != "en" {
			continue
		}
		text := strings.TrimSpace(t.Text)
		if text == "" {
			continue
		}
		when := now.Add(5 * time.Minute)
		why := fmt.Sprintf("organic=%.2f, lang=%s", org, coalesce(t.Language, "n/a"))
		out = append(out, Suggestion{Tweet: t, When: when, Text: generateTemplate(text), Why: why})
	}
	return out
}

func generateTemplate(tweetText string) string {
	// Lightweight prompt-engineered reply template; no external LLM dependency.
	return fmt.Sprintf("Thoughtful take: %s — What trade-offs did you consider?", trimForPrompt(tweetText, 180))
}

func trimForPrompt(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

func coalesce(a, b string) string { if a != "" { return a }; return b }
