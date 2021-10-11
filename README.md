![header image](https://user-images.githubusercontent.com/273727/136764760-3c28515d-eb65-4e27-9503-1375dcbf49f0.png)

# The official Giant Swarm kubectl plug-in

## Quick start

```nohighlight
kubectl krew install gs
kubectl gs
```

Check the [installation docs](https://docs.giantswarm.io/ui-api/kubectl-gs/installation/) for details on installation with and without Krew.

## Features

- **Custom resource templating**: using the `template` command lets you create manifests for
  creating/updating custom resources for:
  - Clusters
  - Node pools
  - App catalogs
  - Apps
- **SSO login**: with the `login` command you can quickly set up a `kubectl context` with
  OIDC authentication for a Giant Swarm management cluster.

## Documentation

Find the [kubectl gs reference](https://docs.giantswarm.io/ui-api/kubectl-gs/) in our documentation site.

## Publishing a release

See [docs/Release.md](https://github.com/giantswarm/kubectl-gs/blob/master/docs/Release.md)
