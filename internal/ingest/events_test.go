package ingest

import (
	"context"
	"testing"
	"time"

	"starseed/internal/store/sqlitevec"
)

type fakeX struct{}

func (f fakeX) GetUserByUsername(ctx context.Context, username string) (struct{ ID, Username string }, error) {
	return struct{ ID, Username string }{ID: "me", Username: username}, nil
}

func (f fakeX) GetLikedTweets(ctx context.Context, userID string, limit int) ([]struct{ ID, AuthorID string; CreatedAt time.Time }, error) { return nil, nil }
func (f fakeX) GetMentions(ctx context.Context, userID string, limit int) ([]struct{ ID, AuthorID string; CreatedAt time.Time }, error) {
	return []struct{ ID, AuthorID string; CreatedAt time.Time }{{ID: "t1", AuthorID: "a1", CreatedAt: time.Now().UTC()}}, nil
}

func TestBackfillLabelsFromMentions(t *testing.T) {
	db, err := sqlitevec.Open(":memory:")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	ctx := context.Background()
	ws := time.Now().UTC().Add(-30 * time.Minute)
	// insert a feature window
	if err := db.PutFeature(ctx, ws, []float32{1,2,3}, nil, nil); err != nil { t.Fatal(err) }
	// ingest a mention treated as reply in next window
	_ = db.PutEvent(ctx, ws.Add(20*time.Minute), "reply", map[string]any{"tweet_id":"t1"})
	if err := BackfillLabels(ctx, db, ws, ws.Add(45*time.Minute)); err != nil { t.Fatal(err) }
	_, _, y, err := db.LoadFeatures(ctx, ws, ws.Add(time.Hour))
	if err != nil { t.Fatal(err) }
	if len(y) == 0 || y[0] <= 0 { t.Fatalf("expected positive label, got %v", y) }
}
