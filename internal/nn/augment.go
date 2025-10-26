package nn

import (
	"context"
	"time"

	"starseed/internal/model"
	"starseed/internal/store/sqlitevec"
)

// AugmentMeta fills in the meta feature slots of fv using authors and keywords.
func AugmentMeta(fv *FeatureVector, tweets []model.Tweet, authors map[string]model.User, keywords []string, weights map[string]float64) {
	m := MetaFeatures(tweets, authors, keywords, weights)
	// base length per BuildFeatures: 8 base + 8 rolling + 2 time = 18; meta starts at index 18
	if len(fv.X) >= 18+5 {
		base := 18
		for i := 0; i < 5; i++ { fv.X[base+i] = m[i] }
	}
}

// BuildAndPersistWindow composes features for a window, augments meta from DB-known authors if available, and stores.
func BuildAndPersistWindow(ctx context.Context, db *sqlitevec.DB, windowStart time.Time, tweets []model.Tweet, events []model.EngagementEvent, authors map[string]model.User, keywords []string, weights map[string]float64) (FeatureVector, error) {
	fv, err := BuildFeaturesWithHistory(ctx, db, windowStart, tweets, events)
	if err != nil { return fv, err }
	AugmentMeta(&fv, tweets, authors, keywords, weights)
	// label remains nil for now; another routine will backfill labels based on events
	_ = db.PutFeature(ctx, windowStart, fv.X, nil, map[string]any{"source":"window"})
	return fv, nil
}
