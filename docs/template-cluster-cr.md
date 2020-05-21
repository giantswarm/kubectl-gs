# Creating a cluster based on custom resources using kubectl-gs

In order to create a cluster using custom resources, kubectl-gs will help you create manifests for these resource types:

- `Cluster` (API version `cluster.x-k8s.io/v1alpha2`) - holds the base cluster specification.
- `AWSCluster` (API version `infrastructure.giantswarm.io/v1alpha2`) - holds AWS-specific configuration.
- `G8sControlPlane` (API version `infrastructure.giantswarm.io/v1alpha2`) - specifies the master nodes
- `AWSControlPlane` (API version `infrastructure.giantswarm.io/v1alpha2`) - specifies the master nodes with AWS-specific details

## Usage

The command to execute is `kubectl gs template cluster`.

It supports the following flags:

  - `--master-az` - AWS availability zone(s) of master instance.
    Must be configured with AZ of the installation region. E.g. for region *eu-central-1* valid value is *eu-central-1a*.
    Use the flag once with a single value to create a cluster with one master node. For master node high availability,
    specify three distinct availability zones instead. This can be done by separating AZ names with comma or using the flag
    three times with a single AZ name.
  - `--domain`  - base domain of your installation. Customer solution engineer can provide this value.
  - `--external-snat` - AWS CNI configuration. Only versions *11.3.1+ support this feature.
  - `--name` - cluster name.
  - `--pods-cidr` - CIDR applied to the pods. If you don't set any, the installation default will be applied. Only versions *11.1.4+ support this feature.
  - `--owner` - organization, owning tenant cluster. Must be configured with existing organization in installation.
  - `--region` - tenant cluster AWS region. Must be configured with installation region.
  - `--release` - valid release version.
    Can be retrieved with `gsctl list releases` for your installation. Only versions *10.x.x*+ support cluster CRs.
  - `--template-default-nodepool` - set to *true* if you want to template nodepool CRs altogether with cluster CRs.

Note: If you'd like to add specific worker nodes and not make use of `--template-default-nodepool`, please see [Node pools](https://github.com/giantswarm/kubectl-gs/blob/master/docs/template-nodepool-cr.md) for instructions how to create worker nodes.


## Example

Example command:

```
gs template cluster \
  --master-az="eu-central-1a" \
  --domain="gauss.eu-central-1.aws.gigantic.io" \
  --external-snat=true \
  --name="Cluster #2" \
  --pods-cidr="10.2.0.0/16" \
  --owner="giantswarm" \
  --release="11.2.1" \
  --region="eu-central-1" \
  --template-default-nodepool=false
```

Generates output

```yaml
apiVersion: cluster.x-k8s.io/v1alpha2
kind: Cluster
metadata:
  creationTimestamp: null
  labels:
    cluster-operator.giantswarm.io/version: 2.1.10
    giantswarm.io/cluster: o4omf
    giantswarm.io/organization: giantswarm
    release.giantswarm.io/version: 11.2.1
  name: o4omf
  namespace: default
spec:
  infrastructureRef:
    apiVersion: infrastructure.giantswarm.io/v1alpha2
    kind: AWSCluster
    name: o4omf
    namespace: default
status:
  controlPlaneInitialized: false
  infrastructureReady: false
---
apiVersion: infrastructure.giantswarm.io/v1alpha2
kind: AWSCluster
metadata:
  creationTimestamp: null
  labels:
    aws-operator.giantswarm.io/version: 8.4.0
    giantswarm.io/cluster: o4omf
    giantswarm.io/organization: giantswarm
    release.giantswarm.io/version: 11.2.1
  name: o4omf
  namespace: default
spec:
  cluster:
    description: 'Cluster #2'
    dns:
      domain: gauss.eu-central-1.aws.gigantic.io
    oidc:
      claims:
        groups: ""
        username: ""
      clientID: ""
      issuerURL: ""
  provider:
    credentialSecret:
      name: credential-default
      namespace: giantswarm
    master:
      availabilityZone: eu-central-1a
      instanceType: m5.xlarge
    pods:
      cidrBlock: 10.2.0.0/16
      externalSNAT: true
    region: eu-central-1
status:
  cluster:
    id: ""
  provider:
    network:
      cidr: ""
---
apiVersion: infrastructure.giantswarm.io/v1alpha2
kind: G8sControlPlane
metadata:
  creationTimestamp: null
  labels:
    cluster-operator.giantswarm.io/version: 2.1.10
    giantswarm.io/cluster: o4omf
    giantswarm.io/control-plane: osss7
    giantswarm.io/organization: giantswarm
    release.giantswarm.io/version: 11.2.1
  name: osss7
  namespace: default
spec:
  infrastructureRef:
    apiVersion: infrastructure.giantswarm.io/v1alpha2
    kind: AWSControlPlane
    name: osss7
    namespace: default
  replicas: 1
status: {}
---
apiVersion: infrastructure.giantswarm.io/v1alpha2
kind: AWSControlPlane
metadata:
  creationTimestamp: null
  labels:
    aws-operator.giantswarm.io/version: 8.4.0
    giantswarm.io/cluster: o4omf
    giantswarm.io/control-plane: osss7
    giantswarm.io/organization: giantswarm
    release.giantswarm.io/version: 11.2.1
  name: osss7
  namespace: default
spec:
  availabilityZones:
  - eu-central-1a
  instanceType: m5.xlarge
status: {}
```
