## Intro

All the inputs in this folder collect "Assets". Assets are defined as elements within your infrastructure, such as containers, machines, pods, clusters, etc.

## Supported Asset Inputs

assetbeat supports the following asset input types at the moment:

- [assets_aws](aws/README.md)
- [assets_gcp](gcp/README.md)
- [assets_k8s](k8s/README.md)


## Index pattern

Each Asset input publishes documents to an index with pattern `assets-{asset.type}-{namespace}`. The default `namespace` value is `default`. For example, asset documents for an AWS EC2 instance would be published to `assets-aws.ec2.instance-default`

##  Common configuration options

The following configuration options are supported by all Asset inputs.

* `period`: How often data should be collected.
* `index_namespace`: This option can be set to replace the default value for `namespace` with a custom string.
* `asset_types`: The list of specific asset types to collect data about.

### Type specific options

- [assets_aws](aws/README.md#Configuration)
- [assets_gcp](gcp/README.md#Configuration)
- [assets_k8s](k8s/README.md#Configuration)

## Asset Inputs Relationships

Certain assets types collected by the different inputs can be connected with each other
with parent/children hierarchy.

## Asset identifier

Each asset is identified by its Elastic Asset Name (EAN), which is an URN-style identifier with the following pattern,

`{asset.kind}:{asset.id}` (e.g. `host:i-123456`).

assetbeat publishes this field under `asset.ean`.

### GKE clusters and nodes
In case `assets_k8s` input is collecting Kubernetes nodes assets and those nodes belong to a GKE cluster, the following field mapping can be used to link the Kubernetes nodes with their cluster.

| assets_k8s (k8s.node) | assets_gcp (k8s.cluster) | Notes/Description                                                                                                                                                                                                                    |
|-----------------------|--------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| cloud.instance.id     | asset.children           | For each GKE cluster, the field `asset.children` contains the EANs of the GCP instances linked. You can extract an instance ID from each EAN and map it to the field `cloud.instance.id`, which assetbeat publishes for GKE nodes. |
| asset.parents         | asset.ean                | The `asset.parents` of k8s.node asset type contains the EAN of the kubernetes cluster it belongs to.                                                                                                                                 |

### EKS clusters and nodes

In case `assets_k8s` input is collecting Kubernetes nodes assets and those nodes belong to an EKS cluster, the following field mapping can be used to link the Kubernetes nodes with their cluster.

| assets_k8s (k8s.node) | assets_aws (k8s.cluster) | Notes/Description                                                                                                                                                                                                                    |
|-----------------------|--------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| cloud.instance.id     | asset.children           | For each EKS cluster, the field `asset.children` contains the EANs of the EC2 instances linked. You can extract an instance ID from each EAN and map it to the field `cloud.instance.id`, which assetbeat publishes for EKS nodes. |

**_Note_:** The above mapping is not currently available for EKS Fargate clusters.
