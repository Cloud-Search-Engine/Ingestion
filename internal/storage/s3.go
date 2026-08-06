package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Store puts and gets objects, including against LocalStack.
type S3Store struct {
	client *s3.Client
	bucket string
}

// S3Options configures the S3 client.
type S3Options struct {
	Region      string
	EndpointURL string // AWS_ENDPOINT_URL_S3 or AWS_ENDPOINT_URL
	Bucket      string
}

// NewS3 creates an S3 client compatible with LocalStack custom endpoints.
func NewS3(ctx context.Context, opts S3Options) (*S3Store, error) {
	if opts.Bucket == "" {
		return nil, fmt.Errorf("S3 bucket is required")
	}
	if opts.Region == "" {
		opts.Region = "us-east-1"
	}

	endpoint := firstNonEmpty(
		opts.EndpointURL,
		os.Getenv("AWS_ENDPOINT_URL_S3"),
		os.Getenv("AWS_ENDPOINT_URL"),
	)

	loadOpts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(opts.Region),
	}
	if endpoint != "" {
		loadOpts = append(loadOpts,
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		)
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true // required for LocalStack
		}
	})

	return &S3Store{client: client, bucket: opts.Bucket}, nil
}

// PutObject uploads raw bytes to the configured bucket.
func (s *S3Store) PutObject(ctx context.Context, key, contentType string, body []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("s3 put %s/%s: %w", s.bucket, key, err)
	}
	return nil
}

// GetObject downloads an object body.
func (s *S3Store) GetObject(ctx context.Context, bucket, key string) ([]byte, string, error) {
	if bucket == "" {
		bucket = s.bucket
	}
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", fmt.Errorf("s3 get %s/%s: %w", bucket, key, err)
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, "", fmt.Errorf("s3 read body: %w", err)
	}
	ct := ""
	if out.ContentType != nil {
		ct = *out.ContentType
	}
	return data, ct, nil
}

// Bucket returns the default bucket name.
func (s *S3Store) Bucket() string { return s.bucket }

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
