#!/usr/bin/env bash
set -euo pipefail

# Basic E2E smoke: analyze, recommend, ingest-events, nn-train-db, engage (dry run)
BIN=${BIN:-./starseed}
CFG=${CFG:-./starseed.yaml}

echo "== analyze =="
$BIN analyze -config "$CFG" -limit 5 || true

echo "== recommend =="
$BIN recommend -config "$CFG" || true

echo "== ingest-events =="
$BIN ingest-events -config "$CFG" -hours 1 || true

echo "== nn-train-db =="
$BIN nn-train-db -config "$CFG" -hours 1 || true

echo "== engage =="
$BIN engage -config "$CFG" || true

echo "E2E smoke done"
