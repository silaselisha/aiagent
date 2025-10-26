package suggest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
    "strings"
)

type oaMessage struct {
	Role    string    `json:"role"`
	Content []oaBlock `json:"content"`
}

type oaBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type oaResp struct {
	OutputText string `json:"output_text"`
}

func defaultNewRequest(ctx context.Context, url, method, body string) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, url, io.NopCloser(strings.NewReader(body)))
}

func defaultDo(req *http.Request) (*http.Response, error) {
	client := &http.Client{}
	return client.Do(req)
}

func parseOpenAIResponse(resp *http.Response) (string, error) {
	// Responses API returns a complex structure; we extract text heuristically.
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil { return "", err }
	if out, ok := raw["output_text"].(string); ok {
		return out, nil
	}
	// fallback: try choices[0].message/content[0].text
	if choices, ok := raw["choices"].([]any); ok && len(choices) > 0 {
		if ch, ok := choices[0].(map[string]any); ok {
			if msg, ok := ch["message"].(map[string]any); ok {
				if content, ok := msg["content"].([]any); ok && len(content) > 0 {
					if blk, ok := content[0].(map[string]any); ok {
						if t, ok := blk["text"].(string); ok { return t, nil }
					}
				}
			}
		}
	}
	return "", nil
}
