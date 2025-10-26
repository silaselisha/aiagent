package ingest

import (
    "context"
    "math"
    "time"

    "starseed/internal/store/sqlitevec"
    "starseed/internal/xclient"
)

// IngestEngagements fetches likes and mentions and stores them as events.
func IngestEngagements(ctx context.Context, db *sqlitevec.DB, client xclient.XClient, userID string, since time.Time) error {
	likes, err := client.GetLikedTweets(ctx, userID, 100)
	if err == nil {
		for _, t := range likes {
			_ = db.PutEvent(ctx, t.CreatedAt, "like", map[string]any{"tweet_id": t.ID, "author_id": t.AuthorID})
		}
	}
	mentions, err := client.GetMentions(ctx, userID, 100)
	if err == nil {
		for _, t := range mentions {
            // Treat mentions as replies proxy for labeling
            _ = db.PutEvent(ctx, t.CreatedAt, "reply", map[string]any{"tweet_id": t.ID, "author_id": t.AuthorID})
		}
	}
	return nil
}

// BackfillLabels computes y(t+1) = log1p(replies) proxy using next-window reply events.
func BackfillLabels(ctx context.Context, db *sqlitevec.DB, start, end time.Time) error {
	keys, _, _, err := db.LoadFeatures(ctx, start, end)
	if err != nil { return err }
	for _, ws := range keys {
		nextStart := ws.Add(15 * time.Minute)
		nextEnd := nextStart.Add(15 * time.Minute)
		events, err := db.LoadEventsRange(ctx, nextStart, nextEnd, "reply")
		if err != nil { continue }
		var replies int
		for range events { replies++ }
		label := float32(mathLog1p(float64(replies)))
		_ = db.UpdateFeatureLabel(ctx, ws, label)
	}
	return nil
}

func mathLog1p(x float64) float32 { return float32(math.Log1p(x)) }
