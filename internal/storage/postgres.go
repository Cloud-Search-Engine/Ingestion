package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/models"
	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/tokenizer"
)

// Postgres stores documents, chunks, embeddings, and BM25 term stats.
type Postgres struct {
	pool *pgxpool.Pool
}

// NewPostgres wraps an existing pool.
func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

// ConnectPool opens a pgx pool with sane defaults.
func ConnectPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

// EnsureSchema adds pgvector support and an embedding column when missing.
// Core tables are expected from Backend migrations; this is additive.
func (p *Postgres) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE EXTENSION IF NOT EXISTS vector`,
		`ALTER TABLE chunks ADD COLUMN IF NOT EXISTS embedding vector(1536)`,
	}
	for _, s := range stmts {
		if _, err := p.pool.Exec(ctx, s); err != nil {
			// vector extension may be unavailable in some local DBs; continue
			// so lexical upsert still works. Embedding insert will no-op gracefully.
			if strings.Contains(err.Error(), "extension") || strings.Contains(err.Error(), "vector") {
				continue
			}
			return fmt.Errorf("ensure schema (%s): %w", s, err)
		}
	}
	return nil
}

// UpsertDocument inserts or updates a document and replaces its chunks idempotently by document_id.
func (p *Postgres) UpsertDocument(ctx context.Context, doc models.Document, chunks []models.Chunk) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if doc.DocumentType == "" {
		doc.DocumentType = "developer_documentation"
	}
	if doc.Version == "" {
		doc.Version = "current"
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO documents (
			document_id, provider, service, category, document_type,
			title, source_url, version, content, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())
		ON CONFLICT (document_id) DO UPDATE SET
			provider = EXCLUDED.provider,
			service = EXCLUDED.service,
			category = EXCLUDED.category,
			document_type = EXCLUDED.document_type,
			title = EXCLUDED.title,
			source_url = EXCLUDED.source_url,
			version = EXCLUDED.version,
			content = EXCLUDED.content,
			updated_at = NOW()
	`, doc.DocumentID, doc.Provider, doc.Service, doc.Category, doc.DocumentType,
		doc.Title, doc.SourceURL, doc.Version, doc.Content)
	if err != nil {
		return fmt.Errorf("upsert document: %w", err)
	}

	_, err = tx.Exec(ctx, `DELETE FROM chunks WHERE document_id = $1`, doc.DocumentID)
	if err != nil {
		return fmt.Errorf("delete chunks: %w", err)
	}

	hasEmbeddingCol := p.columnExists(ctx, tx, "chunks", "embedding")

	for _, c := range chunks {
		if c.TokenCount == 0 {
			c.TokenCount = tokenizer.CountTokens(c.Content)
		}
		if hasEmbeddingCol && len(c.Embedding) > 0 {
			_, err = tx.Exec(ctx, `
				INSERT INTO chunks (
					chunk_id, document_id, provider, service, category,
					title, heading, section, source_url, content, token_count, position, embedding
				) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::vector)
			`, c.ChunkID, doc.DocumentID, c.Provider, c.Service, c.Category,
				c.Title, nullIfEmpty(c.Heading), nullIfEmpty(c.Section), c.SourceURL,
				c.Content, c.TokenCount, c.Position, vectorLiteral(c.Embedding))
		} else {
			_, err = tx.Exec(ctx, `
				INSERT INTO chunks (
					chunk_id, document_id, provider, service, category,
					title, heading, section, source_url, content, token_count, position
				) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			`, c.ChunkID, doc.DocumentID, c.Provider, c.Service, c.Category,
				c.Title, nullIfEmpty(c.Heading), nullIfEmpty(c.Section), c.SourceURL,
				c.Content, c.TokenCount, c.Position)
		}
		if err != nil {
			return fmt.Errorf("insert chunk %s: %w", c.ChunkID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// RebuildTermStats recomputes corpus_stats and term_stats for BM25.
func (p *Postgres) RebuildTermStats(ctx context.Context) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var count int
	var avgLen float64
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(AVG(token_count), 0)
		FROM chunks
	`).Scan(&count, &avgLen)
	if err != nil {
		return fmt.Errorf("corpus aggregates: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO corpus_stats (id, document_count, avg_doc_length, updated_at)
		VALUES (1, $1, $2, NOW())
		ON CONFLICT (id) DO UPDATE SET
			document_count = EXCLUDED.document_count,
			avg_doc_length = EXCLUDED.avg_doc_length,
			updated_at = NOW()
	`, count, avgLen)
	if err != nil {
		return fmt.Errorf("update corpus_stats: %w", err)
	}

	_, err = tx.Exec(ctx, `DELETE FROM term_stats`)
	if err != nil {
		return fmt.Errorf("clear term_stats: %w", err)
	}

	rows, err := tx.Query(ctx, `SELECT content FROM chunks`)
	if err != nil {
		return fmt.Errorf("list chunk content: %w", err)
	}
	defer rows.Close()

	df := map[string]int{}
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return err
		}
		seen := map[string]struct{}{}
		for term := range tokenizer.TokenizeWithFreq(content) {
			if _, ok := seen[term]; ok {
				continue
			}
			seen[term] = struct{}{}
			df[term]++
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	batch := &pgx.Batch{}
	for term, freq := range df {
		batch.Queue(`INSERT INTO term_stats (term, document_freq) VALUES ($1, $2)`, term, freq)
	}
	if batch.Len() > 0 {
		br := tx.SendBatch(ctx, batch)
		if err := br.Close(); err != nil {
			return fmt.Errorf("insert term_stats: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// Ping checks database connectivity.
func (p *Postgres) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Close closes the underlying pool.
func (p *Postgres) Close() {
	p.pool.Close()
}

func (p *Postgres) columnExists(ctx context.Context, tx pgx.Tx, table, column string) bool {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = $1 AND column_name = $2
		)
	`, table, column).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

func vectorLiteral(v []float32) string {
	if len(v) == 0 {
		return "[]"
	}
	var b strings.Builder
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%g", f)
	}
	b.WriteByte(']')
	return b.String()
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
