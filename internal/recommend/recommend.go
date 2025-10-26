package recommend

import (
	"sort"

	"starseed/internal/model"
)

// AccountRecommendation bundles a user with scores.
type AccountRecommendation struct {
	User           model.User
	OrganicScore   float64
	BotLikelihood  float64
	RelevanceScore float64
	FinalScore     float64
}

// RankAccounts ranks users to follow based on heuristic scores.
func RankAccounts(users []model.User, keywords []string, weights map[string]float64) []AccountRecommendation {
	recs := make([]AccountRecommendation, 0, len(users))
	for _, u := range users {
		org := 0.5
		bot := model.BotLikelihood(u)
		text := u.Description + " " + u.Name
		rel := model.InterestRelevance(text, keywords, weights)
		final := rel*0.6 + org*0.2 + (1-bot)*0.2
		recs = append(recs, AccountRecommendation{User: u, OrganicScore: org, BotLikelihood: bot, RelevanceScore: rel, FinalScore: final})
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].FinalScore > recs[j].FinalScore })
	return recs
}
