package xclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// helper to create client with injected http client
func newTestClient() *HTTPClient {
	c := NewHTTPClient("test")
	c.maxAttempts = 3
	c.baseBackoff = 10 * time.Millisecond
	return c
}

func TestDoWithRetryHandles429(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	c := newTestClient()
	c.httpClient = ts.Client()
	c.baseURL = ts.URL

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
	resp, err := c.doWithRetry(context.Background(), req)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if attempts < 2 {
		t.Fatalf("expected at least 2 attempts, got %d", attempts)
	}
}
