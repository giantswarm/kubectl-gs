# `kubectl-gs` - kubectl Plug-In for Giant Swarm custom resources

Plug-in for `kubectl` to create manifests for creating/updating custom resources for:

- Clusters - see [`docs/template-cluster-cr.md`](https://github.com/giantswarm/kubectl-gs/blob/master/docs/template-cluster-cr.md) for details
- Node pools - see [`docs/template-nodepool-cr.md`](https://github.com/giantswarm/kubectl-gs/blob/master/docs/template-nodepool-cr.md) for details
- App catalogs - see [`docs/template-catalog-cr.md`](https://github.com/giantswarm/kubectl-gs/blob/master/docs/template-catalog-cr.md) for details
- Apps - see [`docs/template-app-cr.md`](https://github.com/giantswarm/kubectl-gs/blob/master/docs/template-app-cr.md) for details


Plugin supports rendering for CRs:
  - Tenat control-plane (AWS only):
    
  - Nodepool (AWS only):
    
  - `AppCatalog`
  - `App`

## Installation

Basically you need the executable in your `$PATH`. Here are ways to accomplish this on Linux and Mac OS:

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
