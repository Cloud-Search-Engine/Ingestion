# Azure Service Bus

Azure Service Bus is a fully managed enterprise message broker with message
queues and publish-subscribe topics. Service Bus is used to decouple
applications and services and provides reliable asynchronous message
transfer with advanced features such as sessions, transactions, and
duplicate detection.

## Queues and topics

A queue stores messages until a consuming application is available. Messages
are delivered in a competing-consumer pattern: each message is processed by
a single consumer.

Topics and subscriptions enable a publish-subscribe pattern. Each published
message is made available to every subscription registered to the topic.
Subscriptions can apply SQL-like filter rules so that only matching messages
are delivered.

## Peek-lock vs receive-and-delete

In peek-lock mode, Service Bus locks a message for the consumer. The lock
duration defaults to 30 seconds and can be renewed. After successful
processing the consumer completes (settles) the message. If the lock expires
or the consumer abandons the message, it becomes available again.

Receive-and-delete removes the message as soon as it is read. This mode is
faster but risks message loss if the consumer crashes after receive and
before finishing work.

## Sessions and ordering

Message sessions enable ordered processing of related messages that share a
SessionId. A session-aware receiver holds an exclusive lock on a session,
guaranteeing FIFO processing within that session across multiple messages.

## Dead-lettering

Messages that exceed the MaxDeliveryCount or fail validation can be moved to
the entity's dead-letter sub-queue. Dead-lettered messages retain properties
describing why they were dead-lettered, which helps operators diagnose and
resubmit them after fixing the underlying issue.
