package engage

import (
	"context"
	"testing"
	"time"

	"starseed/internal/config"
	"starseed/internal/store/sqlitevec"
)

func TestShouldAllowEngageRespectsBudgets(t *testing.T) {
	db, err := sqlitevec.Open(":memory:")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	ctx := context.Background()
	now := time.Date(2025,1,1,12,0,0,0,time.UTC)
	cfg := config.EngagementConfig{MaxPerHour: 2, MaxPerDay: 3}
	// No actions yet
	ok, err := ShouldAllowEngage(ctx, db, cfg, now)
	if err != nil || !ok { t.Fatalf("expected allowed, got %v %v", ok, err) }
	// Record two actions in hour
	_ = RecordEngage(ctx, db, now)
	_ = RecordEngage(ctx, db, now.Add(5*time.Minute))
	ok, _ = ShouldAllowEngage(ctx, db, cfg, now.Add(10*time.Minute))
	if ok { t.Fatalf("expected blocked by hourly budget") }
	// Another action next hour, but daily limit 3 blocks
	_ = RecordEngage(ctx, db, now.Add(65*time.Minute))
	ok, _ = ShouldAllowEngage(ctx, db, cfg, now.Add(70*time.Minute))
	if ok { t.Fatalf("expected blocked by daily budget") }
}
