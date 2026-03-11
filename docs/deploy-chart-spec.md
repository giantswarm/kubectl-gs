# Spec: `kubectl gs deploy chart` version 2026-03-05

## Problem

We have [a tutorial](https://docs.giantswarm.io/tutorials/fleet-management/app-platform/deploy-app-helmrelease/) on deploying an app using a Flux OCIRepository and HelmRelease. There are several user experience issues with that approach:

- The user must use the right Flux CLI version for the Flux version deployed in the cluster.
    - Having a specific CLI version installed is not straightforward.
    - Finding out the required Flux version isn't easy currently, as `flux version` is not supported right now.
- Creating the two required resources takes two `flux create` commands that are not trivial.
- The user must follow the right order of commands, otherwise the source reference will not be found.
- Naming conventions for kubeconfig resources, namespaces, and resource naming must be known by the user.
- In `flux create source oci`, users must specify a regex to match versions to facilitate automatic upgrades (major, minor, or patch).

Additional benefits we can get:

- Configuration validation before applying resources.
- Server-side dry-run for deeper validation of resources, including admission controllers.

## Solution

We provide one subcommand in `kubectl-gs` that allows deploying a chart (or alternatively generating a manifest). As far as possible, defaults will help the user to avoid mistakes.

This command targets 80% of use cases with 20% of complexity. For advanced use cases, users should use the Flux CLI directly, or use this command's `--dry-run` flag to generate a manifest and edit it before applying.

### Syntax

```bash
kubectl gs deploy chart \
    --chart-name my-chart \
    --organization my-org \
    --cluster my-cluster \
    --target-namespace my-namespace
```

### Required flags

- `--chart-name`: Name of the chart to deploy. Combined with `--oci-url-prefix` to form the full OCI URL.
- `--organization`: Giant Swarm organization name owning the target cluster.
- `--cluster`: Target cluster name. Not required when `--management-cluster` is set (determined automatically from context).
- `--target-namespace`: Target namespace in the workload cluster to deploy the Helm release into.

### Optional flags

- `--oci-url-prefix`: First part of the OCI URL for the chart. The `--chart-name` value is appended to it. Default: `oci://gsoci.azurecr.io/charts/giantswarm/`. The leading `oci://` and trailing `/` are optional and will be added automatically if missing.
- `--values-file`: Path to a YAML file with chart values.
- `--values-from`: Reference to an existing ConfigMap/Secret on the management cluster containing chart values.
- `--name`: Override the automatically generated resource names. Default is `<clustername>-<chartname>` for both OCIRepository and HelmRelease.
- `--version`: Chart version to deploy. If not specified, the latest semver-compatible chart version is used.
- `--auto-upgrade`: Instead of pinning to a fixed version, apply a regex for automatic upgrades. Possible values:
    - `all`: Upgrade to latest major, minor, and patch version.
    - `minor`: Upgrade to latest minor and patch version.
    - `patch`: Upgrade to latest patch version.
- `--interval`: Reconciliation interval for both OCIRepository and HelmRelease. Default: `10m`.
- `--management-cluster`: Deploy to the management cluster itself instead of a workload cluster. When set, `--cluster` is not required — the cluster name is determined automatically from the current kubectl context. The HelmRelease omits `.spec.kubeConfig`, so Flux deploys the Helm release locally.
- `--dry-run`: Only generate manifests and print them to stdout. Server-side validation is performed via `kubectl apply --dry-run=server`. Useful for GitOps workflows where manifests are committed to a repository rather than applied directly.

#### Examples

Deploy a chart from the default Giant Swarm registry:

```bash
kubectl gs deploy chart \
    --chart-name my-chart \
    --organization my-org \
    --cluster my-cluster \
    --target-namespace my-namespace
```

Deploy a chart from a custom registry:

```bash
kubectl gs deploy chart \
    --oci-url-prefix oci://example.com/my-namespace/ \
    --chart-name my-chart \
    --organization my-org \
    --cluster my-cluster \
    --target-namespace my-namespace
```

Deploy with a pinned version and automatic patch upgrades:

```bash
kubectl gs deploy chart \
    --chart-name my-chart \
    --version 1.2.3 \
    --auto-upgrade patch \
    --organization my-org \
    --cluster my-cluster \
    --target-namespace my-namespace
```

### Context usage

The command uses the current kubectl context to connect to the management cluster.

TODO: Verification that the context is indeed a Giant Swarm management cluster.

### Version resolution and OCIRepository ref

The version reference strategy on the OCIRepository depends on whether `--auto-upgrade` is set:

**Without `--auto-upgrade` (pinned version):** The OCIRepository uses `ref.tag` to pin to an exact version.

- If `--version` is provided, that exact version is used: `ref.tag: "1.2.3"`
- If `--version` is omitted, the CLI queries the registry for the latest available version and pins to it: `ref.tag: "<latest>"`

**With `--auto-upgrade` (semver range):** The OCIRepository uses `ref.semver` with a range constraint relative to the version specified via `--version` (defaulting to the latest available version):

- `--version 1.2.3 --auto-upgrade patch` — `ref.semver: ">=1.2.3 <1.3.0"`
- `--version 1.2.3 --auto-upgrade minor` — `ref.semver: ">=1.2.3 <2.0.0"`
- `--version 1.2.3 --auto-upgrade all` — `ref.semver: ">=1.2.3"`

### Configuration validation

Configuration provided via `--values-file` is validated against the chart's `values.schema.json` file, if present. The schema is taken from the version of the chart specified via `--version` (defaulting to the latest semver-compatible version).

To find the schema file, the chart metadata is fetched from the OCI registry and the `io.giantswarm.application.values-schema` annotation (if present) is read. Its value is expected to be a public HTTP(S) URL for the `values.schema.json` file.

A config validation error causes the command to exit with an error. If no schema file can be found, no configuration validation is performed.

### Private OCI registry authentication

Authentication for private registries is needed in two places:

1. **Client-side (CLI)**: The CLI must authenticate to the registry to look up chart metadata, fetch available tags, and retrieve the values schema for validation.
2. **Server-side (Flux)**: The command must create a Kubernetes Secret that Flux's OCIRepository can use to pull from the private registry at runtime.

#### Default case (public registries)

No authentication is needed. The default Giant Swarm registry (`gsoci.azurecr.io`) is public.

#### Client-side credential resolution

For client-side operations (metadata lookup, tag listing, schema fetching), the CLI resolves credentials in this order:

1. **Explicit flags** `--registry-username` and `--registry-password` — if provided, used directly.
2. **Docker config file** — the CLI reads `~/.docker/config.json` (and its credential helpers like `docker-credential-ecr-login`, `docker-credential-osxkeychain`, etc.). If the user has previously run `docker login` or `skopeo login` for the target registry, authentication works automatically.
3. **Anonymous** — if no credentials are found, unauthenticated access is attempted.

Implementation: Use oras-go's `credentials.NewStoreFromDocker()` to load the Docker config, wrapped with an `auth.Client` for automatic token exchange. Explicit flags override the Docker config.

#### Server-side secret creation (for Flux)

When credentials are provided (either via flags or Docker config) **and `--registry-provider` is not set**, the command creates a `kubernetes.io/dockerconfigjson` Secret in the organization namespace and references it via `secretRef` on the OCIRepository. This is the same format produced by `kubectl create secret docker-registry` or `flux create secret oci`.

When `--registry-provider` is set to `aws`, `azure`, or `gcp`, this sets `.spec.provider` on the OCIRepository, enabling Flux to authenticate via cloud workload identity (e.g., IRSA on AWS, Workload Identity on Azure/GCP). In this case, no Secret is created — even if `--registry-username`/`--registry-password` are provided (those are used only for client-side lookups).

#### Auth-related flags

- `--registry-username`: Username for private registry authentication.
- `--registry-password`: Password or token for private registry authentication.
- `--registry-provider`: Cloud provider for workload-identity-based auth (`aws`, `azure`, `gcp`). Sets `.spec.provider` on the OCIRepository instead of creating a Secret. For client-side operations (metadata lookup, tag listing), the CLI first tries Docker config credentials. If those are not available, `--registry-username` and `--registry-password` are required — but in this case they are only used for client-side lookups and no Secret is created.

### Flag validation

- `--organization`: We check whether the organization and the org-namespace exist and whether the user has access to them. If not, an error is returned.

### Determining CRD API versions

The command queries the cluster for the available CRD versions of `helmreleases.helm.toolkit.fluxcd.io` and `ocirepositories.source.toolkit.fluxcd.io`, then uses the appropriate API versions in the generated manifests. CRDs are cluster-scoped, so any authenticated user can read them — unlike the Flux deployment namespace which may not be accessible.

### Generated resources

The command creates two Kubernetes resources: an **OCIRepository** and a **HelmRelease**.

- API versions are determined by querying the available CRD versions in the cluster (see "Determining CRD API versions" above).
- Both resources are created in the organization namespace on the management cluster.
- Resource names default to `<clustername>-<chartname>` (overridable via `--name`).

#### OCIRepository

- Points to the full OCI URL (composed from `--oci-url-prefix` + `--chart-name`).
- Version reference: `ref.tag` for pinned versions, `ref.semver` for auto-upgrade ranges (see "Version resolution and OCIRepository ref" above).

#### HelmRelease

- Values inline (not in a ConfigMap). May be revised in the future.
- Kubeconfig secret reference named `<cluster-name>-kubeconfig`.
- Label `giantswarm.io/cluster` set to the cluster name.
- Target namespace will be created if not existing (`createTargetNamespace: true`).

### Example manifests

#### Example 1: Basic deployment with latest version pinned

Invocation:

```bash
kubectl gs deploy chart \
    --chart-name hello-world-app \
    --organization acme \
    --cluster mycluster01 \
    --target-namespace hello
```

Generated OCIRepository:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: OCIRepository
metadata:
  name: mycluster01-hello-world-app
  namespace: org-acme
  labels:
    giantswarm.io/cluster: mycluster01
spec:
  interval: 10m
  url: oci://gsoci.azurecr.io/charts/giantswarm/hello-world-app
  ref:
    tag: "1.2.3"
  provider: generic
```

Generated HelmRelease:

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: mycluster01-hello-world-app
  namespace: org-acme
  labels:
    giantswarm.io/cluster: mycluster01
spec:
  interval: 10m
  releaseName: hello-world-app
  targetNamespace: hello
  install:
    createNamespace: true
  kubeConfig:
    secretRef:
      name: mycluster01-kubeconfig
  chartRef:
    kind: OCIRepository
    name: mycluster01-hello-world-app
```

#### Example 2: Deployment with values file

Invocation:

```bash
kubectl gs deploy chart \
    --chart-name hello-world-app \
    --version 1.2.3 \
    --organization acme \
    --cluster mycluster01 \
    --target-namespace hello \
    --values-file my-values.yaml
```

Contents of `my-values.yaml`:

```yaml
replicaCount: 3
ingress:
  enabled: true
  host: hello.example.com
```

Generated OCIRepository: Same as Example 1.

Generated HelmRelease:

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: mycluster01-hello-world-app
  namespace: org-acme
  labels:
    giantswarm.io/cluster: mycluster01
spec:
  interval: 10m
  releaseName: hello-world-app
  targetNamespace: hello
  install:
    createNamespace: true
  kubeConfig:
    secretRef:
      name: mycluster01-kubeconfig
  chartRef:
    kind: OCIRepository
    name: mycluster01-hello-world-app
  values:
    replicaCount: 3
    ingress:
      enabled: true
      host: hello.example.com
```

#### Example 3: Deployment from a private registry

Invocation:

```bash
kubectl gs deploy chart \
    --oci-url-prefix oci://registry.example.com/charts/ \
    --chart-name private-app \
    --version 1.2.3 \
    --registry-username myuser \
    --registry-password mytoken \
    --organization acme \
    --cluster mycluster01 \
    --target-namespace hello
```

Generated Secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: mycluster01-private-app-registry
  namespace: org-acme
  labels:
    giantswarm.io/cluster: mycluster01
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: <base64-encoded {"auths":{"registry.example.com":{"username":"myuser","password":"mytoken","auth":"bXl1c2VyOm15dG9rZW4="}}}>
```

Generated OCIRepository:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: OCIRepository
metadata:
  name: mycluster01-private-app
  namespace: org-acme
  labels:
    giantswarm.io/cluster: mycluster01
spec:
  interval: 10m
  url: oci://registry.example.com/charts/private-app
  ref:
    tag: "1.2.3"
  provider: generic
  secretRef:
    name: mycluster01-private-app-registry
```

Generated HelmRelease: Same as Example 1.

#### Example 4: Deployment to the management cluster

Invocation:

```bash
kubectl gs deploy chart \
    --chart-name hello-world-app \
    --version 1.2.3 \
    --organization acme \
    --target-namespace hello \
    --management-cluster
```

Generated OCIRepository:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: OCIRepository
metadata:
  name: mymc01-hello-world-app
  namespace: org-acme
  labels:
    giantswarm.io/cluster: mymc01
spec:
  interval: 10m
  url: oci://gsoci.azurecr.io/charts/giantswarm/hello-world-app
  ref:
    tag: "1.2.3"
  provider: generic
```

Generated HelmRelease:

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: mymc01-hello-world-app
  namespace: org-acme
  labels:
    giantswarm.io/cluster: mymc01
spec:
  interval: 10m
  releaseName: hello-world-app
  targetNamespace: hello
  install:
    createNamespace: true
  chartRef:
    kind: OCIRepository
    name: mymc01-hello-world-app
```

Note: No `.spec.kubeConfig` is set, so Flux deploys the Helm release to the same cluster where the HelmRelease resource lives (the management cluster). The cluster name `mymc01` was determined automatically from the current kubectl context.

### Behavior on re-run

If resources already exist, the command applies (overwrites) them. Before applying, the command shows a diff of changes and asks for confirmation.

## Open questions

- `--management-cluster` cluster name derivatio* — "Determined automatically from context" is vague. What's the actual logic?
- How to check that we are in an MC context?
