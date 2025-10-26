package model

import (
	"math"
	"strings"

	"starseed/internal/util"
)

// OrganicContentScore estimates how organic a tweet appears.
// Heuristics: no excessive links, balanced engagement, non-spammy tokens.
func OrganicContentScore(t Tweet) float64 {
	score := 0.5
	if !t.HasLink {
		score += 0.2
	}
	// Penalize extremely low or extremely high engagement ratios for spam/viral baits
	total := t.LikeCount + t.ReplyCount + t.RetweetCount + t.QuoteCount
	if total == 0 {
		score += 0.05
	} else {
		ratio := float64(t.ReplyCount+ t.QuoteCount) / float64(total)
		if ratio >= 0.15 && ratio <= 0.55 {
			score += 0.15
		}
	}
	// Penalize spammy tokens
	spammy := []string{"giveaway", "win big", "click here", "promo", "ref code"}
	if util.ContainsAnyCaseInsensitive(t.Text, spammy) {
		score -= 0.25
	}
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return math.Round(score*100) / 100
}

// BotLikelihood estimates if a user is a bot [0,1]. Lower is better.
func BotLikelihood(u User) float64 {
	score := 0.2
	if u.DefaultImage || u.DefaultProfile {
		score += 0.2
	}
	if !u.Verified && u.FollowersCount < 50 && u.FollowingCount > 500 {
		score += 0.3
	}
	if strings.TrimSpace(u.Description) == "" {
		score += 0.1
	}
	if score > 1 {
		score = 1
	}
	return math.Round(score*100) / 100
}

// InterestRelevance scores how relevant text is to our interests.
func InterestRelevance(text string, keywords []string, weights map[string]float64) float64 {
	tokens := util.Tokenize(text)
	if len(tokens) == 0 || len(keywords) == 0 {
		return 0
	}
	kw := make(map[string]float64)
	for _, k := range keywords {
		w := 1.0
		if v, ok := weights[strings.ToLower(k)]; ok {
			w = v
		}
		kw[strings.ToLower(k)] = w
	}
	sum := 0.0
	for _, t := range tokens {
		if w, ok := kw[t]; ok {
			sum += w
		}
	}
	// Normalize roughly by token count
	norm := sum / (float64(len(tokens)) + 1)
	if norm > 1 {
		norm = 1
	}
	return math.Round(norm*100) / 100
}
