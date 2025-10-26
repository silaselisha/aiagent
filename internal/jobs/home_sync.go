package jobs

import (
    "context"

    "starseed/internal/config"
    "starseed/internal/store/sqlitevec"
    "starseed/internal/xclient"
)

const homeCursorKey = "home_timeline:since_id"

// SyncHomeTimeline pages v1.1 home timeline using since_id and stores features.
func SyncHomeTimeline(ctx context.Context, db *sqlitevec.DB, v1 *xclient.V1Client, cfg config.Config, perPage int, pages int) error {
	sinceID, _ := db.LoadCursor(ctx, homeCursorKey)
	var maxID string
	for i := 0; i < pages; i++ {
		items, err := v1.GetHomeTimelineSince(ctx, sinceID, perPage)
		if err != nil {
			break
		}
		if len(items) == 0 {
			break
		}
		for _, t := range items {
			_ = db.PutEvent(ctx, t.CreatedAt, "home", map[string]any{"tweet_id": t.ID, "author_id": t.AuthorID})
			if t.ID > maxID { maxID = t.ID }
		}
		// advance since_id for next page
		sinceID = maxID
	}
	if maxID != "" {
		_ = db.SaveCursor(ctx, homeCursorKey, maxID)
	}
	return nil
}
