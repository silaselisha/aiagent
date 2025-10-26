package ingest

import (
	"context"
	"sort"

	"starseed/internal/model"
	"starseed/internal/xclient"
)

// FromFollowing fetches recent tweets from a set of followings (perUserLimit each),
// merges them, and returns up to totalLimit tweets.
func FromFollowing(ctx context.Context, client xclient.XClient, following []model.User, perUserLimit, totalLimit int) ([]model.Tweet, error) {
	var all []model.Tweet
	for _, u := range following {
		ts, err := client.GetUserTweets(ctx, u.ID, perUserLimit)
		if err != nil { continue }
		all = append(all, ts...)
		if len(all) >= totalLimit {
			break
		}
	}
	// Sort by created_at desc if available
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.After(all[j].CreatedAt) })
	if len(all) > totalLimit { all = all[:totalLimit] }
	return all, nil
}
