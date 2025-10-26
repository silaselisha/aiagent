package ingest

import (
	"context"
	"testing"
	"time"

	"starseed/internal/store/sqlitevec"
	"starseed/internal/model"
)

type fakeXIngest struct{}

func (f fakeXIngest) GetUserByUsername(ctx context.Context, username string) (model.User, error) { return model.User{}, nil }
func (f fakeXIngest) GetHomeTimeline(ctx context.Context, userID string, limit int) ([]model.Tweet, error) { return nil, nil }
func (f fakeXIngest) GetFollowing(ctx context.Context, userID string, limit int) ([]model.User, error) { return nil, nil }
func (f fakeXIngest) SearchRecentTweets(ctx context.Context, query string, limit int) ([]model.Tweet, error) { return nil, nil }
func (f fakeXIngest) SearchRecentTweetsSince(ctx context.Context, query string, limit int, start time.Time) ([]model.Tweet, error) {
	return []model.Tweet{{ID: "r1", AuthorID: "a2", CreatedAt: time.Now().UTC()}}, nil
}
func (f fakeXIngest) GetUserTweets(ctx context.Context, userID string, limit int) ([]model.Tweet, error) { return nil, nil }
func (f fakeXIngest) GetUsersByIDs(ctx context.Context, ids []string) ([]model.User, error) { return nil, nil }
func (f fakeXIngest) GetLikedTweets(ctx context.Context, userID string, limit int) ([]model.Tweet, error) {
	return []model.Tweet{{ID: "l1", AuthorID: "b2", CreatedAt: time.Now().UTC()}}, nil
}
func (f fakeXIngest) GetMentions(ctx context.Context, userID string, limit int) ([]model.Tweet, error) { return nil, nil }
func (f fakeXIngest) GetQuoteTweets(ctx context.Context, tweetID string, limit int) ([]model.Tweet, error) { return nil, nil }

func TestIngestEngagements_InsertsLikeAndReply(t *testing.T) {
	db, err := sqlitevec.Open(":memory:")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	ctx := context.Background()
	fx := fakeXIngest{}
	since := time.Now().UTC().Add(-1 * time.Hour)
	if err := IngestEngagements(ctx, db, fx, "me-id", "me", since); err != nil { t.Fatal(err) }
	// Verify likes
	start := time.Now().UTC().Add(-2 * time.Hour)
	end := time.Now().UTC().Add(2 * time.Hour)
	likes, err := db.LoadEventsRange(ctx, start, end, "like")
	if err != nil { t.Fatal(err) }
	if len(likes) == 0 { t.Fatalf("expected like events") }
	replies, err := db.LoadEventsRange(ctx, start, end, "reply")
	if err != nil { t.Fatal(err) }
	if len(replies) == 0 { t.Fatalf("expected reply events") }
}
