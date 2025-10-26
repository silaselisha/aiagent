package nn

import (
	"context"
	"fmt"
	"time"

	"starseed/internal/store/sqlitevec"
)

// TrainFromDB loads labeled windows from SQLite and trains the model.
func TrainFromDB(ctx context.Context, db *sqlitevec.DB, start, end time.Time, binPath, outPath string) error {
	ts, X, y, err := db.LoadFeatures(ctx, start, end)
	if err != nil { return err }
	var samples []FeatureVector
	for i := range ts {
		if y[i] < 0 { continue }
		samples = append(samples, FeatureVector{X: X[i], Y: []float32{y[i]}})
	}
	if len(samples) == 0 { return fmt.Errorf("no labeled samples") }
    opts := TrainOptions{Hidden: 64, Epochs: 10, LR: 0.01, ValSplit: 0.2, Patience: 3, Calibrate: true, Checkpoint: outPath}
    if err := TrainWithOptions(binPath, outPath, samples, opts); err != nil { return err }
    // Load threshold from model file and save to DB calibration for engage
    thr := LoadThresholdFromModel(outPath)
    if thr > 0 {
        _ = db.SaveThreshold(ctx, float64(thr))
    }
    return nil
}
