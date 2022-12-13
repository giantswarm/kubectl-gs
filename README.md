![header image](https://user-images.githubusercontent.com/273727/136764760-3c28515d-eb65-4e27-9503-1375dcbf49f0.png)

# The official Giant Swarm kubectl plug-in

## Quick start

```nohighlight
kubectl krew install gs
kubectl gs
```

Check the [installation docs](https://docs.giantswarm.io/use-the-api/kubectl-gs/installation/) for details on installation with and without Krew.

## Features

- **Login via single sign-on**: Using the `login` command, you can quickly set up a `kubectl` context with OIDC authentication for a Giant Swarm management cluster, or a workload cluster with our [dex](https://github.com/giantswarm/dex-app) and [athena](https://github.com/giantswarm/athena) apps installed.
- **Custom resource templating**: using the `template` commands lets you create manifests for
  creating/updating custom resources for:
  - Clusters
  - Node pools
  - App catalogs
  - Apps
- **Gitops repository management**: The `gitops` command family allows to create and modify resources in your GitOps repo clone.
- **Resource display**: The `get` commands allow for retrieving a list of resources, or details for a single one.

## Documentation

Find the [kubectl gs reference](https://docs.giantswarm.io/use-the-api/kubectl-gs/) in our documentation site.

## Publishing a release

See [docs/Release.md](https://github.com/giantswarm/kubectl-gs/blob/master/docs/Release.md)
