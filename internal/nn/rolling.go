package nn

import (
	"context"
	"time"

	"starseed/internal/model"
	"starseed/internal/store/sqlitevec"
)

// BuildFeaturesWithHistory computes features using stored history for rolling stats and encodings.
func BuildFeaturesWithHistory(ctx context.Context, db *sqlitevec.DB, windowStart time.Time, tweets []model.Tweet, events []model.EngagementEvent) (FeatureVector, error) {
	fv := BuildFeatures(windowStart, tweets, events)
	// Example: fetch past hour windows to compute true rolling aggregates
	pastStart := windowStart.Add(-60 * time.Minute)
	_, X, _, err := db.LoadFeatures(ctx, pastStart, windowStart)
	if err == nil && len(X) > 0 {
		// Replace placeholder rolling slots with means of last windows
		meanCount := float32(0)
		meanAvgLikes := float32(0)
		for _, v := range X {
			if len(v) >= 6 { // per BuildFeatures: [log_count, log_likes, log_replies, log_retweets, log_quotes, avgLikes, ...]
				meanCount += v[0]
				meanAvgLikes += v[5]
			}
		}
		n := float32(len(X))
		if n > 0 {
			// rolling area begins at the end of base 8 features
			base := 8
			for i := 0; i < 4; i++ {
				fv.X[base+2*i] = meanCount / n
				fv.X[base+2*i+1] = meanAvgLikes / n
			}
		}
	}
	return fv, nil
}
