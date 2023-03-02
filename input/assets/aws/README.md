# AWS Assets Input

## What does it do?

The AWS Assets Input collects data about AWS resources and their relationships to each other.
Information about the following resources is currently collected:

- EC2 instances
- EKS clusters
- VPNs
- Subnets

These resources are related by a hierarchy of parent/child relationships:

```
┌────────────────────────────────────────────────────────────┐
│                                                            │
│ VPC                                                        │
│                                                            │
│   ┌────────────────────────┐  ┌────────────────────────┐   │
│   │  Subnet                │  │  Subnet                │   │
│   │                        │  │                        │   │
│   │ ┌────────┐ ┌────────┐  │  │ ┌────────┐ ┌────────┐  │   │
│   │ │EC2     │ │EC2     │  │  │ │EC2     │ │EC2     │  │   │
│   │ │Instance│ │Instance│  │  │ │Instance│ │Instance│  │   │
│   │ │        │ │        │  │  │ │        │ │        │  │   │
│   │ └────────┘ └────────┘  │  │ └────────┘ └────────┘  │   │
│   │                        │  │                        │   │
│   └────────────────────────┘  └────────────────────────┘   │
│                                                            │
│ ┌────────────────────────────────────────────────────────┐ │
│ │ EKS Cluster                                            │ │
│ │                                                        │ │
│ │ ┌────────────────────────┐  ┌────────────────────────┐ │ │
│ │ │  Subnet                │  │  Subnet                │ │ │
│ │ │                        │  │                        │ │ │
│ │ │ ┌────────┐ ┌────────┐  │  │ ┌────────┐ ┌────────┐  │ │ │
│ │ │ │EC2     │ │EC2     │  │  │ │EC2     │ │EC2     │  │ │ │
│ │ │ │Instance│ │Instance│  │  │ │Instance│ │Instance│  │ │ │
│ │ │ │        │ │        │  │  │ │        │ │        │  │ │ │
│ │ │ └────────┘ └────────┘  │  │ └────────┘ └────────┘  │ │ │
│ │ │                        │  │                        │ │ │
│ │ └────────────────────────┘  └────────────────────────┘ │ │
│ │                                                        │ │
│ └────────────────────────────────────────────────────────┘ │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

Documents are sent to the following data stream:

```
assets-aws-default
```

## Asset schema

### EC2 instances

Field | Example |
------|---------|
asset.type | `"aws.ec2.instance"`
asset.id | `"i-065d58c9c67df73ed"`
asset.ean | `"aws.ec2.instance:i-065d58c9c67df73ed"`
asset.parents | `[ "subnet-b98e46df" ]`
asset.metadata | `{ "state": "running", "tags": { "Name": "JumpBoxGeneral" } }`

### EKS clusters

Field | Example |
------|---------|
asset.type | `"aws.eks.cluster"`
asset.id | `"arn:aws:eks:us-west-1:564797534556:cluster/demo"`
asset.ean | `"aws.eks.cluster:arn:arn:aws:eks:us-west-1:564797534556:cluster/demo"`
asset.parents | `[ "vpc-0184652a9d65033dd" ]`
asset.metadata | `{ "status": "ACTIVE", "tags": { "alpha.eksctl.io/eksctl-version": "0.114.0" } }`

### VPCs

Field | Example |
------|---------|
asset.type | `"aws.vpc"`
asset.id | `"vpc-0184652a9d65033dd"`
asset.ean | `"aws.vpc:vpc-0184652a9d65033dd"`
asset.metadata | `{ "isDefault": true }`

### Subnets

Field | Example |
------|---------|
asset.type | `"aws.subnet"`
asset.id | `"subnet-0a7698f748686b4c6"`
asset.ean | `"aws.subnet:subnet-0a7698f748686b4c6"`
asset.parents | `[ "vpc-0184652a9d65033dd" ]`
asset.metadata | `{ "state": "available", "tags": { "Name": "Private-DB-Subnet-AZ1" } }`