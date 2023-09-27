# AWS Assets Input

## What does it do?

The AWS Assets Input collects data about AWS resources and their relationships to each other.
Information about the following resources is currently collected:

- Amazon Elastic Compute Cloud (EC2) instances
- Amazon Elastic Kubernetes Service (EKS) clusters
- Amazon Virtual Private Clouds (VPCs)
- VPC Subnets

These resources are related by a hierarchy of parent/child relationships:

```mermaid
flowchart TD
A[VPC] -->|is parent of| B[VPC Subnet];
A[VPC] -->|is parent of| C[VPC Subnet];
B[VPC Subnet 1] -->|is parent of| D[EC2 instance 1];
C[VPC Subnet 2] -->|is parent of| E[EC2 instance 2];

A1[VPC] -->|is parent of| B1[EKS Cluster];
B1[EKS Cluster] -->|is parent of| C1[EC2 instance 1];
B1[EKS Cluster] -->|is parent of| D1[EC2 instance 2];
```

## Configuration

```yaml
assetbeat.inputs:
  - type: assets_aws
    regions:
        - <region>
    access_key_id: <your AWS Access Key>
    secret_access_key: <your AWS Secret Key>
    session_token: <your AWS Session Token>
```

The AWS Assets Input supports the following configuration options plus the [Common options](../README.md#Common options).

* `regions`: The list of AWS regions to collect data from.
* `access_key_id`: your AWS access key ID (e.g. `AKIAIOSFODNN7EXAMPLE`).
* `secret_access_key`: your AWS secret access key (e.g. `wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`).
* `session_token`: your AWS session token. It is required when using temporary security credentials.

**_Note_:** `access_key_id`, `secret_access_key` and `session_token` can be omitted if:
* The environment variables `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` and/or `AWS_SESSION_TOKEN` are set.
* assetbeat is running on an EC2 instance with an attached Instance Role.


## AWS Permissions

The following AWS IAM permissions are required for the AWS Assets Input to function.

* `ec2:DescribeInstances`
* `ec2:DescribeVpcs`
* `ec2:DescribeSubnets`
* `autoscaling:DescribeAutoscalingGroups`
* `eks:ListNodeGroups`
* `eks:DescribeNodegroup`
* `eks:ListClusters`
* `eks:DescribeCluster`

## Asset schema

### EC2 instances

#### Exported fields

| Field                          | Description                                                                                                                                     | Example                                  |
|--------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------|------------------------------------------|
| asset.type                     | The type of asset                                                                                                                               | `"aws.ec2.instance"`                     |
| asset.kind                     | The kind of asset                                                                                                                               | `"host`                                  |
| asset.name                     | The name of EC2 instance                                                                                                                        | `"my_instance"`                          |
| asset.id                       | The id of the EC2 instance                                                                                                                      | `"i-065d58c9c67df73ed"`                  |
| asset.ean                      | The EAN of this specific resource                                                                                                               | `"aws.ec2.instance:i-065d58c9c67df73ed"` |
| asset.parents                  | The EANs of the hierarchical parents for this specific asset resource. For an EC2 instance, this corresponds to the VPC subnet it is related to | `[ "aws.subnet:subnet-b98e46df" ]`       |
| asset.metadata.state           | The state of the EC2 instance                                                                                                                   | `"running"`                              |
| asset.metadata.tags.<tag_name> | Any tag specified for this instance                                                                                                             | `"my label value"`                       |

#### Example

```json
{
    "@timestamp": "2023-05-25T13:48:47.245Z",
    "asset.id": "i-0a76814be7c2ec529",
    "asset.ean": "aws.ec2.instance:i-0a76814be7c2ec529",
    "input": {
      "type": "assets_aws"
    },
    "ecs": {
      "version": "8.0.0"
    },
    "cloud.provider": "aws",
    "cloud.region": "eu-west-1",
    "cloud.account.id": "111111111",
    "asset.metadata.state": "stopped",
    "host": {
      "name": "test"
    },
    "asset.type": "aws.ec2.instance",
    "asset.kind": "host",
    "agent": {
      "type": "assetbeat",
      "version": "8.7.0",
      "ephemeral_id": "111f0ffc-c138-43fd-bea3-e5c298612436",
      "id": "6427b093-afa2-4b1d-9d4a-b3a2273c2719",
      "name": "test"
    },
    "asset.parents": [
      "aws.subnet:subnet-a355daf9"
    ],
    "asset.name" : "my_instance",
    "asset.metadata.tags.Name": "elastic-agent"
  }
```

### EKS clusters

#### Exported fields

| Field                            | Description                                                                                                                                                                                                                     | Example                                                     |
|----------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------|
| asset.type                       | The type of asset                                                                                                                                                                                                               | `"k8s.cluster"`                                             |
| asset.kind                       | The kind of asset                                                                                                                                                                                                               | `"cluster`                                                  |
| asset.id                         | The ARN of the EKS cluster                                                                                                                                                                                                      | `"arn:aws:eks:us-west-1:564797534556:cluster/demo"`         |
| asset.ean                        | The EAN of this specific resource                                                                                                                                                                                               | `"cluster:arn:aws:eks:us-west-1:564797534556:cluster/demo"` |
| asset.name                       | The name of the EKS cluster                                                                                                                                                                                                     | `"my_eks_cluster"`                                          |
| asset.parents                    | The EANs of the hierarchical parents for this specific asset resource. For an EKS cluster, this corresponds to the VPC it is related to                                                                                         | `[ "network:test-vpc" ]`                                    |
| asset.children                   | The EANs of the hierarchical children for this specific asset resource. For a EKS cluster, this corresponds to the EC2 instances it is composed of. **_Note_:** this field is currently not populated for EKS Fargate clusters. | `["host:i-1111111"]`                                        |
| asset.metadata.status            | The state of the cluster                                                                                                                                                                                                        | `"ACTIVE"`                                                  |
| asset.metadata.tags.<label_name> | Any label specified for this cluster                                                                                                                                                                                            | `"my label value"`                                          |

#### Example

```json
{
    "@timestamp": "2023-05-25T13:48:47.538Z",
    "asset.id": "arn:aws:eks:eu-west-1:1111111111:cluster/test-cluster",
    "asset.type": "k8s.cluster",
    "asset.kind": "cluster",
    "input": {
      "type": "assets_aws"
    },
    "agent": {
      "ephemeral_id": "111f0ffc-c138-43fd-bea3-e5c298612436",
      "id": "6427b093-afa2-4b1d-9d4a-b3a2273c2719",
      "name": "test",
      "type": "assetbeat",
      "version": "8.7.0"
    },
    "cloud.account.id": "1111111111",
    "cloud.provider": "aws",
    "asset.metadata.status": "ACTIVE",
    "ecs": {
      "version": "8.0.0"
    },
    "cloud.region": "eu-west-1",
    "asset.children": ["host:i-0805c4e8d9c6015fa", "host:i-0805c4e8d9c6015fb"],
    "asset.parents": [
      "network:vpc-0c7da12158a6c225f"
    ],
    "asset.metadata.tags.Name": "test-cluster/ControlPlane",
    "host": {
      "name": "test"
    },
    "asset.name": "my_cluster",
    "asset.ean": "cluster:arn:aws:eks:eu-west-1:1111111111:cluster/test-cluster"
  }
```

### VPCs

#### Exported fields

| Field                          | Description                                                                         | Example                           |
|--------------------------------|-------------------------------------------------------------------------------------|-----------------------------------|
| asset.type                     | The type of asset                                                                   | `"aws.vpc"`                       |
| asset.kind                     | The kind of asset                                                                   | `"network`                        |
| asset.id                       | The id of the VPC                                                                   | `"vpc-0f754418ce7f991f9"`         |
| asset.name                     | The name of the VPC                                                                 | `"my_vpc"`                        |
| asset.ean                      | The EAN of this specific resource                                                   | `"network:vpc-0f754418ce7f991f9"` |
| asset.metadata.isDefault       | true/false value that indicates if the VPC is the default VPC for the region or not | `"true"`                          |
| asset.metadata.tags.<tag_name> | Any label specified for this VPC                                                    | `"my label value"`                |


#### Example

```json
{
    "@timestamp": "2023-05-25T13:48:47.315Z",
    "asset.type": "aws.vpc",
    "asset.kind": "network",
    "asset.name": "my_vpc",
    "input": {
      "type": "assets_aws"
    },
    "cloud.region": "eu-west-1",
    "asset.metadata.isDefault": false,
    "cloud.provider": "aws",
    "cloud.account.id": "11111111111111",
    "asset.id": "vpc-0f754418ce7f991f9",
    "asset.ean": "network:vpc-0f754418ce7f991f9",
    "ecs": {
      "version": "8.0.0"
    },
    "host": {
      "name": "test"
    },
    "asset.metadata.tags.tag1": "test-tag",
    "agent": {
      "version": "8.7.0",
      "ephemeral_id": "111f0ffc-c138-43fd-bea3-e5c298612436",
      "id": "6427b093-afa2-4b1d-9d4a-b3a2273c2719",
      "name": "test",
      "type": "assetbeat"
    }
  }
```
### Subnets

#### Exported fields

| Field                          | Description                                                                                                                           | Example                              |
|--------------------------------|---------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------|
| asset.type                     | The type of asset                                                                                                                     | `"aws.subnet"`                       |
| asset.kind                     | The kind of asset                                                                                                                     | `"network`                           |
| asset.id                       | The id of the VPC subnet                                                                                                              | `"subnet-0f754418ce7f991f9"`         |
| asset.name                     | The name of the VPC subnet                                                                                                            | `"my_subnet"`                        |
| asset.ean                      | The EAN of this specific resource                                                                                                     | `"network:subnet-0f754418ce7f991f9"` |
| asset.parents                  | The EANs of the hierarchical parents for this specific asset resource. For a VPC subnet, this corresponds to the VPC it is related to | `[ "network:test-vpc" ]`             |
| asset.metadata.tags.<tag_name> | Any label specified for this VPC                                                                                                      | `"my label value"`                   |

#### Example

```json
{
    "@timestamp": "2023-05-25T13:48:47.348Z",
    "asset.id": "subnet-a355daf9",
    "asset.ean": "network:subnet-a355daf9",
    "asset.parents": [
      "network:vpc-db3f2fbd"
    ],
    "cloud.provider": "aws",
    "cloud.region": "eu-west-1",
    "asset.type": "aws.subnet",
    "asset.kind": "network",
    "asset.name": "my_subnet",
    "host": {
      "name": "test"
    },
    "asset.metadata.state": "available",
    "cloud.account.id": "627286350134",
    "input": {
      "type": "assets_aws"
    },
    "agent": {
      "ephemeral_id": "111f0ffc-c138-43fd-bea3-e5c298612436",
      "id": "6427b093-afa2-4b1d-9d4a-b3a2273c2719",
      "name": "test",
      "type": "assetbeat",
      "version": "8.7.0"
    },
    "ecs": {
      "version": "8.0.0"
    }
  }
```