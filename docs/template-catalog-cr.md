# Creating an App Catalog based on custom resources using kubectl-gs

In order to create an App Catalog using custom resources, kubectl-gs will help you create manifests for the resource type:

- `AppCatalog` (API version `application.giantswarm.io/v1alpha1`) - holds the base AppCatalog specification.

## Usage

The command to execute is `kubectl gs template appcatalog`.

It supports the following flags:

  - `--name` - Catalog name.
  - `--description` - Description of the purpose of the catalog.
  - `--url` - URL where the helm repository lives.
  - `--owner` - organization, owning tenant cluster. Must be configured with existing organization in installation.
  - `--logo` - tenant cluster AWS region. Must be configured with installation region.

## Example

Example command:

```
kubectl gs template template appcatalog \
  --name pipo-catalog \
  --description "The Awesome and the Greatest Catalog in the work" \
  --url https://pipo02mix.github.io/my-app-catalog/ \
  --logo https://ca.slack-edge.com/T0251EQJH-U7VFNNB7E-683fc4de3d2e-512
```

Generates output

```yaml
apiVersion: v1
data:
  values: ""
kind: ConfigMap
metadata:
  creationTimestamp: null
  name: pipo-catalog558rf
  namespace: default
---
apiVersion: v1
kind: Secret
metadata:
  creationTimestamp: null
  name: pipo-catalog558rf
  namespace: default
---
apiVersion: application.giantswarm.io/v1alpha1
kind: AppCatalog
metadata:
  creationTimestamp: null
  labels:
    app-operator.giantswarm.io/version: 1.0.0
    application.giantswarm.io/catalog-type: awesome
  name: pipo-catalog
spec:
  config:
    configMap:
      name: pipo-catalog558rf
      namespace: default
    secret:
      name: pipo-catalog558rf
      namespace: default
  description: The Awesome and the Greatest Catalog in the work
  logoURL: https://ca.slack-edge.com/T0251EQJH-U7VFNNB7E-683fc4de3d2e-512
  storage:
    URL: https://pipo02mix.github.io/my-app-catalog/
    type: helm
  title: pipo-catalog
```
