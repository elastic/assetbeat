## Supported Asset Inputs

Inputrunner supports the following asset input types at the moment:

- [assets_aws](aws/README.md)
- [assets_gcp](gcp/README.md)
- [assets_k8s](k8s/README.md)


##  Configuration

All the asset input types supports type specific configuration options plus the Common options described below.

### Common options

The following configuration options are supported by all Asset inputs.

* `period`: How often data should be collected.
* `index_namespace`: Each document is published to an index with pattern `assets-{asset.type}-{namespace}`. This option can be set to replace the default value for `namespace`, `default`, with a custom string.
* `asset_types`: The list of specific asset types to collect data about.

### Type specific options

- [assets_aws](aws/README.md#Configuration)
- [assets_gcp](gcp/README.md#Configuration)
- [assets_aws](k8s/README.md#Configuration)


## Asset Inputs Relationships

Certain assets types collected by the different inputs can be connected with each other
with parent/children hierarchy.

### assets_k8s input in a GKE or EKS cluster
In case `assets_k8s` input is collecting kubernetes nodes assets and those nodes belong to either
a GKE or EKS cluster the `cloud.instance.id` is collected from the node's metadata.
That field is published and corresponds to the `asset.id` of [Compute Engine instances](gcp/README.md#Compute Engine instances) for GCP or
[EC2 instances](aws/README.md#EC2 instances) for AWS.
At the same time if `assets_aws` or `assets_gcp` input are configured with `k8s.cluster` asset_type enabled,
they associate the Compute Engine or EC2 instances respectively with the EKS/AKS cluster they belong to.
The instance ids are then published as part of the `asset.children` field of `k8s.cluster` asset types.


### assets_k8s input in a GKE cluster

In case `assets_k8s` input is collecting kubernetes nodes assets and those nodes belong to a
GKE cluster, then the cluster id can be retrieved from the CSP metadata endpoint.
The assets published from [k8s.node](k8s/README.md#K8s Nodes) asset_type will list in the 
`asset.parents` field the EAN of the [Google Kubernetes Engine cluster](gcp/README.md#Google Kubernetes Engine clusters).


