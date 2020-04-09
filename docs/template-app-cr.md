# Creating an App  based on custom resources using kubectl-gs

In order to create an App using custom resources, kubectl-gs will help you create manifests for the resource type:

- `App` (API version `application.giantswarm.io/v1alpha1`) - holds the base App specification.

## Usage

The command to execute is `kubectl gs template app`.

It supports the following flags:

  - `--name` - App name.
  - `--namespace` - Namespace where the app will be deployed.
  - `--catalog` - Catalog name where the app package is stored. `AppCatalog` CR for this catalog must exist in the cluster.
  - `--cluster` - Cluster ID where app will be installed.
  - `--version` - Version of the app to be installed. The version package must exit in the `AppCatalog` storage.

## Example

Example command:

```
kubectl gs template template app \
  --catalog pipo-catalog \
  --name my-app \
  --namespace default \
  --cluster 2hr7z  \
  --version 0.1.0
```

Generates output

```yaml
apiVersion: application.giantswarm.io/v1alpha1
kind: App
metadata:
  creationTimestamp: null
  labels:
    app-operator.giantswarm.io/version: 1.0.0
  name: my-app
  namespace: default
spec:
  catalog: pipo-catalog
  config:
    configMap:
      name: 2hr7z-cluster-values
      namespace: 2hr7z
    secret:
      name: ""
      namespace: ""
  kubeConfig:
    context:
      name: 2hr7z-kubeconfig
    inCluster: false
    secret:
      name: 2hr7z-kubeconfig
      namespace: 2hr7z
  name: my-app
  namespace: 2hr7z
  userConfig:
    configMap:
      name: ""
      namespace: ""
    secret:
      name: ""
      namespace: ""
  version: 0.1.0
status:
  appVersion: ""
  release:
    lastDeployed: "0001-01-01T00:00:00Z"
    status: ""
  version: ""
```
