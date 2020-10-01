# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project's packages adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- Display full error output when getting installation info using the `login` command fails.

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

## [0.5.2] - 2020-07-03

## [0.5.1] - 2020-07-03


## [0.5.0] 2020-06-10


## [0.4.0] 2020-06-09


## [0.3.5] 2020-06-04


## [0.3.4] 2020-05-27


## [0.3.3] 2020-05-21


## [0.3.2] 2020-05-08


## [0.3.1] 2020-05-06


## [0.3.0] 2020-05-06


## [0.2.0] 2020-03-26


## [0.1.0] 2020-03-26


## [0.2.0] 2020-04-23


[Unreleased]: https://github.com/giantswarm/kubectl-gs/compare/v0.7.1...HEAD
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
