package recommend

import (
	"context"
	"sort"

	"starseed/internal/model"
	"starseed/internal/xclient"
    "starseed/internal/store/sqlitevec"
    "time"
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

// RankGraph merges discovered users with scores and returns top candidates, with boosts for mutuals and interaction frequency.
func RankGraph(ctx context.Context, db *sqlitevec.DB, users []model.User, seed []model.User, keywords []string, weights map[string]float64) []AccountRecommendation {
    base := RankAccounts(users, keywords, weights)
    // Build mutuals set
    seedIDs := make(map[string]struct{})
    for _, s := range seed { seedIDs[s.ID] = struct{}{} }
    // Interaction counts over last 7 days
    var counts map[string]int
    if db != nil {
        end := time.Now().UTC()
        start := end.Add(-7 * 24 * time.Hour)
        counts = CountInteractionsByAuthor(ctx, db, start, end)
    }
    for i := range base {
        if _, ok := seedIDs[base[i].User.ID]; ok {
            base[i].FinalScore += 0.1
        }
        if n := counts[base[i].User.ID]; n > 0 {
            base[i].FinalScore += 0.05 * float64(n)
        }
    }
    sort.Slice(base, func(i, j int) bool { return base[i].FinalScore > base[j].FinalScore })
    return base
}
