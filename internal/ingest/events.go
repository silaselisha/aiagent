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
    now := time.Now().UTC()
    // Likes
    likesSince := since
    if v, err := db.LoadCursor(ctx, "ingest:likes_since"); err == nil {
        if ts, err2 := time.Parse(time.RFC3339Nano, v); err2 == nil { likesSince = ts }
    }
    if likes, err := client.GetLikedTweets(ctx, userID, 100); err == nil {
        for _, t := range likes {
            if t.CreatedAt.Before(likesSince) { continue }
            _ = db.PutEventRef(ctx, t.CreatedAt, "like", t.ID, map[string]any{"tweet_id": t.ID, "author_id": t.AuthorID})
        }
    }
    _ = db.SaveCursor(ctx, "ingest:likes_since", now.Format(time.RFC3339Nano))

    // Replies to us via search
    if username != "" {
        repliesSince := since
        if v, err := db.LoadCursor(ctx, "ingest:replies_since"); err == nil {
            if ts, err2 := time.Parse(time.RFC3339Nano, v); err2 == nil { repliesSince = ts }
        }
        q := "to:" + username
        if replies, err := client.SearchRecentTweetsSince(ctx, q, 100, repliesSince); err == nil {
            for _, t := range replies {
                if t.CreatedAt.Before(repliesSince) { continue }
                _ = db.PutEventRef(ctx, t.CreatedAt, "reply", t.ID, map[string]any{"tweet_id": t.ID, "author_id": t.AuthorID})
            }
        }
        _ = db.SaveCursor(ctx, "ingest:replies_since", now.Format(time.RFC3339Nano))
    }

    // Outbound retweets by us (approx): search from:username is:retweet
    if username != "" {
        rtSince := since
        if v, err := db.LoadCursor(ctx, "ingest:retweets_since"); err == nil {
            if ts, err2 := time.Parse(time.RFC3339Nano, v); err2 == nil { rtSince = ts }
        }
        q := "from:" + username + " is:retweet"
        if rts, err := client.SearchRecentTweetsSince(ctx, q, 100, rtSince); err == nil {
            for _, t := range rts {
                if t.CreatedAt.Before(rtSince) { continue }
                _ = db.PutEventRef(ctx, t.CreatedAt, "retweet", t.ID, map[string]any{"tweet_id": t.ID, "author_id": t.AuthorID})
            }
        }
        _ = db.SaveCursor(ctx, "ingest:retweets_since", now.Format(time.RFC3339Nano))
    }

    // Quotes of our recent tweets: fetch our recent tweets then quote_tweets per tweet
    qtSince := since
    if v, err := db.LoadCursor(ctx, "ingest:quotes_since"); err == nil {
        if ts, err2 := time.Parse(time.RFC3339Nano, v); err2 == nil { qtSince = ts }
    }
    if userID != "" {
        if my, err := client.GetUserTweets(ctx, userID, 20); err == nil {
            for _, orig := range my {
                if quotes, err := client.GetQuoteTweets(ctx, orig.ID, 50); err == nil {
                    for _, qt := range quotes {
                        if qt.CreatedAt.Before(qtSince) { continue }
                        _ = db.PutEventRef(ctx, qt.CreatedAt, "quote", qt.ID, map[string]any{"tweet_id": qt.ID, "author_id": qt.AuthorID, "target_id": orig.ID})
                    }
                }
            }
        }
    }
    _ = db.SaveCursor(ctx, "ingest:quotes_since", now.Format(time.RFC3339Nano))
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
