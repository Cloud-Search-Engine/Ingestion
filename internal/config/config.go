package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds runtime configuration for crawler and worker.
type Config struct {
	DatabaseURL       string
	AWSEndpointURL    string
	AWSEndpointURLS3  string
	AWSRegion         string
	S3Bucket          string
	SQSQueueURL       string
	RedisURL          string
	EmbeddingProvider string
	WorkerConcurrency int
	ChunkSizeTokens   int
	VisibilityTimeout time.Duration
	ShutdownTimeout   time.Duration
	SeedDir           string
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://cloudsearch:cloudsearch@localhost:5432/cloudsearch?sslmode=disable"),
		AWSEndpointURL:    getEnv("AWS_ENDPOINT_URL", ""),
		AWSEndpointURLS3:  firstNonEmpty(os.Getenv("AWS_ENDPOINT_URL_S3"), os.Getenv("AWS_ENDPOINT_URL")),
		AWSRegion:         getEnv("AWS_REGION", "us-east-1"),
		S3Bucket:          getEnv("S3_BUCKET", "cloud-search-raw"),
		SQSQueueURL:       getEnv("SQS_QUEUE_URL", ""),
		RedisURL:          getEnv("REDIS_URL", "redis://localhost:6379/0"),
		EmbeddingProvider: getEnv("EMBEDDING_PROVIDER", "noop"),
		WorkerConcurrency: getInt("WORKER_CONCURRENCY", 4),
		ChunkSizeTokens:   getInt("CHUNK_SIZE_TOKENS", 512),
		VisibilityTimeout: getDuration("SQS_VISIBILITY_TIMEOUT", 60*time.Second),
		ShutdownTimeout:   getDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
		SeedDir:           getEnv("SEED_DIR", "data/seed"),
	}

	switch cfg.ChunkSizeTokens {
	case 256, 512, 1024:
	default:
		cfg.ChunkSizeTokens = 512
	}

	switch cfg.EmbeddingProvider {
	case "noop", "openai":
	default:
		return Config{}, fmt.Errorf("unsupported EMBEDDING_PROVIDER %q (use noop|openai)", cfg.EmbeddingProvider)
	}

	if cfg.WorkerConcurrency < 1 {
		cfg.WorkerConcurrency = 1
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func getInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
