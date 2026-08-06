# Amazon Simple Queue Service (SQS)

Amazon SQS is a fully managed message queuing service that enables you to
decouple and scale microservices, distributed systems, and serverless
applications. SQS eliminates the complexity and overhead associated with
managing and operating message-oriented middleware.

## Queue types

Standard queues offer maximum throughput, best-effort ordering, and
at-least-once delivery. Messages may occasionally be delivered more than
once, and order is not strictly guaranteed.

FIFO queues are designed to guarantee that messages are processed exactly
once, in the exact order that they are sent. FIFO queues support up to
300 messages per second with batching, or 3,000 with high-throughput mode.

## Visibility timeout

When a consumer receives a message, that message remains in the queue but
becomes invisible to other consumers for a period known as the visibility
timeout. The default visibility timeout is 30 seconds. If the consumer
fails to delete the message before the timeout expires, the message becomes
visible again and may be processed by another consumer.

Choose a visibility timeout that is longer than the typical processing time
for your workload. You can also use ChangeMessageVisibility to extend the
timeout for long-running jobs.

## Dead-letter queues

A dead-letter queue (DLQ) receives messages that fail processing repeatedly.
Configure `maxReceiveCount` on a redrive policy so that after N failed
receive attempts, SQS moves the message to the DLQ for inspection and
replay. DLQs help isolate poison messages without blocking the main queue.

## Long polling

Long polling reduces empty responses and false empty responses by allowing
ReceiveMessage to wait up to 20 seconds for a message to arrive. Prefer long
polling (`WaitTimeSeconds` > 0) over short polling for production workers.
