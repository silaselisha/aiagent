package engage

import (
    "context"
    "encoding/json"
    "os"
)

// LoadThreshold loads calibration threshold from model file; if missing, returns 0.
func LoadThreshold(modelPath string) float32 {
	b, err := os.ReadFile(modelPath)
	if err != nil { return 0 }
	var tmp struct{ Threshold float32 `json:"threshold"` }
	_ = json.Unmarshal(b, &tmp)
	return tmp.Threshold
}

// ShouldEngage decides based on predicted value and threshold.
func ShouldEngage(ctx context.Context, modelThreshold float32, preds [][]float32) bool {
	if len(preds) == 0 || len(preds[0]) == 0 { return false }
	p := preds[0][0]
	return p >= modelThreshold
}
