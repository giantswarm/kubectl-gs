package orgs

import (
	"fmt"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"

	"github.com/giantswarm/kubectl-gs/pkg/data/domain/organization"
)

type PrintOptions struct {
	Name string
}

func (r *runner) printOutput(orgResource organization.Resource) error {
	var err error
	var printer printers.ResourcePrinter
	var resource runtime.Object = orgResource.Object()

	printOptions := printers.PrintOptions{
		NoHeaders:        false,
		WithNamespace:    false,
		WithKind:         true,
		Wide:             true,
		ShowLabels:       false,
		AllowMissingKeys: true,
	}
	printer = printers.NewTablePrinter(printOptions)

	err = printer.PrintObj(resource, r.stdout)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) printNoResourcesOutput() {
	fmt.Fprintf(r.stdout, "No organizations found.\n")
	fmt.Fprintf(r.stdout, "No resources of type organizations.security.giantswarm.io available. To create one, please check\n\n")
	fmt.Fprintf(r.stdout, "  https://docs.giantswarm.io/ui-api/management-api/crd/organizations.security.giantswarm.io/\n")
}
