![header image](https://user-images.githubusercontent.com/273727/85553386-2ee41980-b624-11ea-91f9-a6bdfe4d10a8.png)

# The official Giant Swarm kubectl plug-in

## Quick start

With [`krew`](https://krew.sigs.k8s.io/):

```nohighlight
$ kubectl krew update
$ kubectl krew install gs
$ alias kgs="kubectl gs"
$ kgs info
```

Without `krew`:

1. Download the [latest release](https://github.com/giantswarm/kubectl-gs/releases/latest) archive for your platform
2. Unpack the archive
3. Copy the executable to your path
4. Create an alias `kgs` for `kubectl gs`
5. Check it's working

For 64Bit Linux, this should work:

```nohighlight
$ wget https://github.com/giantswarm/kubectl-gs/releases/download/v0.5.0/kubectl-gs_0.5.0_linux_amd64.tar.gz
$ tar xzf kubectl-gs_0.5.0_linux_amd64.tar.gz
$ cp ./kubectl-gs /usr/local/bin/
$ alias kgs="kubectl gs"
$ kgs info
```

## Features

- **Cluster templating**: create manifests for creating/updating custom resources for:
  - Clusters - see [`docs/template-cluster-cr.md`](https://github.com/giantswarm/kubectl-gs/blob/master/docs/template-cluster-cr.md) for details
  - Node pools - see [`docs/template-nodepool-cr.md`](https://github.com/giantswarm/kubectl-gs/blob/master/docs/template-nodepool-cr.md) for details
  - App catalogs - see [`docs/template-catalog-cr.md`](https://github.com/giantswarm/kubectl-gs/blob/master/docs/template-catalog-cr.md) for details
  - Apps - see [`docs/template-app-cr.md`](https://github.com/giantswarm/kubectl-gs/blob/master/docs/template-app-cr.md) for details


## Installation

Basically you need the executable in your `$PATH`. Here are ways to accomplish this on Linux and Mac OS:


## Documentation

- [Creating a cluster](https://github.com/giantswarm/kubectl-gs/blob/v0.5.0/docs/template-cluster-cr.md)
- [Creating a node pool](https://github.com/giantswarm/kubectl-gs/blob/v0.5.0/docs/template-nodepool-cr.md)
- [Installing an App](https://github.com/giantswarm/kubectl-gs/blob/v0.5.0/docs/template-app-cr.md)
- [Installing an App Catalog](https://github.com/giantswarm/kubectl-gs/blob/v0.5.0/docs/template-catalog-cr.md)
