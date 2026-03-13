# Review: PR #1890 — Add registry interaction to 'deploy chart' command

**PR**: https://github.com/giantswarm/kubectl-gs/pull/1890
**Author**: @marians
**Status**: Draft
**Branch**: `add-oci-registry-client` → `main`

## Overview

Adds OCI registry client capabilities to `kubectl gs deploy chart`, enabling automatic
chart version resolution when `--version` is omitted. Introduces `internal/ociregistry/`
package with registry authentication, tag listing, manifest annotation reading, and
semver-based version selection.

**Scope**: 9 files changed, +731/-27 lines.

## Positive Aspects

1. **Clean interface design** — `Client` interface with two methods (`ListTags`, `GetManifestAnnotations`) is easy to mock and test.
2. **Solid test coverage** — Mock registry covers anonymous, authenticated, and token exchange flows. Version tests cover edge cases.
3. **Good credential resolution chain** — Explicit flags → Docker config → anonymous matches user expectations.
4. **ACR compatibility** — GET-based token exchange for Azure Container Registry anonymous access is pragmatic and well-documented.

## Issues

### High Priority

1. **`parseWWWAuthenticate` is fragile with commas inside quoted values** (`client.go:243-260`)
   Splits on `,` before extracting key-value pairs. Commas inside quoted values (e.g., `scope="repository:foo:pull,push"`) would break parsing. Consider a proper quoted-string parser.

2. **Missing tag input validation in `GetManifestAnnotations`** (`client.go:102`)
   The `tag` parameter is interpolated directly into a URL. Path traversal characters could construct unintended URLs. Validate or URL-encode the tag.

3. **No HTTP timeout** (`runner.go:56-77`)
   Uses `http.DefaultClient` with no timeout. Slow/unresponsive registries will hang the command indefinitely. Set a default timeout (e.g., 30s).

### Medium Priority

4. **Duplicated Accept header** (`client.go:140-144` and `158-162`)
   Extract to a constant.

5. **`LatestSemverTag` returns `Original()` which may include `v` prefix** (`version.go:34`)
   Verify the `v` prefix is acceptable downstream in `deploychart.OCIRepositoryOptions.Version`.

6. **Removed validation may break auto-upgrade semantics** (`flag.go:91-93`)
   Without `--version`, auto-upgrade `patch`/`minor` constraints are ambiguous (based on latest-at-deploy-time). Document this behavior change or preserve validation for `patch`/`minor`.

7. **Generic error from `LatestSemverTag`** (`version.go:28`)
   Wrap with context in `runner.go` to include the registry/repo URL.

### Low Priority

8. **`credStore` error silently swallowed** (`client.go:64-67`)
   Malformed Docker config gives no indication of failure. Consider logging a warning.

9. **Redundant test cleanup** (`client_test.go:260-262`)
   Both `t.Cleanup(srv.Close)` and `defer srv.Close()` are used. Only one is needed.

## Recommendation

Address items 1-3 and 6 before merging. The rest can be follow-ups.
