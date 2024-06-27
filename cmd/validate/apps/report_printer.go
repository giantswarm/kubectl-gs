package apps

import (
	"fmt"
	"io"

	"github.com/giantswarm/kubectl-gs/v4/pkg/app"
	"github.com/giantswarm/kubectl-gs/v4/pkg/pluralize"
)

// PrintReport prints app validation
// results in a human readable report format.
func PrintReport(results app.ValidationResults, stdout io.Writer) error {

	namespaceSet := make(map[string][]*app.ValidationResult)

	// Group results by namespace.
	for _, r := range results {
		namespaceSet[r.App.Namespace] = append(namespaceSet[r.App.Namespace], r)
	}

	namespaceErrorCount := make(map[string]int)
	// Errors per namespace.
	for namespace, validationResults := range namespaceSet {
		count := 0

		for _, r := range validationResults {
			// Add any validation errors to the error count.
			count += len(r.ValidationErrors)

			// Add any execution errors to the error count.
			if r.Err != nil {
				count++
			}
		}

		namespaceErrorCount[namespace] = count
	}

	resultCount := len(results)
	namespaceCount := len(namespaceSet)

	pluralizedApps := pluralize.Pluralize("app", resultCount)
	pluralizedNamespaces := pluralize.Pluralize("namespace", namespaceCount)

	fmt.Fprintf(stdout, "\n")
	fmt.Fprintf(stdout, "Validated %d %s across %d %s", resultCount, pluralizedApps, namespaceCount, pluralizedNamespaces)
	fmt.Fprintf(stdout, "\n")
	fmt.Fprintf(stdout, "\n")

	for namespace, validationResults := range namespaceSet {

		pluralizedApps := pluralize.Pluralize("app", len(validationResults))
		pluralizedErrors := pluralize.Pluralize("error", namespaceErrorCount[namespace])

		fmt.Fprintf(stdout, "%s [%d %s, %d %s]", namespace, len(validationResults), pluralizedApps, namespaceErrorCount[namespace], pluralizedErrors)
		fmt.Fprintf(stdout, "\n")

		if namespaceErrorCount[namespace] == 0 {
			continue
		}

		for _, r := range validationResults {
			fmt.Fprintf(stdout, "  %s:", r.App.Name)
			fmt.Fprintf(stdout, "\n")
			for _, e := range r.ValidationErrors {
				fmt.Fprintf(stdout, "    %s", e.Field())
				fmt.Fprintf(stdout, " - %v", e.Description())
				fmt.Fprintf(stdout, " - %s", e.Value())
				fmt.Fprintf(stdout, "\n")
			}

			if r.Err != nil {
				fmt.Fprintf(stdout, "    %s", r.Err.Error())
				fmt.Fprintf(stdout, "\n")
			}

			fmt.Fprintf(stdout, "\n")
		}

		fmt.Fprintf(stdout, "\n")
	}

	return nil
}
