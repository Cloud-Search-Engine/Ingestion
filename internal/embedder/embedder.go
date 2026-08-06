package embedder

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
)

const DefaultDim = 1536

// Embedder produces dense vectors for chunk text.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Dimensions() int
}

// New returns an Embedder for the given provider name.
func New(provider string) (Embedder, error) {
	switch provider {
	case "", "noop":
		return NewNoop(DefaultDim), nil
	case "openai":
		return NewOpenAIStub(DefaultDim), nil
	default:
		return nil, fmt.Errorf("unknown embedding provider %q", provider)
	}
}

// NoopEmbedder produces deterministic unit-ish vectors from content hashes.
// Useful for local demos and ensuring vector search plumbing works without an API key.
type NoopEmbedder struct {
	dim int
}

// NewNoop creates a deterministic hash-based embedder.
func NewNoop(dim int) *NoopEmbedder {
	if dim <= 0 {
		dim = DefaultDim
	}
	return &NoopEmbedder{dim: dim}
}

func (n *NoopEmbedder) Dimensions() int { return n.dim }

func (n *NoopEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	_ = ctx
	out := make([][]float32, len(texts))
	for i, t := range texts {
		out[i] = hashEmbedding(t, n.dim)
	}
	return out, nil
}

// OpenAIStub is a placeholder for a future OpenAI embeddings integration.
// It currently falls back to the same deterministic vectors so the pipeline
// remains usable without credentials, and returns a clear error if forced.
type OpenAIStub struct {
	dim    int
	apiKey string
	noop   *NoopEmbedder
}

// NewOpenAIStub creates an OpenAI stub that uses Noop until a real client is wired.
func NewOpenAIStub(dim int) *OpenAIStub {
	return &OpenAIStub{dim: dim, noop: NewNoop(dim)}
}

func (o *OpenAIStub) Dimensions() int { return o.dim }

func (o *OpenAIStub) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	// Real OpenAI calls will go here. For now, deterministic fallback keeps
	// local demos and tests working without network or secrets.
	if o.apiKey == "" {
		return o.noop.Embed(ctx, texts)
	}
	return nil, fmt.Errorf("openai embedder not implemented yet; set EMBEDDING_PROVIDER=noop")
}

// hashEmbedding expands a SHA-256 digest into a deterministic float32 vector
// and L2-normalizes it so cosine similarity is meaningful.
func hashEmbedding(text string, dim int) []float32 {
	sum := sha256.Sum256([]byte(text))
	vec := make([]float32, dim)

	// Expand digest with successive hashes so we fill all dimensions.
	seed := sum
	var offset int
	for offset < dim {
		for i := 0; i+4 <= len(seed) && offset < dim; i += 4 {
			u := binary.BigEndian.Uint32(seed[i : i+4])
			// Map to (-1, 1)
			vec[offset] = (float32(u)/float32(^uint32(0)))*2 - 1
			offset++
		}
		seed = sha256.Sum256(append(seed[:], byte(offset)))
	}

	var norm float64
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return vec
	}
	for i := range vec {
		vec[i] = float32(float64(vec[i]) / norm)
	}
	return vec
}
