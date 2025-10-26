package xclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOAuth1SigningAddsHeader(t *testing.T) {
	base := NewHTTPClient("")
	v1 := NewV1Client(base, "ck", "cs", "at", "as")
ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Fatalf("missing Authorization header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
	}))
	defer ts.Close()
	// call GetHomeTimeline since it signs the request
	v1.Base.baseURL = ts.URL
_, _ = v1.GetHomeTimeline(context.Background(), "", 5)
}
