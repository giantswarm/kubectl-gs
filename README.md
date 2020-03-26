# kubectl-gs

kubectl plugin to render CRs for Giant Swarm clusters.
Plugin supports rendering for CRs:
  - Tenat control-plane (AWS only):
    - `Cluster` (API version `cluster.x-k8s.io/v1alpha2`)
    - `AWSCluster` (API version `infrastructure.giantswarm.io/v1alpha2`)
  - Nodepool (AWS only):
    - `MachineDeployment` (API version `cluster.x-k8s.io/v1alpha2`)
    - `AWSMachineDeployment` (API version `infrastructure.giantswarm.io/v1alpha2`)
  - `AppCatalog`
  - `App`

## How to install plugin

### Linux

```
wget https://github.com/giantswarm/kubectl-gs/releases/download/v0.1.0/kubectl-gs-linux-amd64
chmod +x kubectl-gs-linux-amd64
# mv into /usr/local/bin/ might require sudo
mv kubectl-gs-linux-amd64 /usr/local/bin/kubectl-gs
```

### MacOS

```
wget https://github.com/giantswarm/kubectl-gs/releases/download/v0.1.0/kubectl-gs-darwin-amd64
chmod +x kubectl-gs-darwin-amd64
# mv into /usr/local/bin/ might require sudo
mv kubectl-gs-darwin-amd64 /usr/local/bin/kubectl-gs
```

## How to template CR

 - [cluster CRs](docs/template-cluster-cr.md)
