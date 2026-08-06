.PHONY: build test tidy run-worker run-crawler docker-build clean

APP_WORKER := bin/worker
APP_CRAWLER := bin/crawler

build:
	mkdir -p bin
	go build -o $(APP_WORKER) ./cmd/worker
	go build -o $(APP_CRAWLER) ./cmd/crawler

test:
	go test ./...

tidy:
	go mod tidy

run-worker: build
	./$(APP_WORKER)

run-crawler: build
	./$(APP_CRAWLER) -seed data/seed

docker-build:
	docker build -t cloud-search-ingestion:local .

clean:
	rm -rf bin
