package jobs

import (
    "context"
    "testing"
    "time"

    "starseed/internal/config"
    "starseed/internal/model"
    "starseed/internal/store/sqlitevec"
    "starseed/internal/xclient"
)

// fake client implementing xclient.XClient
type fx struct{}

func (fx) GetUserByUsername(ctx context.Context, username string) (model.User, error) {
    return model.User{ID: "me", Username: username}, nil
}
func (fx) GetHomeTimeline(ctx context.Context, userID string, limit int) ([]model.Tweet, error) { return nil, nil }
func (fx) GetFollowing(ctx context.Context, userID string, limit int) ([]model.User, error) { return nil, nil }
func (fx) SearchRecentTweets(ctx context.Context, query string, limit int) ([]model.Tweet, error) { return nil, nil }
func (fx) SearchRecentTweetsSince(ctx context.Context, query string, limit int, start time.Time) ([]model.Tweet, error) {
    // Return a tweet 20 minutes after the ws used in the test to fall into next window
    // start is ignored in this fake; we base on current time
    return []model.Tweet{{ID: "r1", AuthorID: "a2", CreatedAt: time.Now().UTC().Add(-10 * time.Minute)}}, nil
}
func (fx) GetUserTweets(ctx context.Context, userID string, limit int) ([]model.Tweet, error) { return nil, nil }
func (fx) GetUsersByIDs(ctx context.Context, ids []string) ([]model.User, error) { return nil, nil }
func (fx) GetLikedTweets(ctx context.Context, userID string, limit int) ([]model.Tweet, error) { return nil, nil }
func (fx) GetMentions(ctx context.Context, userID string, limit int) ([]model.Tweet, error) {
    // Emit a mention within the next window relative to ws used in test: now-10m
    return []model.Tweet{{ID: "t1", AuthorID: "a1", CreatedAt: time.Now().UTC().Add(-10 * time.Minute)}}, nil
}
func (fx) GetQuoteTweets(ctx context.Context, tweetID string, limit int) ([]model.Tweet, error) { return nil, nil }

func TestRunIngestionOnce_AdvancesCursorAndLabels(t *testing.T) {
    db, err := sqlitevec.Open(":memory:")
    if err != nil { t.Fatal(err) }
    defer db.Close()
    ctx := context.Background()
    cfg := config.Default()
    cfg.Account.Username = "me"
    // Insert a feature window ending 15m before now
    ws := time.Now().UTC().Add(-30 * time.Minute)
    if err := db.PutFeature(ctx, ws, []float32{1,2,3,4}, nil, nil); err != nil { t.Fatal(err) }
    // Run ingestion once with horizon 1h
    var client xclient.XClient = fx{}
    if err := RunIngestionOnce(ctx, db, client, cfg, time.Hour); err != nil { t.Fatal(err) }
    // Cursor should exist
    if _, err := db.LoadCursor(ctx, cursorKey); err != nil { t.Fatalf("cursor not saved: %v", err) }
    // Label should be > 0 for the window
    _, _, y, err := db.LoadFeatures(ctx, ws, ws.Add(time.Hour))
    if err != nil || len(y) == 0 || y[0] <= 0 { t.Fatalf("expected positive label, got %v, err=%v", y, err) }
}

