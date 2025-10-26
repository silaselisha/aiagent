package xclient

import (
	"os"
	"strconv"

	"golang.org/x/time/rate"
)

// newDefaultLimiter creates a rate limiter using env overrides if present.
func newDefaultLimiter() *rate.Limiter {
	rps := 2.0
	burst := 10
	if v := os.Getenv("X_API_RPS"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 { rps = f }
	}
	if v := os.Getenv("X_API_BURST"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 { burst = n }
	}
	return rate.NewLimiter(rate.Limit(rps), burst)
}
