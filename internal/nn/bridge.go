package nn

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

// Train calls the Rust binary with JSONL samples to produce a model file.
func Train(binaryPath, outPath string, samples []FeatureVector, hidden, epochs int, lr float32) error {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	enc := json.NewEncoder(w)
	for _, s := range samples {
		if err := enc.Encode(s); err != nil { return err }
	}
	_ = w.Flush()
    cmd := exec.Command(binaryPath, "train", "--out", outPath, "--hidden", fmt.Sprint(hidden), "--epochs", fmt.Sprint(epochs), "--lr", fmt.Sprint(lr))
	cmd.Stdin = &buf
	out, err := cmd.CombinedOutput()
	if err != nil { return fmt.Errorf("train error: %v: %s", err, string(out)) }
	return nil
}

type TrainOptions struct {
    Hidden     int
    Epochs     int
    LR         float32
    ValSplit   float32
    Patience   int
    Calibrate  bool
    Checkpoint string
}

// TrainWithOptions calls the Rust trainer with advanced options.
func TrainWithOptions(binaryPath, outPath string, samples []FeatureVector, opts TrainOptions) error {
    var buf bytes.Buffer
    w := bufio.NewWriter(&buf)
    enc := json.NewEncoder(w)
    for _, s := range samples {
        if err := enc.Encode(s); err != nil { return err }
    }
    _ = w.Flush()
    args := []string{"train", "--out", outPath, "--hidden", fmt.Sprint(opts.Hidden), "--epochs", fmt.Sprint(opts.Epochs), "--lr", fmt.Sprint(opts.LR), "--val-split", fmt.Sprint(opts.ValSplit), "--patience", fmt.Sprint(opts.Patience)}
    if opts.Checkpoint != "" { args = append(args, "--checkpoint", opts.Checkpoint) }
    if opts.Calibrate { args = append(args, "--calibrate") }
    cmd := exec.Command(binaryPath, args...)
    cmd.Stdin = &buf
    out, err := cmd.CombinedOutput()
    if err != nil { return fmt.Errorf("train error: %v: %s", err, string(out)) }
    return nil
}

// LoadThresholdFromModel reads threshold from saved model JSON file.
func LoadThresholdFromModel(modelPath string) float32 {
    var tmp struct{ Threshold float32 `json:"threshold"` }
    // Attempt to read file
    out, err := exec.Command("bash", "-lc", fmt.Sprintf("cat %q", modelPath)).Output()
    if err != nil { return 0 }
    if err := json.Unmarshal(out, &tmp); err != nil { return 0 }
    return tmp.Threshold
}

// Infer calls the Rust binary to get predictions for samples.
func Infer(binaryPath, modelPath string, samples []FeatureVector) ([][]float32, error) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	enc := json.NewEncoder(w)
	for _, s := range samples {
		if err := enc.Encode(s); err != nil { return nil, err }
	}
	_ = w.Flush()
	cmd := exec.Command(binaryPath, "infer", "--model", modelPath)
	cmd.Stdin = &buf
	out, err := cmd.Output()
	if err != nil { return nil, err }
	// Parse line-delimited arrays
	scanner := bufio.NewScanner(bytes.NewReader(out))
	var preds [][]float32
	for scanner.Scan() {
		var arr []float32
		if err := json.Unmarshal(scanner.Bytes(), &arr); err != nil { return nil, err }
		preds = append(preds, arr)
	}
	return preds, nil
}
