package recommend

import (
	"context"
	"testing"
    "time"

	"starseed/internal/model"
    "starseed/internal/store/sqlitevec"
)

type fakeGraphClient struct{}

func (fakeGraphClient) GetUserByUsername(ctx context.Context, username string) (model.User, error) { return model.User{}, nil }
func (fakeGraphClient) GetHomeTimeline(ctx context.Context, userID string, limit int) ([]model.Tweet, error) { return nil, nil }
func (fakeGraphClient) GetFollowing(ctx context.Context, userID string, limit int) ([]model.User, error) {
	if userID == "seed" {
		return []model.User{{ID: "a", Username: "a"}, {ID: "b", Username: "b"}}, nil
	}
	return nil, nil
}
func (fakeGraphClient) SearchRecentTweets(ctx context.Context, query string, limit int) ([]model.Tweet, error) { return nil, nil }
func (fakeGraphClient) SearchRecentTweetsSince(ctx context.Context, query string, limit int, start time.Time) ([]model.Tweet, error) {
	return nil, nil
}
func (fakeGraphClient) GetUserTweets(ctx context.Context, userID string, limit int) ([]model.Tweet, error) { return nil, nil }
func (fakeGraphClient) GetUsersByIDs(ctx context.Context, ids []string) ([]model.User, error) { return nil, nil }
func (fakeGraphClient) GetLikedTweets(ctx context.Context, userID string, limit int) ([]model.Tweet, error) { return nil, nil }
func (fakeGraphClient) GetMentions(ctx context.Context, userID string, limit int) ([]model.Tweet, error) { return nil, nil }
func (fakeGraphClient) GetQuoteTweets(ctx context.Context, tweetID string, limit int) ([]model.Tweet, error) { return nil, nil }

func TestDiscoverGraphOneHop(t *testing.T) {
	ctx := context.Background()
	client := fakeGraphClient{}
	seed := []model.User{{ID: "seed", Username: "seed"}}
	got, err := DiscoverGraph(ctx, client, seed, 10)
	if err != nil { t.Fatal(err) }
	if len(got) == 0 { t.Fatalf("expected discovered users") }
}

func TestRankGraphBoosts(t *testing.T) {
    ctx := context.Background()
    db, _ := sqlitevec.Open(":memory:")
    defer db.Close()
    now := time.Now().UTC()
    // record some interactions by author 'a'
    _ = db.PutEvent(ctx, now.Add(-1*time.Hour), "reply", map[string]any{"author_id":"a"})
    users := []model.User{{ID:"a", Username:"a"}, {ID:"b", Username:"b"}}
    seed := []model.User{{ID:"seed", Username:"seed"}}
    recs := RankGraph(ctx, db, users, seed, []string{"golang"}, map[string]float64{"golang":1})
    if len(recs) < 2 { t.Fatalf("need at least two recs") }
    if recs[0].User.ID != "a" { t.Fatalf("expected 'a' boosted to top, got %s", recs[0].User.ID) }
}
