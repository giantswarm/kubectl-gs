---
name: cli-user-experience
description: How to ensure good user experience in a CLI (command line interface) program.
---

## Commands in context

- The hierarchical nature of commands and subcommands should ideally create meaningful phrases, e. g. `deploy chart`.
- Commands use aliases for common short forms (e.g., cluster for clusters)
- Parent commands that have subcommands with aliases must use `usagetemplate.Apply(cmd)` from `pkg/usagetemplate` so that aliases are visible in the `--help` output's "Available Commands" section

## Heuristics for good commands

- Command has a short (single line) description in cobra.Command.Short
- Command has a multi line description in cobra.Command.Long
- Command has examples in cobra.Command.Example
- Examples are broken into multiple lines with subsequent lines indented consistently using spaces, not tabs
- Examples use the long form of all flags
- Examples show realistic values, not placeholders like <foo>
- Examples cover both basic and advanced usage (simple invocation + flag combinations)
- Long description is meaningfully different from Short — not just a repetition
- Long usage description provides docs URL starting with https://docs.giantswarm.io

## Heuristics for good flags

- Flag has a long form name (single-letter name is optional, long name is mandatory)
- Uses dash as the only separator
- Words in flag name are separated. Example: `--values-from`
- Usage description starts with uppercase letter
- If the flag has a default value, the usage should not end in a phrase with brackets, as this would be appended to the default value info in brackets by cobra. Negative example: `Default cache directory (optional) (default "/Users/marian/.kube/cache")`
- Required flags are clearly indicated in their usage description. E.g., suffix with `(required)`
- Mutually exclusive flags have validation with a clear error message naming both flags
- Flag usage description does not end with a period (consistent with kubectl conventions)
- Complex flags include format hints in their description (e.g., "must be valid semver (e.g. 1.2.3)")
- Validation errors name the offending flag with -- prefix (e.g., "--%s must not be empty")

## Output heuristics

- Status/progress messages go to stderr, never stdout — so piping/redirecting output works correctly
- Structured output (JSON/YAML) goes to stdout only, with no interleaved status messages
- "No results" messages include a helpful suggestion for what to do next (e.g., "To create a cluster, see kubectl gs template cluster --help")
- Commands that support --output support all standard formats: default (table), JSON, YAML, name
- Table column descriptions are documented in the Long help text for get commands

## Errors

- Exit code is 1 on error, 0 on success — no other codes unless documented
- Validation errors show the invalid value the user provided (e.g., got %q)
- Commands don't dump full help text on validation errors — just show the error and suggest --help
- Debug mode (--debug) shows stack traces; normal mode shows user-friendly messages only
- SilenceUsage: true is set to prevent cobra from dumping usage on every error

## Confirmation / interactive heuristics

- Destructive or mutating operations prompt for confirmation unless --force or equivalent is provided
- Confirmation prompts default to "No" ([y/N]) — conservative by default
- Interactive prompts detect TTY and fail gracefully in non-interactive contexts instead of hanging
- Sensitive input (passwords, tokens) uses term.ReadPassword() — no echo
