# CloudSearch Ingestion

Go crawler + SQS worker that turns official documentation into searchable chunks (and embeddings) in PostgreSQL.

```
Docs → crawler → S3 → SQS → worker → parse → chunk → embed → PostgreSQL + pgvector
```

Sibling repos: **Database** (schema), **Backend** (search), **Terraform** (S3/SQS on AWS), **Kubernetes** (worker Deployment).

## What’s in this repo

| Path | Purpose |
| --- | --- |
| `cmd/crawler` | Load seed files / URLs → upload S3 → enqueue SQS |
| `cmd/worker` | SQS consumer with worker pool + graceful shutdown |
| `internal/crawler` | Fetch / load seed markdown |
| `internal/parser` | HTML/markdown → clean text + headings |
| `internal/chunker` | Token-window chunks (256 / 512 / 1024) |
| `internal/embedder` | `noop` (deterministic) + OpenAI stub |
| `internal/queue` | LocalStack-compatible SQS client |
| `internal/storage` | S3 + Postgres upsert / BM25 term stats |
| `internal/pipeline` | End-to-end process for one message |
| `data/seed/` | 8 sample docs (SQS, Service Bus, Pub/Sub, EKS, AKS, GKE, S3, Blob) |
| `Dockerfile` | Builds both `worker` and `crawler` binaries |

## Prerequisites

- Go **1.22+**
- Postgres with **Database** migrations applied
- S3 + SQS (LocalStack locally, real AWS in prod)
- Env: `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` (`test`/`test` for LocalStack)

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `DATABASE_URL` | local cloudsearch DSN | Postgres |
| `AWS_ENDPOINT_URL` | _(empty)_ | LocalStack `http://localhost:4566` |
| `AWS_ENDPOINT_URL_S3` | same as above | S3 endpoint override |
| `AWS_REGION` | `us-east-1` | Region |
| `S3_BUCKET` | `cloudsearch-docs` | Raw docs bucket |
| `SQS_QUEUE_URL` | _(required)_ | Ingestion queue URL |
| `EMBEDDING_PROVIDER` | `noop` | `noop` or `openai` |
| `WORKER_CONCURRENCY` | `4` | Parallel handlers |
| `CHUNK_SIZE_TOKENS` | `512` | Chunk size |
| `SEED_DIR` | `data/seed` | Seed directory |

## How to start (with LocalStack)

```bash
# Infra (from parent Cloud_Search_Engine folder is easiest):
#   docker compose up -d postgres localstack
# Or create bucket/queue yourself against LocalStack.

export AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test
export AWS_REGION=us-east-1
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ENDPOINT_URL_S3=http://localhost:4566
export S3_BUCKET=cloudsearch-docs
export SQS_QUEUE_URL=http://localhost:4566/000000000000/cloudsearch-ingestion
export DATABASE_URL=postgres://cloudsearch:cloudsearch@localhost:5432/cloudsearch?sslmode=disable
export EMBEDDING_PROVIDER=noop

go mod tidy
make build

# Terminal A — worker
./bin/worker   # or: go run ./cmd/worker

# Terminal B — seed / crawl
go run ./cmd/crawler -seed ./data/seed
```

## How to start (full local stack)

```bash
# from parent Cloud_Search_Engine folder
docker compose up --build -d
docker compose --profile seed run --rm crawler
```

## Tests

```bash
make test
# or: go test ./...
```
