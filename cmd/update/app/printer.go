package app

import (
	"fmt"
)

func (r *runner) printNoMatchOutput(version string) {
	fmt.Fprintf(r.stdout, "No AppCatalogEntry CR found for the given version: '%s'\n", version)
	fmt.Fprintf(r.stdout, "Please make sure version you are requesting is available in the respective catalog\n")
}
