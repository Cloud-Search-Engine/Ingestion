package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/config"
	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/embedder"
	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/pipeline"
	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/queue"
	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config", "error", err)
		os.Exit(1)
	}
	if cfg.SQSQueueURL == "" {
		logger.Error("SQS_QUEUE_URL is required")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := storage.ConnectPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database", "error", err)
		os.Exit(1)
	}
	db := storage.NewPostgres(pool)
	defer db.Close()

	if err := db.EnsureSchema(ctx); err != nil {
		logger.Warn("ensure schema", "error", err)
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
		Region:            cfg.AWSRegion,
		EndpointURL:       cfg.AWSEndpointURL,
		QueueURL:          cfg.SQSQueueURL,
		VisibilityTimeout: cfg.VisibilityTimeout,
	})
	if err != nil {
		logger.Error("sqs", "error", err)
		os.Exit(1)
	}

	emb, err := embedder.New(cfg.EmbeddingProvider)
	if err != nil {
		logger.Error("embedder", "error", err)
		os.Exit(1)
	}

	proc := &pipeline.Processor{
		S3:        s3Store,
		DB:        db,
		Embedder:  emb,
		ChunkSize: cfg.ChunkSizeTokens,
		Log:       logger,
	}

	jobs := make(chan queue.ReceivedMessage, cfg.WorkerConcurrency*2)
	var wg sync.WaitGroup

	for i := 0; i < cfg.WorkerConcurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for msg := range jobs {
				if err := handleMessage(ctx, proc, sqsClient, msg, logger, workerID); err != nil {
					logger.Error("process failed", "worker", workerID, "message_id", msg.ID, "error", err)
					// Leave message for retry via visibility timeout.
				}
			}
		}(i)
	}

	logger.Info("worker started",
		"concurrency", cfg.WorkerConcurrency,
		"queue", cfg.SQSQueueURL,
		"bucket", cfg.S3Bucket,
		"embedding", cfg.EmbeddingProvider,
		"chunk_size", cfg.ChunkSizeTokens,
	)

	pollLoop(ctx, sqsClient, jobs, cfg.VisibilityTimeout, logger)

	close(jobs)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("workers drained")
	case <-time.After(cfg.ShutdownTimeout):
		logger.Warn("shutdown timed out waiting for workers")
	}
}

func pollLoop(ctx context.Context, sqsClient *queue.Client, jobs chan<- queue.ReceivedMessage, visibility time.Duration, logger *slog.Logger) {
	for {
		if ctx.Err() != nil {
			logger.Info("shutdown signal received, stopping poll")
			return
		}

		msgs, err := sqsClient.ReceiveMessages(ctx, 5, visibility, 10)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Error("receive", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
			continue
		}
		for _, m := range msgs {
			select {
			case <-ctx.Done():
				return
			case jobs <- m:
			}
		}
	}
}

func handleMessage(ctx context.Context, proc *pipeline.Processor, sqsClient *queue.Client, msg queue.ReceivedMessage, logger *slog.Logger, workerID int) error {
	ingestMsg, err := pipeline.ParseMessage(msg.Body)
	if err != nil {
		// Poison message: delete to avoid infinite retries of bad JSON.
		logger.Error("invalid message body", "worker", workerID, "error", err)
		_ = sqsClient.DeleteMessage(ctx, msg.ReceiptHandle)
		return err
	}

	if err := proc.Process(ctx, ingestMsg); err != nil {
		return err
	}
	if err := sqsClient.DeleteMessage(ctx, msg.ReceiptHandle); err != nil {
		return err
	}
	logger.Info("message processed", "worker", workerID, "document_id", ingestMsg.DocumentID, "message_id", msg.ID)
	return nil
}
