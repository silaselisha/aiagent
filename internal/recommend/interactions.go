package recommend

import (
	"context"
	"encoding/json"
	"time"

	"starseed/internal/store/sqlitevec"
)

// CountInteractionsByAuthor returns counts of events (reply, quote, like) by author_id within range.
func CountInteractionsByAuthor(ctx context.Context, db *sqlitevec.DB, start, end time.Time) map[string]int {
	counts := make(map[string]int)
	evts, err := db.LoadEventsRange(ctx, start, end, "")
	if err != nil { return counts }
	for _, e := range evts {
		var p struct{ AuthorID string `json:"author_id"` }
		_ = json.Unmarshal([]byte(e.Payload), &p)
		if p.AuthorID == "" { continue }
		counts[p.AuthorID]++
	}
	return counts
}
