package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestMetricsExposure(t *testing.T) {
	IngestRuns.Inc()
	IngestErrors.Inc()
    IncAPIRetry("/test")
	ObserveIngestDuration(time.Now().Add(-1500 * time.Millisecond))

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	promhttp.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("metrics status: %d", rec.Code)
	}
	body := rec.Body.String()
	for _, m := range []string{
		"starseed_ingest_runs_total",
		"starseed_ingest_errors_total",
		"starseed_ingest_duration_seconds",
		"starseed_api_retries_total",
	} {
		if !strings.Contains(body, m) {
			t.Fatalf("expected metric %s in body", m)
		}
	}
}
