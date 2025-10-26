# syntax=docker/dockerfile:1

# ----------- builder: go -----------
FROM golang:1.24 AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# static-ish build; modernc.org/sqlite is pure Go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/starseed ./cmd/starseed

# ----------- builder: rust -----------
FROM rust:1.82 AS rust-builder
WORKDIR /src
COPY starseed-nn/ ./starseed-nn/
WORKDIR /src/starseed-nn
RUN cargo build --release && install -m 0755 target/release/starseed-nn /out/starseed-nn

# ----------- runner -----------
FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY --from=go-builder /out/starseed /usr/local/bin/starseed
COPY --from=rust-builder /out/starseed-nn /usr/local/bin/starseed-nn
# Default DB path in container
ENV STARSEED_DB=/data/starseed.db
VOLUME ["/data"]
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/starseed"]
CMD ["help"]
