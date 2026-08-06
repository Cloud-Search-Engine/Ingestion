# Google Kubernetes Engine (GKE)

Google Kubernetes Engine (GKE) is a managed Kubernetes service on Google
Cloud. GKE Autopilot manages nodes for you, while GKE Standard gives you
control over node pools, machine types, and upgrade strategies. Both modes
run a Google-managed control plane with multi-zonal high availability
options.

## Autopilot vs Standard

Autopilot charges for pod resource requests and handles node provisioning,
scaling, and security hardening automatically. Standard mode is appropriate
when you need DaemonSets with host access, custom machine families, or
specialized GPU configurations.

## Workload Identity

Workload Identity binds a Kubernetes service account to a Google service
account so pods can call Google Cloud APIs with short-lived credentials.
This is the recommended way to grant access to Cloud Storage, Pub/Sub, and
Secret Manager without downloading service account keys.

## Networking and Gateway API

GKE supports VPC-native clusters where pods use alias IP ranges from your
VPC. The Gateway API and built-in Ingress resources integrate with Google
Cloud Load Balancing for HTTP(S) and multi-cluster ingress scenarios.

## Release channels

GKE release channels (Rapid, Regular, Stable) control how quickly clusters
receive Kubernetes minor versions and security patches. Prefer Regular or
Stable for production, and pin critical workloads with maintenance
exclusions during peak business periods.
