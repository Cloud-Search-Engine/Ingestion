# Google Cloud Pub/Sub

Pub/Sub is an asynchronous messaging service that decouples services that
produce events from services that process those events. Pub/Sub offers
durable message storage, real-time delivery, and at-least-once delivery
semantics with optional exactly-once delivery for compatible subscribers.

## Topics and subscriptions

Publishers send messages to a topic. One or more subscriptions can be
attached to a topic. Each subscription receives a copy of every message
published to the topic (fan-out), unless filters are configured.

Push subscriptions deliver messages to an HTTPS endpoint. Pull subscriptions
require the subscriber application to call the pull API (or use streaming
pull) to fetch messages. Streaming pull is recommended for high-throughput
workers that need low latency.

## Acknowledgement deadline

When a message is delivered to a subscriber, Pub/Sub starts an
acknowledgement deadline. If the subscriber does not acknowledge (ack) the
message before the deadline, Pub/Sub redelivers it. Subscribers can modify
the ack deadline for individual messages when processing takes longer than
expected.

## Ordering keys

Messages published with the same ordering key to an ordered subscription are
delivered in publish order. Ordering keys allow you to preserve sequence for
related events (for example, updates for a single user ID) without forcing
global ordering across the entire topic.

## Dead-letter topics

A dead-letter topic captures messages that cannot be successfully processed
after a configured number of delivery attempts. Attach a subscription to the
dead-letter topic for inspection, alerting, and replay workflows.
