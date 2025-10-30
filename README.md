# Starseed (Retro-Anime Intergalactic X Navigator)

Starseed is a Go CLI that analyzes your X (Twitter) timeline, recommends high-quality accounts to interact with, and suggests wise replies to grow organically. It includes a Rust-based neural model for 15‑minute window forecasting, an ingestion loop with rate-limited API access, a SQLite vector store, JSON logs, and Prometheus metrics.

## Features
- Timeline ingestion
  - v1.1 home timeline (OAuth 1.0a) with since_id paging; followings proxy fallback
  - v2 endpoints for followings, user tweets, recent search, mentions, liked, quote_tweets
  - Adaptive retry/backoff (Retry-After, 5xx, jitter); per-endpoint retry metrics
- Engagement ingestion with cursors and idempotency
  - Likes, replies (to:me), outbound retweets (from:me is:retweet), quotes of our posts
  - Next-window label backfill for 15‑minute windows
- Modeling
  - Rust MLP (val split, early stop, calibration); train from raw or DB (nn-train-db)
  - DB threshold persistence; engage uses DB threshold and budgets
- Recommendations
  - Rank existing followings; interest-based discovery (tweets -> authors)
  - Graph multi-hop expansion with mutual/interaction weighting
- Observability
  - JSON logs; Prometheus /metrics and /health
- Deployment
  - Dockerfile (Go+Rust), docker-compose, K8s manifests (Deployment/Service/Secret)

## Quick start
```bash
# Build binaries
(go build ./cmd/starseed) && (cd starseed-nn && cargo build --release)

# Create a config
yes | ./starseed init -path ./starseed.yaml

# Set credentials (examples)
export X_BEARER_TOKEN=...
# Optional v1.1 Home timeline
export X_CONSUMER_KEY=...
export X_CONSUMER_SECRET=...
export X_ACCESS_TOKEN=...
export X_ACCESS_SECRET=...
# Optional LLM
export OPENAI_API_KEY=...

# Analyze timeline (v1.1 if OAuth set, fallback to proxy)
./starseed analyze -config ./starseed.yaml -limit 100

# Recommend accounts (existing + new via interests + graph)
./starseed recommend -config ./starseed.yaml

# Ingest engagements and backfill labels (one-shot)
./starseed ingest-events -config ./starseed.yaml -hours 6

# Train from DB (last 24h labeled windows)
./starseed nn-train-db -config ./starseed.yaml -hours 24

# Suggest wise replies (threshold+budgets)
./starseed engage -config ./starseed.yaml
```

## Metrics & health
- Enable metrics by setting `METRICS_ADDR=:9090` (or any addr)
- Scrape `/metrics` (Prometheus format). Health at `/health`.
- Key metrics:
  - `starseed_ingest_runs_total`, `starseed_ingest_errors_total`
  - `starseed_ingest_duration_seconds`
  - `starseed_api_retries_total{endpoint=...}`

## Docker/Compose
```bash
docker compose build
METRICS_ADDR=:9090 X_BEARER_TOKEN=... docker compose up -d
```
Prometheus can scrape the `starseed-metrics` service on port 9090 if deployed via K8s.

## Kubernetes
```bash
kubectl apply -f k8s/namespace.yaml
kubectl -n starseed apply -f k8s/secret.example.yaml   # fill secrets first
kubectl -n starseed apply -f k8s/deployment.yaml
kubectl -n starseed apply -f k8s/service.yaml
```
- Secrets: `X_BEARER_TOKEN`, `X_CONSUMER_KEY`, `X_CONSUMER_SECRET`, `X_ACCESS_TOKEN`, `X_ACCESS_SECRET`, `OPENAI_API_KEY`

## E2E smoke
```bash
BIN=./starseed CFG=./starseed.yaml ./scripts/e2e.sh
```

## Configuration
Edit `starseed.yaml`:
- `account.username`: your X handle (without @)
- `credentials`: tokens/keys (env overrides available)
- `interests`: topics/keywords/weights for relevance
- `filters`: organic score/bot threshold/languages
- `engagement`: quiet hours and budgets (hour/day)
- `storage.dbPath`: SQLite location (default `./starseed.db`)
- `llm`: provider/model/API key (optional)

## Safety & rate hygiene
- Threshold gating and budgets prevent over-engagement
- Adaptive backoff and per-endpoint retry metrics
- JSON logs for auditing; no auto-follow/auto-reply by default (tool suggests only)

## Roadmap (production polish)
- Home timeline: stronger cursoring & dup handling; pagination tests
- Ingestion: inbound retweets/quotes precise attribution; outbound actions
- Observability: logs across all commands; success-rate metrics; tracing
- Budgets: per-action-type policies and acceptance tests
- Trainer: model/threshold versioning; periodic retrain jobs
- Recommendation: multi-hop weighting calibration; richer graph features

## License
MIT
cl/id-LVlTb2l6UUdvUnRJZjRoWkdJM0k6MTpjaQ
cl/s-h2i6qMesOmgqWFsOpQ9n9xUzbZ8-fbGKAjtFhXQ6mkeUcuT8Wb