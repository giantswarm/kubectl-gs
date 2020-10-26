package apps

import (
	"fmt"

	"github.com/giantswarm/kubectl-gs/pkg/data/domain/app"
	"github.com/giantswarm/kubectl-gs/pkg/pluralize"
)

// PrintReport prints app validation
// results in a human readable report format.
func PrintReport(results app.ValidationResults) error {

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
			count += len(r.ValidationErrors)
		}

		namespaceErrorCount[namespace] = count
	}

	resultCount := len(results)
	namespaceCount := len(namespaceSet)

	pluralizedApps := pluralize.Pluralize("app", resultCount)
	pluralizedNamespaces := pluralize.Pluralize("namespace", namespaceCount)

	fmt.Print("\n")
	fmt.Printf("Validated %d %s across %d %s", resultCount, pluralizedApps, namespaceCount, pluralizedNamespaces)
	fmt.Print("\n")
	fmt.Print("\n")

	for namespace, validationResults := range namespaceSet {

		pluralizedApps := pluralize.Pluralize("app", len(validationResults))
		pluralizedErrors := pluralize.Pluralize("error", namespaceErrorCount[namespace])

		fmt.Printf("%s [%d %s, %d %s]", namespace, len(validationResults), pluralizedApps, namespaceErrorCount[namespace], pluralizedErrors)
		fmt.Print("\n")

		if namespaceErrorCount[namespace] == 0 {
			continue
		}

		for _, r := range validationResults {
			fmt.Printf("  %s:", r.App.Name)
			fmt.Print("\n")
			for _, e := range r.ValidationErrors {
				fmt.Printf("    %s", e.Field())
				fmt.Printf(" - %v", e.Description())
				fmt.Printf(" - %s", e.Value())
				fmt.Print("\n")
			}
			fmt.Print("\n")
		}

		fmt.Print("\n")
	}

	return nil
}
