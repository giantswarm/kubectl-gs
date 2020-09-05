package apps

import (
	"fmt"

	"github.com/giantswarm/kubectl-gs/pkg/data/domain/app"
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

	fmt.Print("\n")
	fmt.Printf("Validated %d apps across %d namespaces", resultCount, namespaceCount)
	fmt.Print("\n")
	fmt.Print("\n")

	for namespace, validationResults := range namespaceSet {
		fmt.Printf("%s [%d apps, %d errors]", namespace, len(validationResults), namespaceErrorCount[namespace])
		fmt.Print("\n")

		if namespaceErrorCount[namespace] == 0 {
			continue
		}

		for _, r := range validationResults {
			fmt.Printf("  %s:", r.App.Name)
			fmt.Print("\n")
			for _, e := range r.ValidationErrors {
				fmt.Printf("    %s", e.Message)
				fmt.Print("\n")
			}
			fmt.Print("\n")
		}

		fmt.Print("\n")
	}

	return nil
}
