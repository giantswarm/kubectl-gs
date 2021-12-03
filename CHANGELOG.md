# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project's packages adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/giantswarm/kubectl-gs/compare/v1.54.0...HEAD
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
