package jobs

import (
	"context"
	"time"

	"starseed/internal/config"
	"starseed/internal/ingest"
    "starseed/internal/metrics"
	"starseed/internal/store/sqlitevec"
	"starseed/internal/xclient"
    "starseed/internal/logging"
)

const cursorKey = "ingest:last_ts"

// RunIngestionOnce fetches engagements since last cursor (or now-horizon), stores events, and backfills labels.
func RunIngestionOnce(ctx context.Context, db *sqlitevec.DB, client xclient.XClient, cfg config.Config, horizon time.Duration) error {
	now := time.Now().UTC()
	since := now.Add(-horizon)
	if v, err := db.LoadCursor(ctx, cursorKey); err == nil && v != "" {
		if ts, err2 := time.Parse(time.RFC3339Nano, v); err2 == nil {
			since = ts
		}
	}
	me, err := client.GetUserByUsername(ctx, cfg.Account.Username)
	if err != nil {
		return err
	}
    start := time.Now()
    metrics.IngestRuns.Inc()
    if err := ingest.IngestEngagements(ctx, db, client, me.ID, cfg.Account.Username, since); err != nil {
        metrics.IngestErrors.Inc()
		return err
	}
	if err := ingest.BackfillLabels(ctx, db, since, now); err != nil {
        metrics.IngestErrors.Inc()
		return err
	}
    _ = db.SaveCursor(ctx, cursorKey, now.Format(time.RFC3339Nano))
    logging.Info("ingest_once", map[string]any{"since": since, "now": now})
    metrics.ObserveIngestDuration(start)
	return nil
}

// RunIngestionLoop runs RunIngestionOnce on a ticker until ctx is cancelled.
func RunIngestionLoop(ctx context.Context, db *sqlitevec.DB, client xclient.XClient, cfg config.Config, horizon, interval time.Duration) error {
    t := time.NewTicker(interval)
	defer t.Stop()
	// run immediately
    _ = RunIngestionOnce(ctx, db, client, cfg, horizon)
	for {
		select {
		case <-ctx.Done():
            logging.Info("ingest_loop_stop", nil)
            return ctx.Err()
		case <-t.C:
            if err := RunIngestionOnce(ctx, db, client, cfg, horizon); err != nil {
                logging.Error("ingest_once_error", map[string]any{"error": err.Error()})
            }
		}
	}
}
