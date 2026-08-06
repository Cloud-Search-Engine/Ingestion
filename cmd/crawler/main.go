package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/config"
	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/crawler"
	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/models"
	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/queue"
	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/storage"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	seedDir := flag.String("seed", "", "directory of local seed markdown files (overrides SEED_DIR)")
	urlsFile := flag.String("urls", "", "optional file with one URL per line to crawl")
	dryRun := flag.Bool("dry-run", false, "parse seeds and print messages without uploading")
	rps := flag.Float64("rps", 2, "max fetch requests per second")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config", "error", err)
		os.Exit(1)
	}
	if *seedDir != "" {
		cfg.SeedDir = *seedDir
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	crawl := crawler.New(*rps)

	var docs []*crawler.FetchedDoc

	if _, err := os.Stat(cfg.SeedDir); err == nil {
		seedDocs, err := crawl.LoadSeedDir(cfg.SeedDir)
		if err != nil {
			logger.Error("load seeds", "error", err)
			os.Exit(1)
		}
		docs = append(docs, seedDocs...)
		logger.Info("loaded seed files", "count", len(seedDocs), "dir", cfg.SeedDir)
	}

	if *urlsFile != "" {
		urlDocs, err := loadURLs(ctx, crawl, *urlsFile)
		if err != nil {
			logger.Error("crawl urls", "error", err)
			os.Exit(1)
		}
		docs = append(docs, urlDocs...)
	}

	if len(docs) == 0 {
		logger.Error("no documents to ingest; provide -seed dir or -urls file")
		os.Exit(1)
	}

	if *dryRun {
		for _, d := range docs {
			msg := toIngestMessage(d, cfg.S3Bucket)
			b, _ := json.MarshalIndent(msg, "", "  ")
			fmt.Println(string(b))
		}
		return
	}

	if cfg.SQSQueueURL == "" {
		logger.Error("SQS_QUEUE_URL is required (unless --dry-run)")
		os.Exit(1)
	}

	s3Store, err := storage.NewS3(ctx, storage.S3Options{
		Region:      cfg.AWSRegion,
		EndpointURL: cfg.AWSEndpointURLS3,
		Bucket:      cfg.S3Bucket,
	})
	if err != nil {
		logger.Error("s3", "error", err)
		os.Exit(1)
	}

	sqsClient, err := queue.New(ctx, queue.Options{
		Region:      cfg.AWSRegion,
		EndpointURL: cfg.AWSEndpointURL,
		QueueURL:    cfg.SQSQueueURL,
	})
	if err != nil {
		logger.Error("sqs", "error", err)
		os.Exit(1)
	}

	for _, d := range docs {
		msg := toIngestMessage(d, cfg.S3Bucket)
		key := msg.S3Key

		if err := s3Store.PutObject(ctx, key, msg.ContentType, d.Body); err != nil {
			logger.Error("s3 upload", "document_id", msg.DocumentID, "error", err)
			os.Exit(1)
		}

		body, err := json.Marshal(msg)
		if err != nil {
			logger.Error("marshal", "error", err)
			os.Exit(1)
		}
		id, err := sqsClient.SendMessage(ctx, string(body))
		if err != nil {
			logger.Error("sqs send", "document_id", msg.DocumentID, "error", err)
			os.Exit(1)
		}
		logger.Info("enqueued", "document_id", msg.DocumentID, "s3_key", key, "message_id", id)
	}

	logger.Info("crawler finished", "documents", len(docs))
}

func loadURLs(ctx context.Context, crawl *crawler.Crawler, path string) ([]*crawler.FetchedDoc, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var docs []*crawler.FetchedDoc
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		doc, err := crawl.FetchURL(ctx, line)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", line, err)
		}
		base := filepath.Base(strings.TrimSuffix(line, "/"))
		doc.Meta.DocumentID = "url-" + slug(base)
		doc.Meta.Provider = "unknown"
		doc.Meta.Service = "unknown"
		doc.Meta.Category = "documentation"
		doc.Meta.DocumentType = "developer_documentation"
		doc.Meta.Version = "current"
		docs = append(docs, doc)
	}
	return docs, nil
}

func toIngestMessage(d *crawler.FetchedDoc, bucket string) models.IngestMessage {
	meta := d.Meta
	if meta.DocumentID == "" {
		meta.DocumentID = "doc-" + slug(meta.Title)
	}
	if meta.Provider == "" {
		meta.Provider = "unknown"
	}
	if meta.Service == "" {
		meta.Service = "unknown"
	}
	if meta.Category == "" {
		meta.Category = "documentation"
	}
	if meta.DocumentType == "" {
		meta.DocumentType = "developer_documentation"
	}
	if meta.Version == "" {
		meta.Version = "current"
	}
	if meta.Title == "" {
		meta.Title = meta.DocumentID
	}

	key := fmt.Sprintf("raw/%s/%s", meta.Provider, meta.DocumentID)
	ext := ".bin"
	switch {
	case strings.Contains(d.ContentType, "markdown"):
		ext = ".md"
	case strings.Contains(d.ContentType, "html"):
		ext = ".html"
	case strings.Contains(d.ContentType, "plain"):
		ext = ".txt"
	}
	key += ext

	return models.IngestMessage{
		DocumentID:   meta.DocumentID,
		S3Bucket:     bucket,
		S3Key:        key,
		Provider:     meta.Provider,
		Service:      meta.Service,
		Category:     meta.Category,
		DocumentType: meta.DocumentType,
		Title:        meta.Title,
		SourceURL:    meta.SourceURL,
		Version:      meta.Version,
		ContentType:  d.ContentType,
	}
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prev := false
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			prev = false
			continue
		}
		if !prev {
			b.WriteByte('-')
			prev = true
		}
	}
	return strings.Trim(b.String(), "-")
}
