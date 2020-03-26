### Cluster CRs

Cluster CRs can be templated with command `kubectl gs template cluster`. It requires few flags to be configured:

  - `master-az` - AWS availability zone of master instance. 
    Must be configured with AZ of the installation region. E.g. for region *eu-central-1* valid value is *eu-central-1a*.
  - `domain`  - base domain of your installation. Customer solution engineer can provide this value.
  - `name` - cluster name.
  - `owner` - organization, owning tenant cluster. Must be configured with existing organization in installation.
  - `region` - tenant cluster AWS region. Must be configured with installation region.
  - `release` - valid release version. 
    Can be retrieved with `gsctl list releases` for your installation. Only versions *10.x.x*+ support cluster CRs.
  - `--template-default-nodepool` - set to *true* if you want to template nodepool CRs altogether with cluster CRs.


### Nodepool CRs

Nodepool CRs can be templated with command `kubectl gs template nodepool`. It requires few flags to be configured:

  - `availability-zones` - list of availability zones to use, instead of setting a number. Use comma to separate values.
  - `aws-instance-type`- EC2 instance type to use for workers, e. g. *m5.2xlarge*. (default *m5.xlarge*)
  - `cluster-id` - tenant cluster ID, generated during running `kubectl gs template cluster`.
  - `nodepool-name` - nodepoolName or purpose description of the node pool. (default *Unnamed node pool*)
  - `nodex-max` - maximum number of worker nodes for the node pool. (default 10)
  - `nodex-min` - minimum number of worker nodes for the node pool. (default 3)
  - `num-availability-zones` - number of availability zones to use. (default 1)
  - `owner` - organization, owning tenant cluster. Must be configured with existing organization in installation.
  - `region` - tenant cluster AWS region. Must be configured with installation region.
  - `release` - valid release version. 
    Can be retrieved with `gsctl list releases` for your installation. Only versions *10.x.x*+ support cluster CRs.


#### Example

Running command 

```
gs template cluster \
--master-az="eu-central-1a" \
--domain="gauss.eu-central-1.aws.gigantic.io" \
--name="Cluster #2" \
--owner="giantswarm" \
--release="11.0.1" \
--region="eu-central-1" \
--template-default-nodepool
```

Generates output

```
apiVersion: cluster.x-k8s.io/v1alpha2
kind: Cluster
metadata:
  creationTimestamp: null
  labels:
    cluster-operator.giantswarm.io/version: 2.1.1
    giantswarm.io/cluster: o4omf
    giantswarm.io/organization: giantswarm
    release.giantswarm.io/version: 11.0.1
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
    aws-operator.giantswarm.io/version: 8.1.1
    giantswarm.io/cluster: o4omf
    giantswarm.io/organization: giantswarm
    release.giantswarm.io/version: 11.0.1
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
    region: eu-central-1
status:
  cluster:
    id: ""
  provider:
    network:
      cidr: ""
---
apiVersion: cluster.x-k8s.io/v1alpha2
kind: MachineDeployment
metadata:
  creationTimestamp: null
  labels:
    cluster-operator.giantswarm.io/version: 2.1.1
    giantswarm.io/cluster: o4omf
    giantswarm.io/machine-deployment: fo2xh
    giantswarm.io/organization: giantswarm
    release.giantswarm.io/version: 11.0.1
  name: fo2xh
  namespace: default
spec:
  selector: {}
  template:
    metadata: {}
    spec:
      bootstrap: {}
      infrastructureRef:
        apiVersion: infrastructure.giantswarm.io/v1alpha2
        kind: AWSMachineDeployment
        name: fo2xh
        namespace: default
      metadata: {}
status: {}
---
apiVersion: infrastructure.giantswarm.io/v1alpha2
kind: AWSMachineDeployment
metadata:
  creationTimestamp: null
  labels:
    aws-operator.giantswarm.io/version: 8.1.1
    giantswarm.io/cluster: o4omf
    giantswarm.io/machine-deployment: fo2xh
    giantswarm.io/organization: giantswarm
    release.giantswarm.io/version: 11.0.1
  name: fo2xh
  namespace: default
spec:
  nodePool:
    description: Unnamed node pool
    machine:
      dockerVolumeSizeGB: 100
      kubeletVolumeSizeGB: 100
    scaling:
      max: 10
      min: 3
  provider:
    availabilityZones:
    - eu-central-1a
    worker:
      instanceType: m5.xlarge
```