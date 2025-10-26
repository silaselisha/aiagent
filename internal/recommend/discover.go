package recommend

import (
    "context"
    "strings"

    "starseed/internal/config"
    "starseed/internal/model"
    "starseed/internal/xclient"
)

// DiscoverTweetsByInterests queries recent tweets for compound interest keywords.
func DiscoverTweetsByInterests(ctx context.Context, client xclient.XClient, cfg config.Config, limit int) ([]model.Tweet, error) {
	if len(cfg.Interests.Keywords) == 0 {
		return nil, nil
	}
	// Build OR query (url-escaped by client)
	q := strings.Join(cfg.Interests.Keywords, " OR ")
	return client.SearchRecentTweets(ctx, q, limit)
}

// DiscoverAccountsFromTweets extracts unique authors not already followed and returns users.
func DiscoverAccountsFromTweets(ctx context.Context, client xclient.XClient, tweets []model.Tweet, alreadyFollowing map[string]struct{}) ([]model.User, error) {
    ids := make(map[string]struct{})
    for _, t := range tweets {
        if t.AuthorID == "" { continue }
        if _, ok := alreadyFollowing[t.AuthorID]; ok { continue }
        ids[t.AuthorID] = struct{}{}
    }
    if len(ids) == 0 { return nil, nil }
    arr := make([]string, 0, len(ids))
    for id := range ids { arr = append(arr, id) }
    return client.GetUsersByIDs(ctx, arr)
}
