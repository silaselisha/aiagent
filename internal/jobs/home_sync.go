package jobs

import (
    "context"

    "starseed/internal/config"
    "starseed/internal/store/sqlitevec"
    "starseed/internal/xclient"
    "starseed/internal/model"
)

const homeCursorKey = "home_timeline:since_id"

// homeGetter abstracts v1.1 home paging.
type homeGetter interface { GetHomeTimelineSince(ctx context.Context, sinceID string, limit int) ([]model.Tweet, error) }

// SyncHomeTimeline pages v1.1 home timeline using since_id and stores events idempotently.
func SyncHomeTimeline(ctx context.Context, db *sqlitevec.DB, getter homeGetter, cfg config.Config, perPage int, pages int) error {
    sinceID, _ := db.LoadCursor(ctx, homeCursorKey)
    var maxID string
    for i := 0; i < pages; i++ {
        items, err := getter.GetHomeTimelineSince(ctx, sinceID, perPage)
        if err != nil { break }
        if len(items) == 0 { break }
        for _, t := range items {
            _ = db.PutEventRef(ctx, t.CreatedAt, "home", t.ID, map[string]any{"tweet_id": t.ID, "author_id": t.AuthorID})
            if t.ID > maxID { maxID = t.ID }
        }
        sinceID = maxID
    }
    if maxID != "" { _ = db.SaveCursor(ctx, homeCursorKey, maxID) }
    return nil
}
