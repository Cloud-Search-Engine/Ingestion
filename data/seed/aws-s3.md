# Amazon S3

Amazon Simple Storage Service (S3) is object storage built to store and
retrieve any amount of data from anywhere. S3 provides industry-leading
durability (designed for 99.999999999% durability) and offers multiple
storage classes for cost optimization across access patterns.

## Buckets and objects

Data is stored as objects within buckets. An object consists of data,
metadata, and a key (the unique identifier within the bucket). Bucket names
must be globally unique. You can organize keys with prefixes that resemble
folders, though S3 is a flat namespace.

## Storage classes

S3 Standard is for frequently accessed data. S3 Intelligent-Tiering
automatically moves objects between access tiers. S3 Glacier Instant
Retrieval, Flexible Retrieval, and Deep Archive address archival workloads
with increasing retrieval latency and decreasing cost. Lifecycle rules can
transition or expire objects automatically.

## Consistency and versioning

S3 provides strong read-after-write consistency for PUT and DELETE of
objects in all AWS Regions. Enable versioning to preserve, retrieve, and
restore every version of an object, which protects against accidental
deletes and overwrites.

## Access control

Control access with IAM policies, bucket policies, Access Control Lists
(ACLs), and S3 Access Points. Block Public Access settings help prevent
accidental exposure. Prefer bucket policies and IAM over ACLs for new
applications, and use presigned URLs for time-limited client uploads.
