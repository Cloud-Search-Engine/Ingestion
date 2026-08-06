package embedder_test

import (
	"context"
	"testing"

	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/embedder"
)

func TestNoopDeterministic(t *testing.T) {
	e := embedder.NewNoop(embedder.DefaultDim)
	ctx := context.Background()

	a, err := e.Embed(ctx, []string{"hello world"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := e.Embed(ctx, []string{"hello world"})
	if err != nil {
		t.Fatal(err)
	}
	if len(a[0]) != embedder.DefaultDim {
		t.Fatalf("dim=%d want %d", len(a[0]), embedder.DefaultDim)
	}
	for i := range a[0] {
		if a[0][i] != b[0][i] {
			t.Fatalf("vectors differ at %d", i)
		}
	}

	c, err := e.Embed(ctx, []string{"different text"})
	if err != nil {
		t.Fatal(err)
	}
	same := true
	for i := range a[0] {
		if a[0][i] != c[0][i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("expected different content to produce different vectors")
	}
}
