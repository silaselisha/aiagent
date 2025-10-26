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
