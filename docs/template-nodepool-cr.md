# Creating a node pool based on custom resources using kubectl-gs

Node pools are groups of worker nodes sharing common configuration. In terms of custom resources they consist of custom resources of type

*On AWS*
- `MachineDeployment` (API version `cluster.x-k8s.io/v1alpha2`)
- `AWSMachineDeployment` (API version `infrastructure.giantswarm.io/v1alpha2`)

*On Azure*
- `MachinePpool` (API version `cluster.x-k8s.io/v1alpha3`)
- `AzureMachinePool` (API version `exp.infrastructure.cluster.x-k8s.io/v1alpha3`)
- `Spark` (API version `core.giantswarm.io/v1alpha1`)

## Usage

To create the manifests for a new node pool, use this command:

    kubectl gs template nodepool

Here are the supported flags:

  - `--provider` - The infrastructure provider (e.g *aws* or *azure*)
  - `--availability-zones` - list of availability zones to use, instead of setting a number. Use comma to separate values. (e.g. `eu-central-1a,eu-central-1b`)
  - `--aws-instance-type`- EC2 instance type to use for workers, e. g. *m5.2xlarge*. (default *m5.xlarge*)
  - `--cluster-id` - tenant cluster ID, generated during running `kubectl gs template cluster`.
  - `--nodepool-name` - nodepoolName or purpose description of the node pool. (default *Unnamed node pool*)
  - `--nodex-max` - maximum number of worker nodes for the node pool. (default 10)
  - `--nodex-min` - minimum number of worker nodes for the node pool. (default 3)
  - `--num-availability-zones` - number of availability zones to use. (default 1)
  - `--owner` - organization, owning tenant cluster. Must be configured with existing organization in installation.
  - `--region` - tenant cluster AWS region. Must be configured with installation region.
  - `--release` - valid release version.
    Can be retrieved with `gsctl list releases` for your installation. Only versions *10.x.x*+ support cluster CRs.
  - `--azure-vm-size` - Azure VM size to use for workers (e.g. *Standard_D4_v3*).
  - `--azure-public-ssh-key` - Azure master machines Base64-encoded public key used for SSH.
  - `--release-branch` - The Giant Swarm releases repository branch to use. (default *master*)

```yaml
---
apiVersion: cluster.x-k8s.io/v1alpha2
kind: MachineDeployment
metadata:
  creationTimestamp: null
  labels:
    cluster-operator.giantswarm.io/version: 2.1.10
    giantswarm.io/cluster: o4omf
    giantswarm.io/machine-deployment: fo2xh
    giantswarm.io/organization: giantswarm
    release.giantswarm.io/version: 11.2.1
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
    aws-operator.giantswarm.io/version: 8.4.0
    giantswarm.io/cluster: o4omf
    giantswarm.io/machine-deployment: fo2xh
    giantswarm.io/organization: giantswarm
    release.giantswarm.io/version: 11.2.1
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
