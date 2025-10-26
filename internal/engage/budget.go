package engage

import (
	"context"
	"time"

	"starseed/internal/config"
	"starseed/internal/store/sqlitevec"
)

// ShouldAllowEngage checks hourly/daily budgets before engaging.
func ShouldAllowEngage(ctx context.Context, db *sqlitevec.DB, cfg config.EngagementConfig, now time.Time) (bool, error) {
	startHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
	startDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	hourCount, err := db.CountActionsWithin(ctx, startHour, startHour.Add(time.Hour), "engage")
	if err != nil {
		return false, err
	}
	dayCount, err := db.CountActionsWithin(ctx, startDay, startDay.Add(24*time.Hour), "engage")
	if err != nil {
		return false, err
	}
	if cfg.MaxPerHour > 0 && hourCount >= cfg.MaxPerHour {
		return false, nil
	}
	if cfg.MaxPerDay > 0 && dayCount >= cfg.MaxPerDay {
		return false, nil
	}
	return true, nil
}

// RecordEngage logs an engagement action.
func RecordEngage(ctx context.Context, db *sqlitevec.DB, now time.Time) error {
	return db.PutAction(ctx, now, "engage")
}

// ShouldAllowByType enforces per-type budgets if configured.
func ShouldAllowByType(ctx context.Context, db *sqlitevec.DB, cfg config.EngagementConfig, typ string, now time.Time) (bool, error) {
    b, ok := cfg.PerType[typ]
    if !ok { return true, nil }
    startHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
    startDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
    hourCount, err := db.CountActionsWithin(ctx, startHour, startHour.Add(time.Hour), typ)
    if err != nil { return false, err }
    dayCount, err := db.CountActionsWithin(ctx, startDay, startDay.Add(24*time.Hour), typ)
    if err != nil { return false, err }
    if b.MaxPerHour > 0 && hourCount >= b.MaxPerHour { return false, nil }
    if b.MaxPerDay > 0 && dayCount >= b.MaxPerDay { return false, nil }
    return true, nil
}

func RecordByType(ctx context.Context, db *sqlitevec.DB, typ string, now time.Time) error {
    return db.PutAction(ctx, now, typ)
}
