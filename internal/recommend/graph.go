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

// DiscoverGraphMultiHop expands follow graph up to given depth (>=1).
func DiscoverGraphMultiHop(ctx context.Context, client xclient.XClient, seed []model.User, depth int, limit int) ([]model.User, error) {
    if depth < 1 { depth = 1 }
    seen := make(map[string]struct{})
    var frontier []model.User = seed
    var collected []model.User
    for _, s := range seed { seen[s.ID] = struct{}{} }
    for d := 0; d < depth && len(collected) < limit; d++ {
        var next []model.User
        for _, u := range frontier {
            follows, err := client.GetFollowing(ctx, u.ID, 200)
            if err != nil { continue }
            for _, v := range follows {
                if _, ok := seen[v.ID]; ok { continue }
                seen[v.ID] = struct{}{}
                collected = append(collected, v)
                next = append(next, v)
                if len(collected) >= limit { break }
            }
            if len(collected) >= limit { break }
        }
        frontier = next
    }
    return collected, nil
}

// GraphParams controls calibration for multi-hop and mutual weighting.
type GraphParams struct {
    MaxDepth          int
    HopWeight         float64 // added as HopWeight * (1/(1+hop))
    MutualWeight      float64 // added per mutual link (seed/frontier connections)
    InteractionWeight float64 // added per past interaction
}

// BuildGraphStats returns candidates plus hop distance and mutual counts.
func BuildGraphStats(ctx context.Context, client xclient.XClient, seed []model.User, depth int, limit int) ([]model.User, map[string]int, map[string]int, error) {
    if depth < 1 { depth = 1 }
    hop := make(map[string]int)
    mutual := make(map[string]int)
    seen := make(map[string]struct{})
    for _, s := range seed { seen[s.ID] = struct{}{} }
    frontier := seed
    var collected []model.User
    for d := 1; d <= depth && len(collected) < limit; d++ {
        var next []model.User
        for _, u := range frontier {
            follows, err := client.GetFollowing(ctx, u.ID, 200)
            if err != nil { continue }
            for _, v := range follows {
                mutual[v.ID]++
                if _, ok := seen[v.ID]; ok { continue }
                seen[v.ID] = struct{}{}
                hop[v.ID] = d
                collected = append(collected, v)
                next = append(next, v)
                if len(collected) >= limit { break }
            }
            if len(collected) >= limit { break }
        }
        frontier = next
    }
    return collected, hop, mutual, nil
}

// RankGraph merges discovered users with scores and returns top candidates, with calibrated boosts.
func RankGraph(ctx context.Context, db *sqlitevec.DB, users []model.User, seed []model.User, keywords []string, weights map[string]float64) []AccountRecommendation {
    // default params when not using BuildGraphStats
    params := GraphParams{MaxDepth: 2, HopWeight: 0.2, MutualWeight: 0.1, InteractionWeight: 0.05}
    return RankGraphCalibrated(ctx, db, params, users, seed, keywords, weights, nil, nil)
}

// RankGraphCalibrated allows passing precomputed hop/mutual stats (e.g., from BuildGraphStats).
func RankGraphCalibrated(ctx context.Context, db *sqlitevec.DB, params GraphParams, users []model.User, seed []model.User, keywords []string, weights map[string]float64, hop map[string]int, mutual map[string]int) []AccountRecommendation {
    base := RankAccounts(users, keywords, weights)
    // Interaction counts over last 7 days
    var counts map[string]int
    if db != nil {
        end := time.Now().UTC()
        start := end.Add(-7 * 24 * time.Hour)
        counts = CountInteractionsByAuthor(ctx, db, start, end)
    }
    for i := range base {
        // hop weighting
        if hop != nil {
            if h, ok := hop[base[i].User.ID]; ok && h > 0 {
                base[i].FinalScore += params.HopWeight * (1.0 / float64(1+h))
            }
        }
        // mutual count weighting
        if mutual != nil {
            if m := mutual[base[i].User.ID]; m > 0 {
                base[i].FinalScore += params.MutualWeight * float64(m)
            }
        }
        // historical interaction weighting
        if n := counts[base[i].User.ID]; n > 0 {
            base[i].FinalScore += params.InteractionWeight * float64(n)
        }
    }
    sort.Slice(base, func(i, j int) bool { return base[i].FinalScore > base[j].FinalScore })
    return base
}
