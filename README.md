![header image](https://user-images.githubusercontent.com/273727/85553386-2ee41980-b624-11ea-91f9-a6bdfe4d10a8.png)

# The official Giant Swarm kubectl plug-in

## Quick start

```nohighlight
$ kubectl krew update
$ kubectl krew install gs
$ alias kgs="kubectl gs"
$ kgs info
```

Check the [installation docs](https://github.com/giantswarm/kubectl-gs/blob/master/docs/installation.md) for details on installation with and without Krew.

## Features

- **Custom resource templating**: create manifests for creating/updating custom resources for:
  - Clusters
  - Node pools
  - App catalogs
  - Apps

## Documentation

- [Installation](https://github.com/giantswarm/kubectl-gs/blob/master/docs/installation.md)
- [Creating a cluster](https://github.com/giantswarm/kubectl-gs/blob/master/docs/template-cluster-cr.md)
- [Creating a node pool](https://github.com/giantswarm/kubectl-gs/blob/master/docs/template-nodepool-cr.md)
- [Installing an App](https://github.com/giantswarm/kubectl-gs/blob/master/docs/template-app-cr.md)
- [Installing an App Catalog](https://github.com/giantswarm/kubectl-gs/blob/master/docs/template-catalog-cr.md)

## Publishing a release

See [docs/release.md](https://github.com/giantswarm/kubectl-gs/blob/master/docs/Release.md)
