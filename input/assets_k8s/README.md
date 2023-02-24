# K8s Assets Input

## What does it do?

The K8s Assets Input collects data about  resources running on a K8s cluster.
Information about the following resources is currently collected:

- K8s Nodes


## Asset schema

### K8s Nodes

Field | Example |
------|---------|
"asset.id": "aws:///us-east-2b/i-0699b78f46f0fa248",
"asset.ean": "k8s.node:aws:///us-east-2b/i-0699b78f46f0fa248",
"asset.name": "ip-172-31-29-242.us-east-2.compute.internal",
"asset.type": "k8s.node",
"input": {
    "type": "assets_k8s"
}


In order to run set the following configuration in inputrunner.yml

inputrunner.inputs:
  - type: assets_k8s
    period: 600s
    kube_config: /Users/michaliskatsoulis/go/src/github.com/elastic/inputrunner/kube_config

output.elasticsearch:
  hosts: ["localhost:9200"]
  protocol: "https"
  username: "elastic"
  password: "changeme"
  ssl.verification_mode: "none"


logging.level: info
logging.to_files: false
logging.to_stderr: true
logging.selectors: ["*"]


The kube_config path must contain a kube config file so that if the inputrunner runs as a process anywhere, can access the cluster.
In case it runs as a pod in the same k8s cluster it needs to monitor, then the kube_config is collected from withing the cluster(inClusterconfig)
and the values should be left empty.
