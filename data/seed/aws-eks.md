# Amazon Elastic Kubernetes Service (EKS)

Amazon EKS is a managed Kubernetes service that runs the Kubernetes control
plane across multiple Availability Zones for high availability. EKS manages
the availability and scalability of the Kubernetes API servers and the etcd
persistence layer, while you run worker nodes on EC2, AWS Fargate, or a mix
of both.

## Cluster networking

EKS integrates with Amazon VPC so that pods can receive IP addresses from
your VPC subnets via the Amazon VPC CNI plugin. Security groups and network
policies control east-west and north-south traffic. Choose private subnets
for worker nodes in production and expose services through load balancers.

## Managed node groups

Managed node groups automate the provisioning and lifecycle management of
EC2 instances for your cluster. You can configure instance types, AMI
updates, scaling, and labels. When you update a node group, EKS drains nodes
gracefully before replacing them, reducing application disruption.

## Fargate profiles

AWS Fargate lets you run pods without managing EC2 instances. A Fargate
profile selects which pods run on Fargate based on namespace and labels.
Fargate is useful for spiky workloads and for teams that want to minimize
node operations, though it has constraints around DaemonSets and privileged
pods.

## IAM and IRSA

IAM Roles for Service Accounts (IRSA) associates a Kubernetes service account
with an IAM role using OIDC federation. Pods that use the service account
receive temporary AWS credentials scoped to that role, avoiding long-lived
secrets on nodes and enabling least-privilege access to S3, SQS, and other
AWS APIs.
