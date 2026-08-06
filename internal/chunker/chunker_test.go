package chunker_test

import (
	"testing"

	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/chunker"
	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/models"
)

func TestChunkPreservesHeading(t *testing.T) {
	parsed := models.ParsedDocument{
		Title:   "SQS Guide",
		Content: "Visibility timeout\nThe visibility timeout hides a message.\n\nDead-letter queues\nUse a DLQ for poison messages.",
		Headings: []models.Heading{
			{Level: 2, Text: "Visibility timeout", CharStart: 0},
			{Level: 2, Text: "Dead-letter queues", CharStart: 50},
		},
	}
	msg := models.IngestMessage{
		DocumentID: "aws-sqs",
		Provider:   "aws",
		Service:    "sqs",
		Category:   "messaging",
		Title:      "SQS Guide",
		SourceURL:  "https://example.com",
	}

	chunks := chunker.Chunk("aws-sqs", msg, parsed, chunker.Options{SizeTokens: 256})
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	if chunks[0].DocumentID != "aws-sqs" {
		t.Fatalf("document_id=%s", chunks[0].DocumentID)
	}
	if chunks[0].Position != 0 {
		t.Fatalf("position=%d", chunks[0].Position)
	}
}
