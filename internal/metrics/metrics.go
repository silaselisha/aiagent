package metrics

import (
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	IngestRuns = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "starseed_ingest_runs_total",
		Help: "Total ingestion runs",
	})
	IngestErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "starseed_ingest_errors_total",
		Help: "Total ingestion errors",
	})
	IngestDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "starseed_ingest_duration_seconds",
		Help:    "Ingestion duration seconds",
		Buckets: prometheus.DefBuckets,
	})
    APIRetries = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "starseed_api_retries_total",
        Help: "Total API retry attempts",
    }, []string{"endpoint"})
)

func init() {
    prometheus.MustRegister(IngestRuns, IngestErrors, IngestDuration, APIRetries)
}

// StartServer starts a metrics HTTP server on addr (e.g., ":9090").
func StartServer(addr string) {
	if addr == "" {
		addr = os.Getenv("METRICS_ADDR")
	}
	if addr == "" {
		return
	}
    http.Handle("/metrics", promhttp.Handler())
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request){ w.WriteHeader(http.StatusOK) })
	go func() { _ = http.ListenAndServe(addr, nil) }()
}

// ObserveIngestDuration records a run duration
func ObserveIngestDuration(start time.Time) {
	d := time.Since(start).Seconds()
	IngestDuration.Observe(d)
}

// IncAPIRetry increments the retry counter for an endpoint.
func IncAPIRetry(endpoint string) { APIRetries.WithLabelValues(endpoint).Inc() }
