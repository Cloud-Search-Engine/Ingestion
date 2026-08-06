package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/chunker"
	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/embedder"
	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/models"
	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/parser"
	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/storage"
)

// Processor runs the ingest pipeline for a single SQS message.
type Processor struct {
	S3        *storage.S3Store
	DB        *storage.Postgres
	Embedder  embedder.Embedder
	ChunkSize int
	Log       *slog.Logger
}

// Process downloads, parses, chunks, embeds, and upserts a document. Idempotent on document_id.
func (p *Processor) Process(ctx context.Context, msg models.IngestMessage) error {
	if msg.DocumentID == "" {
		return fmt.Errorf("ingest message missing document_id")
	}
	if msg.S3Key == "" {
		return fmt.Errorf("ingest message missing s3_key")
	}

	log := p.Log
	if log == nil {
		log = slog.Default()
	}

	body, ct, err := p.S3.GetObject(ctx, msg.S3Bucket, msg.S3Key)
	if err != nil {
		return err
	}
	if msg.ContentType == "" {
		msg.ContentType = ct
	}
	if msg.ContentType == "" {
		msg.ContentType = "text/markdown"
	}

	parsed, err := parser.Parse(body, msg.ContentType, msg.Title)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	if msg.Title == "" {
		msg.Title = parsed.Title
	}

	chunks := chunker.Chunk(msg.DocumentID, msg, parsed, chunker.Options{
		SizeTokens: p.ChunkSize,
		Overlap:    p.ChunkSize / 8,
	})

	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Content
	}
	vectors, err := p.Embedder.Embed(ctx, texts)
	if err != nil {
		return fmt.Errorf("embed: %w", err)
	}
	for i := range chunks {
		if i < len(vectors) {
			chunks[i].Embedding = vectors[i]
		}
	}

	doc := models.Document{
		DocumentID:   msg.DocumentID,
		Provider:     msg.Provider,
		Service:      msg.Service,
		Category:     msg.Category,
		DocumentType: msg.DocumentType,
		Title:        msg.Title,
		SourceURL:    msg.SourceURL,
		Version:      msg.Version,
		Content:      parsed.Content,
		S3Key:        msg.S3Key,
	}

	if err := p.DB.UpsertDocument(ctx, doc, chunks); err != nil {
		return err
	}
	if err := p.DB.RebuildTermStats(ctx); err != nil {
		return fmt.Errorf("rebuild term stats: %w", err)
	}

	log.Info("ingested document",
		"document_id", msg.DocumentID,
		"chunks", len(chunks),
		"provider", msg.Provider,
		"service", msg.Service,
	)
	return nil
}

// ParseMessage unmarshals an SQS body into IngestMessage.
func ParseMessage(body string) (models.IngestMessage, error) {
	var msg models.IngestMessage
	if err := json.Unmarshal([]byte(body), &msg); err != nil {
		return models.IngestMessage{}, fmt.Errorf("decode ingest message: %w", err)
	}
	return msg, nil
}
