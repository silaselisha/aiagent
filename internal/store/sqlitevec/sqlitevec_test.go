package sqlitevec

import (
	"context"
	"testing"
	"time"
)

func TestCursorsAndActions(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	ctx := context.Background()
	if err := db.SaveCursor(ctx, "ingest:likes", "123"); err != nil { t.Fatal(err) }
	v, err := db.LoadCursor(ctx, "ingest:likes")
	if err != nil || v != "123" { t.Fatalf("cursor mismatch: %v %s", err, v) }
	if err := db.PutAction(ctx, time.Now().UTC(), "reply"); err != nil { t.Fatal(err) }
	n, err := db.CountActionsWithin(ctx, time.Now().UTC().Add(-time.Hour), time.Now().UTC().Add(time.Hour), "reply")
	if err != nil || n != 1 { t.Fatalf("action count mismatch: %v %d", err, n) }
}
