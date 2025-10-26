package ingest

import (
    "context"
    "math"
    "time"

    "starseed/internal/store/sqlitevec"
    "starseed/internal/xclient"
)

// IngestEngagements fetches likes and mentions and stores them as events.
func IngestEngagements(ctx context.Context, db *sqlitevec.DB, client xclient.XClient, userID string, username string, since time.Time) error {
    likes, err := client.GetLikedTweets(ctx, userID, 100)
	if err == nil {
		for _, t := range likes {
			_ = db.PutEvent(ctx, t.CreatedAt, "like", map[string]any{"tweet_id": t.ID, "author_id": t.AuthorID})
		}
	}
    // Fetch replies to our account via recent search: from cursor to next window end
    if username != "" {
        q := "to:" + username
        replies, err2 := client.SearchRecentTweetsSince(ctx, q, 100, time.Now().UTC().Add(-15*time.Minute))
        if err2 == nil {
            for _, t := range replies {
                _ = db.PutEvent(ctx, t.CreatedAt, "reply", map[string]any{"tweet_id": t.ID, "author_id": t.AuthorID})
            }
        }
    }
    // Retweets and quotes: infer via search and quote_tweets endpoint if needed
    // For simplicity, treat retweets as mentions of "RT @" in text (approximation), quotes via /quote_tweets per tweet event in window
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
