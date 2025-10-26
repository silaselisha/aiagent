package ingest

import (
	"context"
	"testing"
	"time"

	"starseed/internal/model"
	"starseed/internal/store/sqlitevec"
)

type fakeLikeClient struct{}

func (fakeLikeClient) GetUserByUsername(ctx context.Context, username string) (model.User, error) { return model.User{}, nil }
func (fakeLikeClient) GetHomeTimeline(ctx context.Context, userID string, limit int) ([]model.Tweet, error) { return nil, nil }
func (fakeLikeClient) GetFollowing(ctx context.Context, userID string, limit int) ([]model.User, error) { return nil, nil }
func (fakeLikeClient) SearchRecentTweets(ctx context.Context, query string, limit int) ([]model.Tweet, error) { return nil, nil }
func (fakeLikeClient) SearchRecentTweetsSince(ctx context.Context, query string, limit int, start time.Time) ([]model.Tweet, error) { return nil, nil }
func (fakeLikeClient) GetUserTweets(ctx context.Context, userID string, limit int) ([]model.Tweet, error) { return nil, nil }
func (fakeLikeClient) GetUsersByIDs(ctx context.Context, ids []string) ([]model.User, error) { return nil, nil }
func (fakeLikeClient) GetMentions(ctx context.Context, userID string, limit int) ([]model.Tweet, error) { return nil, nil }
func (fakeLikeClient) GetQuoteTweets(ctx context.Context, tweetID string, limit int) ([]model.Tweet, error) { return nil, nil }
func (fakeLikeClient) GetLikedTweets(ctx context.Context, userID string, limit int) ([]model.Tweet, error) {
	return []model.Tweet{{ID: "l1", AuthorID: "x", CreatedAt: time.Now().UTC()}}, nil
}

func TestOutboundLikesIdempotent(t *testing.T) {
	db, _ := sqlitevec.Open(":memory:")
	defer db.Close()
	ctx := context.Background()
	fx := fakeLikeClient{}
	since := time.Now().UTC().Add(-1 * time.Hour)
	if err := IngestEngagements(ctx, db, fx, "me", "me", since); err != nil { t.Fatal(err) }
	if err := IngestEngagements(ctx, db, fx, "me", "me", since); err != nil { t.Fatal(err) }
	start := time.Now().UTC().Add(-2 * time.Hour)
	end := time.Now().UTC().Add(2 * time.Hour)
	likes, err := db.LoadEventsRange(ctx, start, end, "like")
	if err != nil { t.Fatal(err) }
	if len(likes) != 1 { t.Fatalf("expected 1 like after idempotency, got %d", len(likes)) }
}
