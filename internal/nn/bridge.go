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
