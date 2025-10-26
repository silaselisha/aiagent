package nn

import (
	"math"

	"starseed/internal/model"
)

// MetaFeatures computes topic relevance stats and bot score histogram.
// Returns [relevanceMean, relevanceVar, botLow, botMid, botHigh].
func MetaFeatures(tweets []model.Tweet, authors map[string]model.User, keywords []string, weights map[string]float64) [5]float32 {
	if len(tweets) == 0 {
		return [5]float32{0, 0, 0, 0, 0}
	}
	rels := make([]float64, 0, len(tweets))
	low, mid, high := 0, 0, 0
	for _, t := range tweets {
		r := model.InterestRelevance(t.Text, keywords, weights)
		rels = append(rels, float64(r))
		if u, ok := authors[t.AuthorID]; ok {
			b := model.BotLikelihood(u)
			switch {
			case b < 0.33:
				low++
			case b < 0.66:
				mid++
			default:
				high++
			}
		}
	}
	mean := 0.0
	for _, v := range rels { mean += v }
	mean /= math.Max(1, float64(len(rels)))
	variance := 0.0
	for _, v := range rels { variance += (v-mean)*(v-mean) }
	variance /= math.Max(1, float64(len(rels)))
	n := float64(len(tweets))
	return [5]float32{float32(mean), float32(variance), float32(float64(low)/n), float32(float64(mid)/n), float32(float64(high)/n)}
}
