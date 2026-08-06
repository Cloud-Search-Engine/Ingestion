# syntax=docker/dockerfile:1

FROM golang:1.22-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/crawler ./cmd/crawler

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /out/worker /app/worker
COPY --from=builder /out/crawler /app/crawler
COPY data/seed /app/data/seed

USER nonroot:nonroot
ENTRYPOINT ["/app/worker"]
