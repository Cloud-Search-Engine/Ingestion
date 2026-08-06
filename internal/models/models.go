package models

import "time"

// Document is a full ingested documentation page.
type Document struct {
	ID           string    `json:"id,omitempty"`
	DocumentID   string    `json:"document_id"`
	Provider     string    `json:"provider"`
	Service      string    `json:"service"`
	Category     string    `json:"category"`
	DocumentType string    `json:"document_type"`
	Title        string    `json:"title"`
	SourceURL    string    `json:"source_url"`
	Version      string    `json:"version"`
	Content      string    `json:"content,omitempty"`
	S3Key        string    `json:"s3_key,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
}

// Chunk is a searchable unit of a document.
type Chunk struct {
	ID         string    `json:"id,omitempty"`
	ChunkID    string    `json:"chunk_id"`
	DocumentID string    `json:"document_id"`
	Provider   string    `json:"provider"`
	Service    string    `json:"service"`
	Category   string    `json:"category"`
	Title      string    `json:"title"`
	Heading    string    `json:"heading,omitempty"`
	Section    string    `json:"section,omitempty"`
	SourceURL  string    `json:"source_url"`
	Content    string    `json:"content"`
	TokenCount int       `json:"token_count"`
	Position   int       `json:"position"`
	Embedding  []float32 `json:"embedding,omitempty"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
}

// IngestMessage is the SQS payload that triggers worker processing.
type IngestMessage struct {
	DocumentID   string `json:"document_id"`
	S3Bucket     string `json:"s3_bucket"`
	S3Key        string `json:"s3_key"`
	Provider     string `json:"provider"`
	Service      string `json:"service"`
	Category     string `json:"category"`
	DocumentType string `json:"document_type"`
	Title        string `json:"title"`
	SourceURL    string `json:"source_url"`
	Version      string `json:"version"`
	ContentType  string `json:"content_type"` // text/html, text/markdown, text/plain
}

// SeedMeta is optional front-matter style metadata for local seed files.
type SeedMeta struct {
	DocumentID   string `json:"document_id"`
	Provider     string `json:"provider"`
	Service      string `json:"service"`
	Category     string `json:"category"`
	DocumentType string `json:"document_type"`
	Title        string `json:"title"`
	SourceURL    string `json:"source_url"`
	Version      string `json:"version"`
}

// ParsedDocument is the output of the parser stage.
type ParsedDocument struct {
	Title    string
	Content  string
	Headings []Heading
}

// Heading captures a section heading and its approximate character offset.
type Heading struct {
	Level    int
	Text     string
	CharStart int
}
