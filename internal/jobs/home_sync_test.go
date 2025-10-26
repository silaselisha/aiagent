package jobs

import (
	"context"
	"testing"
	"time"

	"starseed/internal/config"
	"starseed/internal/model"
	"starseed/internal/store/sqlitevec"
)

type fakeHome struct{ pages [][]model.Tweet }

func (f *fakeHome) GetHomeTimelineSince(ctx context.Context, sinceID string, limit int) ([]model.Tweet, error) {
	if len(f.pages) == 0 { return nil, nil }
	p := f.pages[0]
	f.pages = f.pages[1:]
	return p, nil
}

func TestHomeSyncPaginationAndCursor(t *testing.T) {
	db, _ := sqlitevec.Open(":memory:")
	defer db.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	f := &fakeHome{pages: [][]model.Tweet{
		{{ID:"1", AuthorID:"a", CreatedAt: now.Add(-2*time.Minute)}},
		{{ID:"2", AuthorID:"b", CreatedAt: now.Add(-1*time.Minute)}},
	}}
	cfg := config.Default()
	if err := SyncHomeTimeline(ctx, db, f, cfg, 10, 5); err != nil { t.Fatal(err) }
	// cursor should be max id "2"
	v, err := db.LoadCursor(ctx, homeCursorKey)
	if err != nil || v != "2" { t.Fatalf("cursor expected 2, got %s err=%v", v, err) }
}
