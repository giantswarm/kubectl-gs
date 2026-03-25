# Open questions and unclear areas for this specm as of 2026-03-05

### Critical / High Impact

- **`--auto-upgrade` without `--version`** — How is the semver range constructed if we first have to resolve "latest" then derive a filter from it? Race condition possible between resolution and creation.

  - We accept the risk that "latest" resolution may be inaccurate at the moment of HelmRelease creation, but we should document this clearly.

- **`--values-from` is underspecified** — No syntax documented, no example manifest, unclear namespace, unclear interaction with `--values-file`, unclear if repeatable.

  - Must be in same namespace
  - HelmRelease can work with multiple value sources. `--values-file` always ends up as .spec.values. `--values-from` can be used multiple times and each entry is added to .spec.valuesFrom[].

- **`--dry-run` and server access** — Spec says server-side validation is performed, but a key use case (GitOps manifest generation) may not have cluster access. Should there be a client-only dry-run option?

  - No, we require server access.

### Medium Impact

- **`--management-cluster` cluster name derivation** — "Determined automatically from context" is vague. What's the actual logic?

  - Not sure yet. TODO

- **Re-run in non-interactive mode** — No `--force`/`--yes` flag for CI pipelines. How to skip the diff confirmation?

  - We leave this open for now, as the command is intended for interactive use. If the need arises, we can add a `--yes` flag to skip confirmation in non-interactive environments.

- **Config validation when registry is unreachable** — Fail or skip? Also, validation only works for charts with the GS-specific annotation — worth stating explicitly.

  - If the registry is unreachable, we fail with a clear error message.

- **Flux not installed / `--dry-run` without cluster** — What errors are shown? Can Flux version be specified manually for offline manifest generation?

  - We'll determine CRD versions directly instead.

- **Secret naming convention** — `<cluster>-<chart>-registry` pattern not explicitly documented. Does `--name` override affect it?

  - Yes, the `--name` flag will override the `<chart>` part of the secret name.

### Lower Impact / Edge Cases

- **Resource name length** — `<clustername>-<chartname>` could exceed K8s limits.
  - We should validate the combined length and return an error if it exceeds the maximum characters.
- **`--chart-name` with special characters** — What if it contains `/`?
  - Since we look up the chart anyway, we will found out it it's a valid name. Slashes are not allowed though and should be prevented early.
- **`--registry-provider` alone is insufficient for client-side auth** — Is this clearly communicated?
  - In this case we should require a docker credential available to the user.
- **`interval: 10m` hardcoded** — Should it be configurable?
  - Adding flag `--interval`
- **`releaseName` always equals `--chart-name`** — Can it be overridden?
  - No
- **Unresolved TODO** — MC context verification is still undefined.
- **Label set** — Only `giantswarm.io/cluster` specified; are standard k8s labels needed?
