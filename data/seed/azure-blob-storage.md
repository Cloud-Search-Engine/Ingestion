# Azure Blob Storage

Azure Blob Storage is Microsoft's object storage solution for the cloud. It
is optimized for storing massive amounts of unstructured data such as text,
binary data, images, documents, and media files. Blob Storage is used for
data lakes, backups, archival, and serving content to applications.

## Blob types

Block blobs are ideal for text and binary files and support efficient
parallel uploads. Append blobs are optimized for append operations such as
logging. Page blobs store random-access files up to 8 TiB and underpin Azure
Virtual Machine disks (VHDs).

## Access tiers

The Hot tier is optimized for frequent access. The Cool and Cold tiers
reduce storage cost for infrequently accessed data with higher access
charges. The Archive tier offers the lowest storage cost with
hours-scale retrieval. Lifecycle management policies move blobs between
tiers based on age or last-access time.

## Containers and authorization

Blobs live inside containers within a storage account. Authorize access with
Microsoft Entra ID (recommended), shared key, or shared access signatures
(SAS). SAS tokens grant fine-grained, time-bounded permissions for upload or
download without sharing the account key.

## Redundancy

Locally redundant storage (LRS), zone-redundant storage (ZRS),
geo-redundant storage (GRS), and geo-zone-redundant storage (GZRS) provide
increasing durability and availability. Choose redundancy based on recovery
objectives and regional compliance requirements.
