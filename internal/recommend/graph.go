package recommend

import (
	"context"
	"sort"

	"starseed/internal/model"
	"starseed/internal/xclient"
)

// DiscoverGraph expands accounts by mutual follow edges (up to one hop).
func DiscoverGraph(ctx context.Context, client xclient.XClient, seed []model.User, limit int) ([]model.User, error) {
	seen := make(map[string]struct{})
	for _, u := range seed { seen[u.ID] = struct{}{} }
	var out []model.User
	for _, u := range seed {
		f, err := client.GetFollowing(ctx, u.ID, 200)
		if err != nil { continue }
		for _, v := range f {
			if _, ok := seen[v.ID]; ok { continue }
			seen[v.ID] = struct{}{}
			out = append(out, v)
			if len(out) >= limit { break }
		}
		if len(out) >= limit { break }
	}
	return out, nil
}

// RankGraph merges discovered users with scores and returns top candidates.
func RankGraph(users []model.User, keywords []string, weights map[string]float64) []AccountRecommendation {
	recs := RankAccounts(users, keywords, weights)
	sort.Slice(recs, func(i, j int) bool { return recs[i].FinalScore > recs[j].FinalScore })
	return recs
}
