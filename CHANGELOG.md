# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project's packages adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Add organisation namepsace to the gitops add command.
- Update `github.com/getsops/sops/v3` from v3.9.2 to v3.9.3.

### Removed

- Removed any code specific to KVM installations.

## [4.7.0] - 2025-01-08

### Changed

- Add deletion prevention label also to templated cluster's `App` and `ConfigMap` if using the `kubectl gs template --prevent-deletion` parameter

## [4.6.0] - 2024-11-28

### Changed

- Use Phase instead of Status field for Clusters and Nodepools. Affected commands:
  - `kubectl gs get clusters`
  - `kubectl gs get nodepools`

### Fixed

- Fix getting nodepools via `kubectl gs get nodepools`.
- Remove node classes from vsphere template used in `kubectl gs template cluster` command.

## [4.5.0] - 2024-11-15

### Changed

- Adjust columns for `kubectl gs get releases`.

## [4.4.0] - 2024-11-13

### Added

- Cloud Director Provider.

## [4.3.1] - 2024-11-04

### Fixed

- Fixed unique user tracking
- Remove debug logging regarding telemetry

## [4.3.0] - 2024-10-28

### Added

- Basic usage tracking data is now collected for every command execution. This should help us maintain and develop the tool. Set the `KUBECTL_GS_TELEMETRY_OPTOUT` environment variable to an arbitrary value to disable this. Data is submitted to [TelemetryDeck](https://telemetrydeck.com/) in the EU. More details are available in our [docs](https://docs.giantswarm.io/vintage/use-the-api/kubectl-gs/telemetry/).

## [4.2.0] - 2024-10-15

### Changed

- **BREAKING** When templating cluster manifests for CAPV clusters with `kubectl gs template cluster` command, now we set the workload cluster release version via the `--release` flag, instead setting cluster-vsphere version via `--cluster-version`.

## [4.1.0] - 2024-09-04

### Added

- Add support for unified cluster-vsphere app. With cluster-vsphere v0.61.0 and newer, default apps are deployed with cluster-vsphere, and default-apps-vsphere app is not deployed anymore.

## [4.0.0] - 2024-08-22

### Changed

- The way to specify a release in `kubectl gs gitops add workload-cluster` has changed. The flag `--cluster-release` has been replaced by `--release`.

### Removed

- `kubectl gs gitops add workload-cluster`:
  - The flag `--default-apps-user-config` has been removed
  - The flag `--default-apps-release` has been removed
  - The flag `--cluster-release` has been removed

## [3.2.0] - 2024-08-12

### Changed

- Use more portable, Bash specific shebang for GitOps pre-commit script template
- Schedule cluster upgrades for CAPI clusters.
- Print Release information in `get cluster` command.

## [3.1.0] - 2024-07-23

### Added

- Add `--prevent-deletion` flag to cluster template command for capv clusters
- Helpful error messages for invalid subnet split parameters of CAPA private clusters

### Changed

- **BREAKING** When templating cluster manifests for CAPZ clusters with `kubectl gs template cluster` command, now we set the workload cluster release version via the `--release` flag, instead setting cluster-azure version via `--cluster-version`.

## [3.0.0] - 2024-06-27

### Removed

- **BREAKING** Remove deprecated `--enable-long-name` flag (affected commands: kubectl gs template cluster/nodepool/networkpool/catalog)

### Changed

- **BREAKING** When templating cluster manifests for CAPA clusters with `kubectl gs template cluster` command, now we set the workload cluster release version via the `--release` flag (like for vintage AWS), instead setting cluster-aws version via `--cluster-version`.
- Update module version to v3.

## [2.57.0] - 2024-06-21

### Added

- Support unified cluster-azure app. With cluster-azure v0.14.0 and newer, default apps are deployed with cluster-azure and default-apps-azure app is not deployed anymore.
- Added `--prevent-deletion` flag to cluster template command for capa, capa-eks, capz clusters

## [2.56.0] - 2024-06-10

### Added

- Allow `kubectl gs update app` to update App CR to any version from any catalog.
- Also add `--suspend` flag to manage Flux App reconciliation.

### Changed

- **BREAKING** `kubectl gs template cluster` for Cluster API provider vSphere has been adapted to work with the values schema of `cluster-vsphere` v0.52.0.

## [2.55.0] - 2024-05-14

### Added

- Support unified cluster-aws app. With cluster-aws v0.76.0 and newer, default apps are deployed with cluster-aws and default-apps-aws app is not deployed anymore.

## [2.54.0] - 2024-05-09

### Changed

- Default value for CAPA Node Pool `rootVolumeSizeGB` was decreased from `300` to `8`.

## [2.53.0] - 2024-04-23

### Changed

- **BREAKING** `kubectl gs template cluster` for Cluster API provider Azure has been adapted to work with the values schema of `cluster-azure` v0.7.0.

## [2.52.3] - 2024-04-23

### Changed

- Make error message actionable in case `kubectl gs template cluster` fails because the user did not log into, or point to, the management cluster
- Support internal api URLs in `kubectl gs login` id token verification
- Print a warning in case `kubectl gs login` id token verification fails but don't fail the command

## [2.52.2] - 2024-03-26

### Added

- Add `kubectl gs get nodepools` for CAPA,CAPZ,CAPV,EKS and CAPVCD.
- Add validation of ID token retrieved from OIDC provider during `kubectl gs login`

### Changed

- Errors during update checks no longer interrupt the command execution.
- Fix authentication failure in case the browser sends multiple requests to the callback server during the `login` command execution

## [2.52.1] - 2024-02-01

No significant changes compared to v2.52.0. This release was made to ensure the proper distribution to all channels, which failed with the last release.

## [2.52.0] - 2024-01-25

### Added

- Allow subnet generation customization for CAPA clusters.

## [2.51.0] - 2024-01-10

### Changed

- Remove bastion section for generating CAPA cluster manifests.

## [2.50.1] - 2023-12-13

### Changed

- Update values schema for generating EKS cluster.

## [2.50.0] - 2023-12-12

### Breaking changes

- `kubectl gs template cluster`: Either `--name` or new `--generated-name` parameter is now required for CAPI cluster names. We kept the CLI backward-compatible for vintage, so if none of these parameters is specified, the old default of generating a random name still applies and no error is thrown.

## [2.49.1] - 2023-12-06

## [2.49.0] - 2023-12-05

### Changed

- **BREAKING** All values of cluster userconfig for `CAPA` are moving under `global`.

## [2.48.1] - 2023-11-30

### Changed

- Changed the length of randomly-generated cluster names to 10

## [2.48.0] - 2023-11-29

### Added

- Add support for device authentication flow in the `login` command and a new `--device-auth` flag to activate it.

### Fixed

- Fix storage of separate kubeconfig file (`--self-contained`) for EKS workload clusters

## [2.47.1] - 2023-11-15

### Changed

- Increase the cluster name length to 20 characters.
- Change how `login` works to use our DNS record for the k8s API when using non-vintage providers, rather than the value found in the CAPI CRs.
- Defaults for `cluster template --provider vsphere` since it was migrated to flatcar os.

## [2.47.0] - 2023-11-13

### Changed

- Change default volume Docker size to 10Gb on AWS vintage NodePools.

## [2.46.0] - 2023-11-08

### Added

- Add CAPA cluster templating parameter `--control-plane-load-balancer-ingress-allow-cidr-block` which automatically adds NAT Gateway IPs of the MC to the allowlist

## [2.45.4] - 2023-11-08

### Added

- Added a bash script to generate self-contained kubeconfig files with client certificate for workload clusters in Vintage installations using device auth flow in Dex

## [2.45.3] - 2023-10-26

## [2.45.3] - 2023-10-26

## [2.45.2] - 2023-10-26

## [2.45.1] - 2023-10-26

### Changed

- Upgrade K8s dependencies ([#1149](https://github.com/giantswarm/kubectl-gs/pull/1149)).
  - Upgrade giantswarm/k8sclient to 7.1.0.
  - Upgrade sigs.k8s.io/cluster-api to v1.5.2.
  - Migrate sigs.k8s.io/cluster-api-provider-aws to v2 (2.2.4).
  - Upgrade sigs.k8s.io/cluster-api-provider-azure to v1.11.4.
  - Upgrade sigs.k8s.io/controller-runtime to v0.16.3.
  - Upgrade github.com/coreos/go-oidc/v3 to v3.6.0.
  - Upgrade other dependencies.
  - Replace capi.ClusterLabelName with capi.ClusterNameLabel.

## [2.45.0] - 2023-10-25

### Added

- `cluster template` for up-to-date vsphere cluster app.

## [2.44.0] - 2023-10-16

### Changed

- Make the `--organization` flag visible when templating App CR.

### Fixed

- `kubectl-gs login`: listen only on localhost for callbacks

## [2.43.0] - 2023-10-11

### Added

- Support deletion prevention for `kubectl gs template app` command

## [2.42.0] - 2023-10-06

### Fixed

- Removed `controlPlane.replicas` value for CAPA since it is not supported anymore

### Removed

- Removed support for private DNS mode for proxy based CAPA clusters

## [2.41.1] - 2023-09-19

### Changed

- Change how `login` works on CAPA and `gcp` to use our DNS record for the k8s API when using these providers, rather than the value found in the CAPI CRs.

## [2.41.0] - 2023-08-16

### Added

- Adding `opsctl login` support for EKS clusters.

## [2.40.0] - 2023-08-09

### Added

- Add `--login-timeout` flag to control the time period of OIDC login timeout
- Add experimental support for templating `cluster-eks` with provider `eks`.

### Changed

- Graceful failure of the `login` command in case workload cluster API is not known
- Improved error message after login timeout
- Adjusted description of the `--cluster-admin` flag in the `login` command

## [2.39.0] - 2023-06-22

### Breaking changes

- Add minimal support for templating CAPZ clusters by command line parameters. This removes `--cluster-config` and `--default-app-config` parameters which required handcrafted YAML input. It leaves one consistent templating option for CAPI products (`kubectl gs template cluster --provider ... --other-params`).

## [2.38.0] - 2023-06-14

### Changed

- App: Rename `nginx-ingress-controller-app` to `ingress-nginx`. ([#1077](https://github.com/giantswarm/kubectl-gs/pull/1077))
- vSphere: Fix templating. ([#1079](https://github.com/giantswarm/kubectl-gs/pull/1079))

### Fixed

- Sanitize file passed as inputs data for config maps by stripping spaces from the right end of the lines.
  - This makes the output to use proper multi-line strings for the embedded YAML content by working around a bug in `sig.k8s.io/yam@v1.3.0`

## [2.37.0] - 2023-05-17

### Changed

- Use non-exp apiVersion for azure machine pool types in `template nodepool`.

## [2.36.1] - 2023-05-17

### Fixed

- Setting `spec.config.configMap` in `app/<cluster-name>-default-apps` for `CAPZ` clusters.

## [2.36.0] - 2023-05-04

### Changed

- Add support for `--proxy` and `--proxy-port` flags to `login cmd` to enable `proxy-url: socks5://localhost:9000` in the cluster section of the configuration added to kubeconfig
  - This is only supported for `clientcert` Workload Clusters

## [2.35.0] - 2023-04-17

### Changed

- Stop using old `v1alpha3` version when using CAPI CRDs.

## [2.34.1] - 2023-03-30

### Fixed

- `kubectl gs template cluster` now by default creates a node pool with the name `nodepool0`, instead of `machine-pool0`, as the latter is no longer valid according to the cluster-aws schema as of v0.24.0.

## [2.34.0] - 2023-03-23

### Added

- `cluster template` supports a generic way to template `CAPI` based clusters where all the input parameters are given as `values.yaml` from the corresponding `cluster` and `default-apps` chart.
- CAPZ: removed unmaintained `CAPZ` implementation and switched to the generic templating implementation.

### Changed

- `kubectl gs template cluster` for Cluster API provider AWS has been adapted to work with the values schema of cluster-aws v0.28.0.

## [2.33.0] - 2023-03-08

### Added

- Add workload cluster login support for `CAPZ` based clusters
- CAPA: Add hidden flags `--aws-prefix-list-id` and `--aws-transit-gateway-id` for private clusters

### Changed

- CAPA: Renamed hidden parameter `--role` to `--aws-cluster-role-identity-name` and adapted manifest output to the new name `awsClusterRoleIdentityName` (see [cluster-aws](https://github.com/giantswarm/cluster-aws/pull/192) change)

## [2.32.0] - 2023-02-02

As part of our automatic upgrades journey, we have learnt that cluster chart should structure in such a way that allows overwriting all sections in different levels

**Warning:** This results in a **breaking change** in the template output of a `capa` clusters machine pools.

### Changed

- Breaking: Update `capa` machine pools to template usings an object instead of arrays as of cluster-aws `v0.24.0`

## [2.31.2] - 2023-02-02

### Fixed

- Fixed creating client certificates for workload clusters in `capvcd` installations.

## [2.31.1] - 2023-01-19

### Changed

- Updated CAPA template output to support new subnet layout as of cluster-aws v0.21.0
- Change default folder for cluster base templates generated by `gitops add base` from `bases/clusters/<PROVIDER>` to `bases/clusters/<PROVIDER>/template`

### Added

- Add default machine pool name for `gitops add base` generated templates to `capa` and `gcp` (CAPG) providers with value: `machine-pool0`

## [2.31.0] - 2023-01-18

### Changed

- Adjusted communication with Dex in the `login` command to provide an option to choose from multiple connectors
- Modified detection of installation providers - downloading the provider information from Athena with a fallback to the old detection from API URLs
- Added a new provider - `cloud-director`

### Added

- Introduced a new `--connector-id` flag in the `login` command to specify a Dex connector to use and skip the selection step
- Ensured that ID tokens needed for OIDC authentication are renewed only when they expire

## [2.30.0] - 2023-01-12

### Added

- Add flags `--cluster-type`, `--https-proxy`, `--http-proxy`, `--no-proxy`, `--api-mode`, `--dns-mode`, `--vpc-mode` and `--topol
- ogy-mode` to `template cluster` that specify `capa` as provider.
- Add `gitops add base` command to generate CAP[A,G,O] bases. The values for `--provider` flag is compatible with the `template cluster` command (A: capa, G: gcp, O: openstack).

## [2.29.5] - 2022-12-20

### Changed

- Extended detection of providers in the login command to take the provider value primarily from Athena with fallback to the original way of inspecting the API URL

## [2.29.4] - 2022-12-15

### Fixed

- Respect `--control-plane-instance-type` for AWS cluster templating. Previously, the default value `m5.xlarge` was always used.

## [2.29.3] - 2022-12-08

### Fixed

- Fixed logging in to clusters running on custom domains by removing domain restriction from API URL validation

## [2.29.2] - 2022-12-02

### Fixed

- Add missing namespace in SOPS related secrets created by GitOps subcommands.

## [2.29.1] - 2022-11-30

### Fixed

- Fix `update cluster` command when scheduling an upgrade for a Cluster when the Cluster CR had no previous annotations.

## [2.29.0] - 2022-11-24

- Ensure dedicated `cert-operator` version `0.0.0` is used for client certificate creation in `login` command to avoid timeouts.
- Adjusted GCP config to support the volume type for all kind of volumes (root, etcd, kubelet, containerd)

## [2.28.2] - 2022-11-16

## [2.28.1] - 2022-11-09

### Changed

- Use `n1-standard-4` as default instance types for CAPG worker nodes.
- Adjusted behaviour of the `login` command to prevent updates of the main kubeconfig file in case there are no changes in access tokens and/or the current context, or if the current context is provided via override (e.g. by using the `--context` flag).

## [2.28.0] - 2022-11-09

### Fixed

- Fixed a bug in CLI output tests that made them fail randomly

## [2.27.0] - 2022-10-25

### Changed

- Disable `kubectl gs template nodepool` command for Cluster API (CAPI) based workload Clusters.

## [2.26.1] - 2022-10-24

### Fixed

- Avoid panic in `get nodepools` when node pool is lacking the release version label.
- When templating Vintage Azure Cluster, use Flatcar version from the Release CR rather than hardcoded one.

## [2.26.0] - 2022-10-20

### Fixed

- Change module name to `github.com/giantswarm/kubectl-gs/v2`.

### Changed

- Upgraded dependencies

## [2.25.0] - 2022-10-19

### Changed

- Use the `cluster-values` configmap when templating the `default-apps-aws` app.

### Fixed

- Fixed a bug in modifying existing entries in self-contained config files where new data for the existing entries failed to be written to the file.

## [2.24.2] - 2022-10-13

### Fixed

- Fixed a bug in `login` command where the `issuer` URL was used instead of the `server` address in login retry attempt.

### Added

- Add flags `--gcp-machine-deployment-sa-email` and `--gcp-machine-deployment-sa-scopes` to `template cluster` that specify a Google Cloud Platform service account and its scope to a cluster's machine deployments
- Added read header timeout to http server

### Changed

- Adjusted `kubectl gs login` command to ensure that it writes to the main kubeconfig file only in case there are actual changes in the content of the file.

## [2.24.1] - 2022-10-12

### Fixed

- Fix login command failing because opening the browser blocks and callback webserver does not start on some operating systems

## [2.24.0] - 2022-10-10

### Changed

- Add `giantswarm.io/cluster` label to the 'default-apps' bundle so that it's deleted when a `Cluster` is deleted.
- Deprecated `--enable-long-names` flag and added support for generating and validating long resource names (up to 10 characters) by default.
- Add option to reference the `cluster-values` configmap in the `App` CR created for CAPI clusters.

## [2.23.2] - 2022-10-04

### Added

- Add timeouts support to App CR.
- Added support for deriving internal API hostname from workload cluster main API URLs

## [2.23.1] - 2022-09-27

### Fixed

- `kubectl gs template app` help text: Replace deprecated `--cluster` flag by new `--cluster-name`.
- Fixed generating common names for workload cluster certificates from internal management cluster API URLs in `kubectl gs login --workload-cluster --internal-api ...`

## [2.23.0] - 2022-09-22

### Added

- Added `--cn-prefix` flag to `login` command which allows setting a specific CN prefix for workload cluster client certificates.

## [2.22.0] - 2022-09-14

### Changed

- Renamed local flags, whose names conflicted with global flags and deprecated local flags with the old names
  - `--namespace` in `kubectl gs gitops add app` has been deprecated and replaced with `--target-namespace`
  - `--namespace` in `kubectl gs template app` has been deprecated and replaced with `--target-namespace`
  - `--cluster` in `kubectl gs template app` has been deprecated and replaced with `--cluster-name`
  - `--namespace` in `kubectl gs template catalog` has been deprecated and replaced with `--target-namespace`

### Added

- Added a test to detect local flags with names conflicting with global flag names

## [2.21.0] - 2022-09-08

### Changed

- Switched from exp to non-exp apiVersion for `MachinePools` and `AzureMachinePools` CR on `Azure` in `get nodepool` command.

## Added

- Added `organizations` subcommand to `kubectl gs get` family of commands to list and display details of organizations

## [2.20.0] - 2022-09-02

## Added

- Introduced `kubectl gs gitops` family of commands.

## [2.19.3] - 2022-08-23

### Fixed

- Set domain name for the Kubernetes APIs server address when logging in to CAPI provider workload clusters.


## [2.19.2] - 2022-08-17


### Fixed

- Fixed common name in certificates generated for workload clusters by stripping https:// prefix from cluster base path

## [2.19.1] - 2022-08-17

### Fixed

- Fix nil pointer panic in `template nodepool` command.

# [2.19.0] - 2022-08-12

### Changed

- Make all `kubectl` config flags (e.g `--context` and `--kubeconfig` global and unify kubeconfig management throughout commands.
- Remove CAPA templating from `aws` provider.
- Add new provider `capa` for templating a cluster.
- Remove fetching ssh sso ca pub key for capa from management cluster.
- Add test for CAPA provider cluster templating.

## [2.18.0] - 2022-07-08

### Added

- In the `login` command, allow concatenation of contexts in destination file when creating WC client certificates with `--self-contained` flag.

## [2.17.0] - 2022-07-07

### Added

- Allow calling `login` command with a second argument to select WC contexts.
- Add `-clientcert` suffix to WC client certificate contexts created by the `login` command. Fall back to `-clientcert` context selection if no other context exists for a cluster.
- Use `CertificateAuthorityData` to store CA data and ensure that `CertificateAuthority` is not set when manipulating the kubeconfig in the `login` command.

## [2.16.0] - 2022-07-01

### Changed

- Command `template cluster --provider gcp` no longer tries to get SSH SSO public key secret in the `giantswarm` namespace

## [2.15.0] - 2022-06-22

### Added

- Add flags `--gcp-control-plane-sa-email` and `--gcp-control-plane-sa-scopes` to `template cluster` that specify a Google Cloud Platform service account and its scopes to a cluster's control plane nodes

### Removed

- Removed `giantswarm.io/cluster` label from the default apps bundle and the `App` representing a CAPI cluster.

## [2.14.0] - 2022-06-15

### Added

- Added flag `--service-priority` to `template cluster` command that allows setting the service priority label.
- Updated `kubectl gs template catalog` to support multiple repository mirrors.

## [2.13.2] - 2022-06-09

### Removed

- Remove `giantswarm.io/managed-by: flux` for App CRs labeled for unique App Operator.

## [2.13.1] - 2022-06-09

### Added

- In the `template app` command, add the `giantswarm.io/cluster` label to in-cluster App CR when requested by the user.

## [2.13.0] - 2022-06-09

### Added

- Add `service-priority` label value `highest` by default to `vintage` clusters
- Add `SERVICE PRIORITY` column to `get clusters` command table output.

## [2.12.1] - 2022-06-08

### Fixed

- Take `--context` flag into account when building config for `login`.

## [2.12.0] - 2022-06-02

### Changed

- When loging in, take the k8s API endpoint from the `Cluster` CR rather than calculating it.
- Make `kubectl gs login` to work on GCP clusters.

## [2.11.2] - 2022-05-26

### Fixed

- Pass region flag to template config

## [2.11.1] - 2022-05-25

### Fixed

- Use provided name as cluster name when using `kubectl-gs template cluster --provider gcp`

## [2.11.0] - 2022-05-19

### Added

- Add a NOTES column to the output of the `get apps` command. The column contains information why the last Helm release attempt failed if so, empty otherwise.

## [2.10.0] - 2022-05-13

### Added

- Allow to reuse any current context in `login` command by omitting the argument. This allows creating clientCerts for WCs in an arbitrary MC context. (not following `gs-codename` format)

## [2.9.1] - 2022-05-06

### Changed

- Fix retry fetching clientcert secret in the `default` namespace for legacy azure clusters.

## [2.9.0] - 2022-05-05

### Changed

- base64 encode ssh key for CAPZ clusters.
- Fix bastion systemd unit on CAPZ clusters.
- Make CAPZ clusters compatible with cluster-apps-operator new version.

## [2.8.1] - 2022-05-03

### Fixed

- Fixed missing `Kind` in `template` command.

## [2.8.0] - 2022-05-03

### Changed

- Bump CAPI (cluster-api) dependencies to v1beta1

## [2.7.11] - 2022-04-20

### Fixed

- Disable colored output on Windows to avoid printing of ANSII escape codes.

## [2.7.10] - 2022-04-20

- Build signed Windows binaries
- Extend CI config to include the Windows package in the Krew index when a new release is published

## [2.7.1] - 2022-04-14

### Changed

- Improved description of the `--control-plane-az` parameter when templating a cluster.

## [2.7.0] - 2022-04-01

- In `kubectl gs login`, add support for workload clusters on OpenStack.

## [2.6.0] - 2022-03-31

### Added

- Add templating for clusters using Cluster API provider Google Cloud (CAPG).

### Changed

- Make the region and availability zones flags optional for CAPA clusters.

## [2.5.0] - 2022-03-23

### Added

- Add cluster name label to Cluster API provider AWS (CAPA) Apps and ConfigMaps created with `kubectl-gs template`

## [2.4.0] - 2022-03-21

### Added

- Add tests for `kubectl gs login`.
- Add `--visibility` flag to `template catalog` to add label to control display in web UI.

### Fixed

- Look up cluster-related AppCatalogEntries in the `giantswarm` namespace instead of the `default` namespace.

## [2.3.1] - 2022-03-11

### Fixed

- Set correct labels of GiantSwarm components on cluster templates.

### Changed

- `login`: simplify description for the `--certificate-ttl` flag.

## [2.3.0] - 2022-03-09

### Added

- Add description column to the `get catalog` limited to 80 characters.
- Add `--enable-long-names` feature flag to `template cluster/networkpool/nodepool` to allow resource names longer than 5 characters. Only for internal testing.
- Implement `get clusters` command for OpenStack.

### Changed

- Add missing availability zones to cluster configuration for OpenStack.
- Change default catalog for `cluster-*` and `default-apps-*` apps from `giantswarm` to `cluster`.

## [2.2.0] - 2022-03-04

### Added

- Add OIDC flags to the `template cluster` command (OpenStack only).

### Changed

- Improve flag handling and naming for `template cluster` command (no user facing changes).
- Add new flags for `template cluster --provider-openstack` to be able to use existing networks and subnets.
- Update the kubectl version in Dockerfile

## [2.1.1] - 2022-02-25

### Fixed

- Fixed crash if listing nodepools when one is missing the release version label.
- Add audit log configuration file to the `KubeadmControlPlane` CR.
- Use the CAPZ controller manager env vars for control-plane identity when authenticating to Azure API.

## [2.1.0] - 2022-02-08

### Fixed

- `login` command: Try logging in again if token renewal fails.
- Add `security` API group to scheme in order to get `organizations` during `login`.

### Changed

- Enable logging into clusters in all versions and namespaces if `--insecure-namespace` flag is active.
- Simplify log in with context name

### Added

- Add support for self-contained kubeconfig creation for management cluster context.
- Add `--keep-context` flag to `login`.

## [2.0.0] - 2022-02-04

### Changed

- Enable `cluster-topology` templates for OpenStack by default.
- Update default `cluster-openstack` version to 0.3.0.

### Removed

- Remove deprecated `--cluster-id` flag from `get nodepools`, `template cluster`, and `template nodepool` commands. Replaced by `--cluster-name`.
- Remove deprecated `--owner` flag from `template cluster`, `template networkpool`, and `template nodepool` commands. Replaced by `--organization`.
- Remove deprecated `--master-az` flag from `template cluster` command. Replaced by `--control-plane-az`.
- Remove deprecated `--nodepool-name` flag from `template nodepool` command. Replaced by `--description`.
- Remove deprecated `--nodex-min` flag from `template nodepool` command. Replaced by `--nodes-min`.
- Remove deprecated `--nodex-max` flag from `template nodepool` command. Replaced by `--nodes-max`.

### Added

- Add support for templating App CRs in organization namespace.
- Add `--catalog-namespace` flag to `template app`.

## [1.60.0] - 2022-01-27

### Changed

- Use `v1beta1` api version when templating ClusterAPI manifests on Azure.

## [1.59.0] - 2022-01-26

### Added

- Add support to `template cluster --provider openstack` for templating clusters as App CRs.

## [1.58.2] - 2022-01-13

### Added

- Add `--in-cluster` flag to `template app` command to support installation of MC apps.

### Fixed

- `login` command: Prevent deletion of all CertConfig resources in a namespace, instead delete only one.
- Adjust `login` to consider other prefixes while parsing the MC API endpoint.

## [1.58.1] - 2021-12-17

### Fixed

- Populate the nodepool release label for AWS provider

## [1.58.0] - 2021-12-14

### Added

- Add support cluster updates and scheduling cluster updates.

## [1.57.0] - 2021-12-09

### Changed

- Modify `STATUS` column of `get releases` command table output to display release state.

## [1.56.0] - 2021-12-07

### Added

- Add support for the new URL scheme `api.INSTALLATION.OWNER_ID.gigantic.io` for `kubectl-gs login` command.

## [1.55.0] - 2021-12-06

### Added

- Add alpha support for OpenStack cluster templating.

## [1.54.0] - 2021-12-03

### Fixed

- Fix a problem preventing the `login` command from creating a client certificate for older workload clusters on Azure.
- Fix the problem where the `template cluster` output for a v20 Cluster API cluster on AWS contained a bad infrastructure reference, resulting in the cluster not being provisioned.

## [1.53.0] - 2021-11-29

### Changed

- Disable version caching for the `selfupdate` command, so you will always get the latest version right after it's released.
- Make the `--release` flag mandatory in the `template cluster` and `template nodepool` subcommands.

## [1.52.0] - 2021-11-23

### Changed

- Replace the `CREATED` column with `AGE` in all the `get` subcommand table outputs.

## [1.51.0] - 2021-11-18

### Added

- Add the ability of executing the management cluster login part of the `login` command with a `ServiceAccount` token.

## [1.50.1] - 2021-11-17

### Fixed

- Strip ':<port>' suffix when requesting a client certificate.

## [1.50.0] - 2021-11-17

### Added

- Add `--self-contained` flag to `kubectl-gs login` command for workload clusters to allow output of standalone kubeconfig file.

## [1.49.0] - 2021-11-16

### Changed

- Validate `--certificate-ttl` flag of the `login` command.

## [1.48.1] - 2021-11-11

### Fixed

- Fix self-update command suggestion in the update warning.

## [1.48.0] - 2021-11-11

### Changed

- Allow using `ServiceAccount` tokens for creating workload cluster certificates.
- Let users override their kubectl config using flags in the `login` command.

## [1.47.0] - 2021-11-09

### Added

- Print warning after running any command if there is a newer version available.
- Implement command for self-updating (`kubectl gs selfupdate`).

### Changed

- Make the `--organization` flag optional when using the `login` command with a workload cluster. The cluster will be searched in all the organization namespaces that the user has access to.

## [1.46.0] - 2021-11-09

### Added

- Find `Cluster` resources on AWS based on the `giantswarm.io/cluster` label if the `cluster.x-k8s.io/cluster-name` label does not yield results.
- Add `cluster.x-k8s.io/cluster-name` label to all CRs created by AWS Cluster and Nodepol templating.

### Changed

- Usa CAPI templates for all releases from `v20.0.0-alpha1` onwards, to include alpha and beta releases.
- Move AWS Cluster templating from `apiextensions`
- Move AWS Node Pool templating from `apiextensions`

## [1.45.0] - 2021-10-26

### Added

- Add support for updating `App` CRs.

## [1.44.0] - 2021-10-25

### Added

- Add support for creating workload cluster client certificates using the `login` command.

## [1.43.1] - 2021-10-15

### Fixed

- Fix a problem with fetching Catalog CRs in `validate apps`.
- Fixing a problem where the function to fetch the SSH secret to generate the templates was using `inCluster` config ignoring the kubeconfig.

## [1.43.0] - 2021-10-13

### Added

- Add templating for EKS clusters using the management cluster API
- Add templating for EKS node pools using the management cluster API
- Add templating for CAPA node pools using the management cluster API

### Changed

- In the `get catalogs` command output, rename the colum `APP VERSION` to `UPSTREAM VERSION` and change the column order.

## [1.42.1] - 2021-10-08

### Fixed

- Fix a problem where the template subcommands would be slower than expected because of obsolete API requests.

## [1.42.0] - 2021-10-07

### Added

- Add CRs to create a bastion host in CAPZ cluster template.
- Enable termination events for CAPZ node pools.

## [1.41.1] - 2021-10-04

### Changed

- Use org-namespace for AWS Clusters by default

## [1.41.0] - 2021-10-04

### Added

- Nodepool nodes are labeled with nodepool id on AWS using `giantswarm.io/machine-pool`.
- `MachinePool` and `AzureMachinePool` are labeled with the `giantswarm.io/machine-pool` label.
- `get releases` command to return details of available releases.

## [1.40.0] - 2021-09-24

### Added

- Nodepool nodes are labeled with nodepool id on Azure using `giantswarm.io/machine-pool`.

### Changed

- Update the `template cluster` command to add CAPI defaults and validation using the management cluster API.

## [1.39.2] - 2021-09-17

### Changed

- In the `template cluster` and `template nodepool` commands, the `--owner` flag got replaced by `--organization`.

## [1.39.1] - 2021-09-14

### Added

- The `template organization` command now also offers an `--output` flag to specify an output path optionally.

### Changed

- In the `template` commands, the flag `--owner` is replaced by `--organization`.
- Make the `login` command be able to start a new authentication flow if one of the tokens of an existing authentication provider are not present.
- Update cluster templating error messages for `--name` flag to use correct terminology

## [1.39.0] - 2021-09-10

### Added

- Add support for templating `Organization` CRs.

### Changed

- Allow providing Kubernetes API URLs with prefixes to the `login` command.

## [1.38.0] - 2021-09-08

### Added

- Add tags to enable `cluster autoscaler` to Azure Node Pool template.
- Enable system assigned identity in Azure CAPI clusters' master nodes.
- Set routing table in master subnet in Azure CAPI clusters.

### Fixed

- Set `cluster.giantswarm.io/description` annotation for `Cluster` CR in template generation command on Azure.
- Set `machine-pool.giantswarm.io/name` annotation for `MachinePool` CR in template generation command on Azure.

## [1.37.0] - 2021-09-03

### Changed

- Template cluster and nodepool resources in the org-namespace from AWS release `16.0.0` onwards.

### Added

- Added `aws-cluster-namespace` flag for nodepools to override the standard namespace to support nodepool creation for
  upgraded >v16.0.0 clusters that remain in the default namespace.
- Added support to generate templates for CAPZ clusters and node pools.

## [1.36.0] - 2021-08-26

### Added

- `kubectl gs login` now offers the flag `--callback-port` to specify the port number the OIDC callback server on localhost should use.

## [1.35.1] - 2021-08-24

### Added### Added

- Update the Dockerfile to include kuebctl v1.19 and be based on Alpine v3.14.1.

### Fixed

- Make the `login` command continue to work even if opening the default browser fails.

## [1.35.0] - 2021-08-11

### Changed

- Apply only `v1alpha3` CRs on provider AWS.

### Added

- Add CRs to create bastion host, when creating a CAPI cluster via `template cluster` command.
- Add configuration to allow SSH for Giant Swarm employees when creating CAPI cluster. Applies to `template cluster` and `template nodepool` commands.
- Update template version for CAPA cluster and nodepool templating to version 0.6.8

## [1.34.0] - 2021-07-30

Throughout our UIs and documentation we are aligning our terminology regarding cluster and node pool details, to use consistent terminology matching our Management API. The unique, immutable identifier that was formerly called ID in our user interfaces, is now called the name. The user-friendly, changeable description of the cluster's and node pool's purpose was called name in our UIs and is now called the description.

**Warning:** This terminology change results in a **breaking change** in the `template cluster` command, as the purpose of the flag `--name` has changed. Also several flags in other commands have been deprecated.

If you are upgrading from an earlier releases, apply these changes to migrate any scripts:

- When using `template cluster`, replace `--name` with `--description` to set the user-friendly cluster description, and replace `--cluster-id` with `--name` to set the cluster's unique identifier.
- When using `template nodepool`, replace `--cluster-id` with `--cluster-name`.

### Added

- `template app`: Added the `--namespace-annotations` and `--namespace-labels` flags to allow users to
  specify the `namespaceConfig` of the generated `App` manifest. Read [App CR's target namespace configuration](https://docs.giantswarm.io/app-platform/namespace-configuration/) for more information.

### Changed

- `get clusters`:
  - The output table header `ID` has been renamed to `NAME`.
- `get nodepools`:
  - The `--cluster-id` flag is now deprecated, replaced with `--cluster-name`.
  - Output column headers have been renamed from `ID` to `NAME` and from `CLUSTER ID` to `CLUSTER NAME`.
- `template cluster`:
  - Deprecated the `--cluster-id` flag.
  - Breaking: the `--name` flag changed purpose to set the cluster's unique identifier.
  - The `--description` flag has been added to set the user-friendly description.
- `template nodepool`:
  - Deprecated the `--cluster-id` flag, added the `--cluster-name` flag as a replacement.
  - Deprecated the `--nodepool-name` flag, add the `--description` flag as a replacement.

## [1.33.0] - 2021-07-19

### Added

- Add support for Spot VMs for Azure Node Pools.

## [1.32.0] - 2021-07-16

### Changed

- Replace AppCatalog CRD with new namespace scoped Catalog CRD.

### Added

- Add templating using CAPA upstream templates for clusters in release version `v20.0.0` on AWS.
- Add templating using CAPA upstream templates for machinepools in release version `v20.0.0` on AWS.
- Add optional `--release` flag to nodepool templating so that the new functionality can be used for CAPA versions.

### Fixed

- Extend `login` error message to mention if OIDC is configured.

## [1.31.0] - 2021-07-08

### Added

- Allow overriding the app CR name in the `template app` command.

### Changed

- Update Dockerfile to use alpine:3.14 as a base image

## [1.30.0] - 2021-06-29

### Changed

- Extend `template app` to only output required fields, the flag `--defaulting-enabled`
can be set to false to disable this.

## [1.29.2] - 2021-06-17

- In the `template cluster` command, the flag `--control-plane-az` is replacing `--master-az`.

## [1.29.1] - 2021-06-16

- Modify the AWS subnet validation for machine deployments.

## [1.29.0] - 2021-06-15

### Added

- Add the AWS subnet annotations into the `template` command.
- Limit the time allowed for the `login` command to call the authentication proxy to one minute.

### Changed

- Updated terminology to use "control plane nodes" instead of "master nodes".

## [1.28.0] - 2021-05-11

### Changed

- Disable unique AZ validation to allow China cluster templating.

### Fixed

- Only set configmap or secret in `template appcatalog` if values are provided.

### Added

- Add `get appcatalogs` and `get apps` commands.

## [1.27.1] - 2021-04-28

### Fixed

- Bug fixed on the internal API URL composition.

## [1.27.0] - 2021-04-27

### Added

- Publish darwin and linux arm64 to krew index.
- Login command now supports internal API.

### Fixed

- Fix templating nested YAML for configmaps and secrets referenced in App and AppCatalog CRs.

## [1.26.0] - 2021-04-13

### Added

- Add clusterresourcesets and clusterresourcesetbindings CRDs to the information about Cluster API CRDs and controllers.

### Removed

- Removed the `--num-availability-zones` flag from the `kubectl-gs template` commands. The `--availability-zones` flag
  should be used to specify a list of availability zones.

### Changed

- Build release binaries using go 1.16. Kubectl-gs is now available for Linux and Darwin ARM64 machines including Apple M1 Macs.
- Upgrade dependency github.com/giantswarm/app to 4.9.0
- Fetch installation information using a new service, instead of relying on the Giant Swarm REST API.

## [1.25.0] - 2021-03-16

### Changed

- Disallow provided cluster IDs from starting with a digit.

## [1.24.0] - 2021-03-10

### Added

- Add support for showing information about Cluster API CRDs and controllers.

### Changed

- Switch to a PKCE authentication flow.

### Fixed

- If the CLI quits with an error, display the error via the default OS error stream.

## [1.23.0] - 2021-02-24

### Changed

- Allow having node pools with the scaling set to `0`.

## [1.22.0] - 2021-02-11

### Changed

- The `MachinePool` CRs now hold a reference to the `Spark` CR in their `spec.template.spec.bootstrap.configRef` field.
- Add missing fields when templating `AzureMachine` and `AzureCluster`, in order to pass CAPZ validation.

## [1.21.0] - 2021-01-29

### Changed

- Make the `login` command validate the current context before considering it good to go.
- Re-enabled the `--pods-cidr` flag in the `template cluster` command.
- Return the Cluster API `Cluster` resource when using the `get clusters` command with `YAML`/`JSON` output.

## [1.20.0] - 2021-01-18

### Added

- Add support for getting nodepools.

### Removed

- Remove the mention of the unexistent 'create cluster' command.

## [1.19.0] - 2021-01-12

### Added

- Add support for node pool autoscaling on Azure.

## [0.18.0] - 2020-12-14

## [0.17.0] - 2020-12-14

### Changed

- Allow for empty `--release` flag in AWS since it is defaulted in the admission controller.
- Allow for empty `--master-az` flag in AWS since it is defaulted in the admission controller.
- Allow for empty `--name` flag in AWS since it is defaulted in the admission controller.

### Removed

- Removed the `--domain` flag since it is managed by admission controller.
- Removed defaulting of the `--provider` flag for `template cluster` and `template nodepool` commands.

## [0.16.0] - 2020-12-09

- In the `template nodepool` command, the flags `--nodex-min` and `--nodex-max` have been renamed to `--nodes-min` and `--nodes-max`.

## [0.15.0] - 2020-12-07

### Added

- Add support for fetching installation information using non-standard Giant Swarm API URLs.

### Removed

- Removed the `--credential` flag, now it is managed by admission controller.

## [0.14.0] - 2020-11-24

### Added

- Add `--cluster-admin` flag to `login` command, which allows full access for Giant Swarm staff.
- Print namespace when using the `get clusters` command with the `--all-namespaces` flag.

### Removed

- Remove client-side validation of the GS `release` when creating a `Cluster`'s template.

## [0.13.0] - 2020-11-20

### Removed

- Removed the `--release` and `--release-branch` version from `kubectl-gs template nodepool` command.

## [0.12.0] - 2020-11-13

### Removed

- Removed the `--region` flag from the `kubectl-gs template` commands. Region gets set automatically according to the installation the cluster is created in.

### Added

- Added the `validate apps` command for validating the values of apps against the `values.schema.json` in their chart, or against a `values.schema.json` locally via a path provided through the command line.

## [0.11.0] - 2020-10-29

### Added

- Add support for using a custom namespace set for a specific Kubernetes context in the Kubeconfig file.
- Add support for using the `--all-namespaces, -A` flag for listing resources in all namespaces.

## [0.10.0] - 2020-10-23

### Removed

- Removed availability zones for `GermanyWestCentral` in `Azure`.
- Removed the `--domain` flag on `Azure`.

## [0.9.0] - 2020-10-16

### Removed

- Remove SSH key parameter for azure in the template command.

## [0.8.0] - 2020-10-14

### Added

- Start publishing a container image of kubectl-gs as giantswarm/kubectl-gs

### Changed

- Normalize organization name when used as a namespace, to match company requirements.
- Allow using inactive release versions for templating clusters. This is especially useful for testing cluster upgrades.

## [0.7.2] - 2020-10-12

### Changed

- Store Azure node pools resources in the organization-specific namespace.
- Display full error output when getting installation info fails or when the OIDC configuration is incorrect, while running the `login` command fails.
- Use proper CAPI conditions to determine Azure Cluster status.

### Fixed

- Use the custom releases branch when fetching release components.

## [0.7.1] - 2020-09-30

### Added

- Add support for using a custom release branch when templating clusters or node pools.

### Changed

- Change the default Azure VM size to `Standard_D4s_v3`

### Fixed

- Store all Azure resources in the organization-specific namespace.
- Use correct K8s API version for Cluster API Machine Pools.

## [0.7.0] - 2020-09-30

### Added

- Add support for templating clusters and node pools on Azure.
- Add support for templating NetworkPools.

## [0.6.1] - 2020-09-14

### Added

- Add the `--version` flag for printing the current version. Run `kgs --version` to check which version you're running.

### Changed

- Disabled templating clusters with legacy or deprecated release versions.
- Allow specifying the `--release` flag for templating clusters and node pools with leading `v`.

## [0.6.0] - 2020-08-11

### Added

- Implemented support for the `get cluster(s) <id>` command.
- Improved error printing formatting.

### Changed

- Running the `template` command without any arguments how displays the command help output.

## [0.5.5] - 2020-07-28

### Fixed

- Make executable work on lightweight linux distributions, such as `alpine`.

## [0.5.4] - 2020-07-24

### Fixed

- Prevent breaking the client's kubeconfig if token renewal fails.

### Added

- Add `--use-alike-instance-types` for node pools.

## [0.5.3] - 2020-07-13

- Add `kubectl gs login` command (#85, #86, #87)

## [0.5.2] - 2020-07-03

No changes

## [0.5.1] - 2020-07-03

- Several changes regarding the use as a kubectl plugin
- Remove non-existing AZ cn-north-1c (#54)
- Allow specifying tenant cluster labels through --label flags (#55)
- Update main README, Installation docs for Krew (#56)

## [0.5.0] 2020-06-10

- Add support for organization credentials

## [0.4.0] 2020-06-09

- Add support for new release info structure

## [0.3.5] 2020-06-04

- Add goreleaser github action
- Add instance distribution (#48)
- Remove default node pool creation (#49)

## [0.3.4] 2020-05-27

- Add support for AWS China https://github.com/giantswarm/kubectl-gs/pull/47
- add AWS availability zone `ap-southeast-1a` https://github.com/giantswarm/kubectl-gs/pull/46

## [0.3.3] 2020-05-21

- Add External SNAT option

## [0.3.2] 2020-05-08

- Allow user to create cluster with cluster ID containing `[a-z0-9]`

## [0.3.1] 2020-05-06

- Fix mixed namespace/cluster namespaces usage in App CR

## [0.3.0] 2020-05-06

- Allow user to specify Cluster ID

## [0.2.0] 2020-03-26

- Added `pods-cidr` flag to generate pods CIDR in Cluster CRs
- Added support for new Release CR

## [0.1.0] 2020-03-26

This release supports rendering for CRs:

- Tenant cluster control plane:
  - `Cluster` (API version `cluster.x-k8s.io/v1alpha2`)
  - `AWSCluster` (API version `infrastructure.giantswarm.io/v1alpha2`)
- Node pool:
  - `MachineDeployment` (API version `cluster.x-k8s.io/v1alpha2`)
  - `AWSMachineDeployment` (API version `infrastructure.giantswarm.io/v1alpha2`)
- `AppCatalog`
- `App`

[Unreleased]: https://github.com/giantswarm/kubectl-gs/compare/v4.7.0...HEAD
[4.7.0]: https://github.com/giantswarm/kubectl-gs/compare/v4.6.0...v4.7.0
[4.6.0]: https://github.com/giantswarm/kubectl-gs/compare/v4.5.0...v4.6.0
[4.5.0]: https://github.com/giantswarm/kubectl-gs/compare/v4.4.0...v4.5.0
[4.4.0]: https://github.com/giantswarm/kubectl-gs/compare/v4.3.1...v4.4.0
[4.3.1]: https://github.com/giantswarm/kubectl-gs/compare/v4.3.0...v4.3.1
[4.3.0]: https://github.com/giantswarm/kubectl-gs/compare/v4.2.0...v4.3.0
[4.2.0]: https://github.com/giantswarm/kubectl-gs/compare/v4.1.0...v4.2.0
[4.1.0]: https://github.com/giantswarm/kubectl-gs/compare/v4.0.0...v4.1.0
[4.0.0]: https://github.com/giantswarm/kubectl-gs/compare/v3.2.0...v4.0.0
[3.2.0]: https://github.com/giantswarm/kubectl-gs/compare/v3.1.0...v3.2.0
[3.1.0]: https://github.com/giantswarm/kubectl-gs/compare/v3.0.0...v3.1.0
[3.0.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.57.0...v3.0.0
[2.57.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.56.0...v2.57.0
[2.56.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.55.0...v2.56.0
[2.55.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.54.0...v2.55.0
[2.54.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.53.0...v2.54.0
[2.53.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.52.3...v2.53.0
[2.52.3]: https://github.com/giantswarm/kubectl-gs/compare/v2.52.2...v2.52.3
[2.52.2]: https://github.com/giantswarm/kubectl-gs/compare/v2.52.1...v2.52.2
[2.52.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.52.0...v2.52.1
[2.52.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.51.0...v2.52.0
[2.51.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.50.1...v2.51.0
[2.50.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.50.0...v2.50.1
[2.50.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.49.1...v2.50.0
[2.49.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.49.0...v2.49.1
[2.49.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.48.1...v2.49.0
[2.48.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.48.0...v2.48.1
[2.48.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.47.1...v2.48.0
[2.47.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.47.0...v2.47.1
[2.47.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.46.0...v2.47.0
[2.46.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.45.4...v2.46.0
[2.45.4]: https://github.com/giantswarm/kubectl-gs/compare/v2.45.3...v2.45.4
[2.45.3]: https://github.com/giantswarm/kubectl-gs/compare/v2.45.3...v2.45.3
[2.45.3]: https://github.com/giantswarm/kubectl-gs/compare/v2.45.2...v2.45.3
[2.45.2]: https://github.com/giantswarm/kubectl-gs/compare/v2.45.1...v2.45.2
[2.45.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.45.0...v2.45.1
[2.45.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.44.0...v2.45.0
[2.44.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.43.0...v2.44.0
[2.43.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.42.0...v2.43.0
[2.42.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.41.1...v2.42.0
[2.41.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.41.0...v2.41.1
[2.41.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.40.0...v2.41.0
[2.40.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.39.0...v2.40.0
[2.39.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.38.0...v2.39.0
[2.38.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.37.0...v2.38.0
[2.37.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.36.1...v2.37.0
[2.36.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.36.0...v2.36.1
[2.36.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.35.0...v2.36.0
[2.35.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.34.1...v2.35.0
[2.34.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.34.0...v2.34.1
[2.34.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.33.0...v2.34.0
[2.33.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.32.0...v2.33.0
[2.32.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.31.2...v2.32.0
[2.31.2]: https://github.com/giantswarm/kubectl-gs/compare/v2.31.1...v2.31.2
[2.31.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.31.0...v2.31.1
[2.31.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.30.0...v2.31.0
[2.30.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.29.5...v2.30.0
[2.29.5]: https://github.com/giantswarm/kubectl-gs/compare/v2.29.4...v2.29.5
[2.29.4]: https://github.com/giantswarm/kubectl-gs/compare/v2.29.3...v2.29.4
[2.29.3]: https://github.com/giantswarm/kubectl-gs/compare/v2.29.2...v2.29.3
[2.29.2]: https://github.com/giantswarm/kubectl-gs/compare/v2.29.1...v2.29.2
[2.29.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.29.0...v2.29.1
[2.29.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.28.2...v2.29.0
[2.28.2]: https://github.com/giantswarm/kubectl-gs/compare/v2.28.1...v2.28.2
[2.28.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.28.0...v2.28.1
[2.28.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.27.0...v2.28.0
[2.27.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.26.1...v2.27.0
[2.26.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.26.0...v2.26.1
[2.26.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.25.0...v2.26.0
[2.25.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.24.2...v2.25.0
[2.24.2]: https://github.com/giantswarm/kubectl-gs/compare/v2.24.1...v2.24.2
[2.24.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.24.0...v2.24.1
[2.24.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.23.2...v2.24.0
[2.23.2]: https://github.com/giantswarm/kubectl-gs/compare/v2.23.1...v2.23.2
[2.23.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.23.0...v2.23.1
[2.23.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.22.0...v2.23.0
[2.22.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.21.0...v2.22.0
[2.21.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.20.0...v2.21.0
[2.20.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.19.3...v2.20.0
[2.19.3]: https://github.com/giantswarm/kubectl-gs/compare/v2.19.2...v2.19.3
[2.19.2]: https://github.com/giantswarm/kubectl-gs/compare/v2.19.1...v2.19.2
[2.19.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.19.0...v2.19.1
[2.19.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.18.0...v2.19.0
[2.18.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.17.0...v2.18.0
[2.17.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.16.0...v2.17.0
[2.16.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.15.0...v2.16.0
[2.15.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.14.0...v2.15.0
[2.14.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.13.2...v2.14.0
[2.13.2]: https://github.com/giantswarm/kubectl-gs/compare/v2.13.1...v2.13.2
[2.13.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.13.0...v2.13.1
[2.13.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.12.1...v2.13.0
[2.12.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.12.0...v2.12.1
[2.12.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.11.2...v2.12.0
[2.11.2]: https://github.com/giantswarm/kubectl-gs/compare/v2.11.1...v2.11.2
[2.11.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.11.0...v2.11.1
[2.11.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.10.0...v2.11.0
[2.10.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.9.1...v2.10.0
[2.9.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.9.0...v2.9.1
[2.9.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.8.1...v2.9.0
[2.8.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.8.0...v2.8.1
[2.8.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.7.11...v2.8.0
[2.7.11]: https://github.com/giantswarm/kubectl-gs/compare/v2.7.10...v2.7.11
[2.7.10]: https://github.com/giantswarm/kubectl-gs/compare/v2.7.1...v2.7.10
[2.7.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.7.0...v2.7.1
[2.7.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.6.0...v2.7.0
[2.6.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.5.0...v2.6.0
[2.5.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.4.0...v2.5.0
[2.4.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.3.1...v2.4.0
[2.3.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.3.0...v2.3.1
[2.3.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.2.0...v2.3.0
[2.2.0]: https://github.com/giantswarm/kubectl-gs/compare/v2.1.1...v2.2.0
[2.1.1]: https://github.com/giantswarm/kubectl-gs/compare/v2.1.0...v2.1.1
[2.1.0]: https://github.com/giantswarm/giantswarm/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/giantswarm/giantswarm/compare/v1.60.0...v2.0.0
[1.60.0]: https://github.com/giantswarm/giantswarm/compare/v1.59.0...v1.60.0
[1.59.0]: https://github.com/giantswarm/giantswarm/compare/v1.58.2...v1.59.0
[1.58.2]: https://github.com/giantswarm/kubectl-gs/compare/v1.58.1...v1.58.2
[1.58.1]: https://github.com/giantswarm/kubectl-gs/compare/v1.58.0...v1.58.1
[1.58.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.57.0...v1.58.0
[1.57.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.56.0...v1.57.0
[1.56.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.55.0...v1.56.0
[1.55.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.54.0...v1.55.0
[1.54.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.53.0...v1.54.0
[1.53.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.52.0...v1.53.0
[1.52.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.51.0...v1.52.0
[1.51.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.50.1...v1.51.0
[1.50.1]: https://github.com/giantswarm/kubectl-gs/compare/v1.50.0...v1.50.1
[1.50.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.49.0...v1.50.0
[1.49.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.48.1...v1.49.0
[1.48.1]: https://github.com/giantswarm/kubectl-gs/compare/v1.48.0...v1.48.1
[1.48.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.47.0...v1.48.0
[1.47.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.46.0...v1.47.0
[1.46.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.45.0...v1.46.0
[1.45.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.44.0...v1.45.0
[1.44.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.43.1...v1.44.0
[1.43.1]: https://github.com/giantswarm/kubectl-gs/compare/v1.43.0...v1.43.1
[1.43.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.42.1...v1.43.0
[1.42.1]: https://github.com/giantswarm/kubectl-gs/compare/v1.42.0...v1.42.1
[1.42.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.41.1...v1.42.0
[1.41.1]: https://github.com/giantswarm/kubectl-gs/compare/v1.41.0...v1.41.1
[1.41.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.40.0...v1.41.0
[1.40.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.39.2...v1.40.0
[1.39.2]: https://github.com/giantswarm/kubectl-gs/compare/v1.39.1...v1.39.2
[1.39.1]: https://github.com/giantswarm/kubectl-gs/compare/v1.39.0...v1.39.1
[1.39.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.38.0...v1.39.0
[1.38.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.37.0...v1.38.0
[1.37.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.36.0...v1.37.0
[1.36.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.35.1...v1.36.0
[1.35.1]: https://github.com/giantswarm/kubectl-gs/compare/v1.35.0...v1.35.1
[1.35.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.34.0...v1.35.0
[1.34.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.33.0...v1.34.0
[1.33.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.32.0...v1.33.0
[1.32.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.31.0...v1.32.0
[1.31.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.30.0...v1.31.0
[1.30.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.29.2...v1.30.0
[1.29.2]: https://github.com/giantswarm/kubectl-gs/compare/v1.29.1...v1.29.2
[1.29.1]: https://github.com/giantswarm/kubectl-gs/compare/v1.29.0...v1.29.1
[1.29.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.28.0...v1.29.0
[1.28.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.27.1...v1.28.0
[1.27.1]: https://github.com/giantswarm/kubectl-gs/compare/v1.27.0...v1.27.1
[1.27.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.26.0...v1.27.0
[1.26.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.25.0...v1.26.0
[1.25.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.24.0...v1.25.0
[1.24.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.23.0...v1.24.0
[1.23.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.22.0...v1.23.0
[1.22.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.21.0...v1.22.0
[1.21.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.20.0...v1.21.0
[1.20.0]: https://github.com/giantswarm/kubectl-gs/compare/v1.19.0...v1.20.0
[1.19.0]: https://github.com/giantswarm/kubectl-gs/compare/v0.18.0...v1.19.0
[0.18.0]: https://github.com/giantswarm/kubectl-gs/compare/v0.17.0...v0.18.0
[0.17.0]: https://github.com/giantswarm/kubectl-gs/compare/v0.16.0...v0.17.0
[0.16.0]: https://github.com/giantswarm/kubectl-gs/compare/v0.15.0...v0.16.0
[0.15.0]: https://github.com/giantswarm/kubectl-gs/compare/v0.14.0...v0.15.0
[0.14.0]: https://github.com/giantswarm/kubectl-gs/compare/v0.13.0...v0.14.0
[0.13.0]: https://github.com/giantswarm/kubectl-gs/compare/v0.12.0...v0.13.0
[0.12.0]: https://github.com/giantswarm/kubectl-gs/compare/v0.11.0...v0.12.0
[0.11.0]: https://github.com/giantswarm/kubectl-gs/compare/v0.10.0...v0.11.0
[0.10.0]: https://github.com/giantswarm/kubectl-gs/compare/v0.9.0...v0.10.0
[0.9.0]: https://github.com/giantswarm/kubectl-gs/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/giantswarm/kubectl-gs/compare/v0.7.2...v0.8.0
[0.7.2]: https://github.com/giantswarm/kubectl-gs/compare/v0.7.1...v0.7.2
[0.7.1]: https://github.com/giantswarm/kubectl-gs/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/giantswarm/kubectl-gs/compare/v0.6.1...v0.7.0
[0.6.1]: https://github.com/giantswarm/kubectl-gs/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/giantswarm/kubectl-gs/compare/v0.5.5...v0.6.0
[0.5.5]: https://github.com/giantswarm/kubectl-gs/compare/v0.5.4...v0.5.5
[0.5.4]: https://github.com/giantswarm/kubectl-gs/compare/v0.5.3...v0.5.4
[0.5.3]: https://github.com/giantswarm/kubectl-gs/compare/v0.5.2...v0.5.3
[0.5.2]: https://github.com/giantswarm/kubectl-gs/compare/v0.5.1...v0.5.2
[0.5.1]: https://github.com/giantswarm/kubectl-gs/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/giantswarm/kubectl-gs/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/giantswarm/kubectl-gs/compare/v0.3.5...v0.4.0
[0.3.5]: https://github.com/giantswarm/kubectl-gs/compare/v0.3.4...v0.3.5
[0.3.4]: https://github.com/giantswarm/kubectl-gs/compare/v0.3.3...v0.3.4
[0.3.3]: https://github.com/giantswarm/kubectl-gs/compare/v0.3.2...v0.3.3
[0.3.2]: https://github.com/giantswarm/kubectl-gs/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/giantswarm/kubectl-gs/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/giantswarm/kubectl-gs/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/giantswarm/kubectl-gs/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/giantswarm/kubectl-gs/releases/tag/v0.1.0
