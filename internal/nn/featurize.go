package nn

import (
    "math"
    "time"

    "starseed/internal/model"
)

// FeatureVector represents a single 15-min window features and target.
type FeatureVector struct {
	X []float32 `json:"x"`
	Y []float32 `json:"y"`
}

// BuildFeatures constructs features from tweets and events in a 15-min window.
// This is a first-cut, simple set we can iterate on.
func BuildFeatures(windowStart time.Time, tweets []model.Tweet, events []model.EngagementEvent) FeatureVector {
	var x []float32
	var y []float32

    // Features: volume and engagement statistics within window
	var count, likes, replies, retweets, quotes int
	for _, t := range tweets {
		if t.CreatedAt.Before(windowStart) || t.CreatedAt.After(windowStart.Add(15*time.Minute)) { continue }
		count++
		likes += t.LikeCount
		replies += t.ReplyCount
		retweets += t.RetweetCount
		quotes += t.QuoteCount
	}
	avgLikes := 0.0
	if count > 0 { avgLikes = float64(likes) / float64(count) }
	avgReplies := 0.0
	if count > 0 { avgReplies = float64(replies) / float64(count) }
	avgRetweets := 0.0
	if count > 0 { avgRetweets = float64(retweets) / float64(count) }

    // Normalize with simple log scaling to control magnitude
	x = append(x, float32(math.Log1p(float64(count))))
	x = append(x, float32(math.Log1p(float64(likes))))
	x = append(x, float32(math.Log1p(float64(replies))))
	x = append(x, float32(math.Log1p(float64(retweets))))
	x = append(x, float32(math.Log1p(float64(quotes))))
	x = append(x, float32(avgLikes))
	x = append(x, float32(avgReplies))
	x = append(x, float32(avgRetweets))

    // Rolling aggregates over previous N windows (N=4 => past hour)
    // Note: caller should provide tweets aggregated; simple proxies here using current window stats as placeholders for demo.
    // These are slots to be filled with true rolling stats when history is available.
    for i := 0; i < 4; i++ { // placeholders
        x = append(x, float32(math.Log1p(float64(count))))
        x = append(x, float32(avgLikes))
    }

    // Time-of-week encoding (hour of week as sin/cos)
    dow := int(windowStart.Weekday())
    how := dow*24 + windowStart.Hour()
    angle := 2 * math.Pi * float64(how) / (7.0 * 24.0)
    x = append(x, float32(math.Sin(angle)))
    x = append(x, float32(math.Cos(angle)))

    // Targets: next-window desired engagement proxy (e.g., replies we aim to elicit)
	var futureReplies int
	for _, e := range events {
		if e.Timestamp.After(windowStart.Add(15*time.Minute)) && e.Timestamp.Before(windowStart.Add(30*time.Minute)) {
			if e.Type == "reply" { futureReplies++ }
		}
	}
	y = append(y, float32(math.Log1p(float64(futureReplies))))

	return FeatureVector{X: x, Y: y}
}
