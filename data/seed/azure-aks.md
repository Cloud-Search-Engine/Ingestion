# Azure Kubernetes Service (AKS)

Azure Kubernetes Service (AKS) is a managed Kubernetes offering that
offloads control-plane operations to Azure while you manage agent pools and
workloads. AKS integrates with Azure Active Directory, Azure Monitor,
Azure Policy, and Azure Container Registry for a full application platform.

## Node pools

AKS clusters contain one or more node pools. The system node pool runs
critical cluster pods. User node pools run application workloads and can use
different VM sizes, OS types (Linux or Windows), and spot instances. You can
scale node pools manually or with the cluster autoscaler.

## Networking models

AKS supports kubenet and Azure CNI networking. With Azure CNI, every pod
receives an IP address from the subnet, which simplifies connectivity to
other Azure resources but requires careful IP capacity planning. Overlay
networking modes reduce IP consumption for large clusters.

## Ingress and load balancing

You can expose applications with Azure Load Balancer services, Application
Gateway Ingress Controller (AGIC), or NGINX ingress. Managed identities and
Key Vault CSI integration help inject certificates and secrets without
baking them into container images.

## Upgrades and maintenance

AKS provides planned maintenance windows and surge upgrades that temporarily
add nodes so workloads can be drained and rescheduled during Kubernetes
version upgrades. Always validate add-on compatibility and review the AKS
release calendar before upgrading production clusters.
