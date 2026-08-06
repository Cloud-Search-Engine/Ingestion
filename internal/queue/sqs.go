package queue

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// Client wraps SQS operations with LocalStack-friendly endpoint support.
type Client struct {
	sqs      *sqs.Client
	queueURL string
}

// Options configures the SQS client.
type Options struct {
	Region            string
	EndpointURL       string // AWS_ENDPOINT_URL (LocalStack)
	QueueURL          string
	VisibilityTimeout time.Duration
}

// New creates an SQS client. When EndpointURL is set (LocalStack), static
// test credentials are used and path-style addressing is not required for SQS.
func New(ctx context.Context, opts Options) (*Client, error) {
	if opts.QueueURL == "" {
		return nil, fmt.Errorf("SQS queue URL is required")
	}
	if opts.Region == "" {
		opts.Region = "us-east-1"
	}

	loadOpts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(opts.Region),
	}

	endpoint := firstNonEmpty(opts.EndpointURL, os.Getenv("AWS_ENDPOINT_URL"))
	if endpoint != "" {
		loadOpts = append(loadOpts,
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		)
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	var sqsOpts []func(*sqs.Options)
	if endpoint != "" {
		sqsOpts = append(sqsOpts, func(o *sqs.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}

	return &Client{
		sqs:      sqs.NewFromConfig(cfg, sqsOpts...),
		queueURL: opts.QueueURL,
	}, nil
}

// SendMessage enqueues a raw JSON body.
func (c *Client) SendMessage(ctx context.Context, body string) (string, error) {
	out, err := c.sqs.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(c.queueURL),
		MessageBody: aws.String(body),
	})
	if err != nil {
		return "", fmt.Errorf("sqs send: %w", err)
	}
	if out.MessageId == nil {
		return "", nil
	}
	return *out.MessageId, nil
}

// ReceivedMessage is a polled SQS message.
type ReceivedMessage struct {
	ID            string
	ReceiptHandle string
	Body          string
}

// ReceiveMessages long-polls up to maxMessages with the given visibility timeout.
func (c *Client) ReceiveMessages(ctx context.Context, maxMessages int32, visibilityTimeout time.Duration, waitSeconds int32) ([]ReceivedMessage, error) {
	if maxMessages < 1 {
		maxMessages = 1
	}
	if maxMessages > 10 {
		maxMessages = 10
	}
	if waitSeconds < 0 {
		waitSeconds = 0
	}
	if waitSeconds > 20 {
		waitSeconds = 20
	}
	vt := int32(visibilityTimeout.Seconds())
	if vt < 0 {
		vt = 30
	}

	out, err := c.sqs.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(c.queueURL),
		MaxNumberOfMessages: maxMessages,
		WaitTimeSeconds:     waitSeconds,
		VisibilityTimeout:   vt,
		MessageAttributeNames: []string{
			string(types.QueueAttributeNameAll),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("sqs receive: %w", err)
	}

	msgs := make([]ReceivedMessage, 0, len(out.Messages))
	for _, m := range out.Messages {
		msg := ReceivedMessage{}
		if m.MessageId != nil {
			msg.ID = *m.MessageId
		}
		if m.ReceiptHandle != nil {
			msg.ReceiptHandle = *m.ReceiptHandle
		}
		if m.Body != nil {
			msg.Body = *m.Body
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

// DeleteMessage acknowledges successful processing.
func (c *Client) DeleteMessage(ctx context.Context, receiptHandle string) error {
	if receiptHandle == "" {
		return fmt.Errorf("empty receipt handle")
	}
	_, err := c.sqs.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(c.queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	if err != nil {
		return fmt.Errorf("sqs delete: %w", err)
	}
	return nil
}

// ChangeVisibility extends or resets the visibility timeout for in-flight handling.
func (c *Client) ChangeVisibility(ctx context.Context, receiptHandle string, timeout time.Duration) error {
	_, err := c.sqs.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(c.queueURL),
		ReceiptHandle:     aws.String(receiptHandle),
		VisibilityTimeout: int32(timeout.Seconds()),
	})
	if err != nil {
		return fmt.Errorf("sqs change visibility: %w", err)
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
