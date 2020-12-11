package nodepools

import (
	"fmt"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"

	"github.com/giantswarm/kubectl-gs/cmd/get/nodepools/provider"
	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/output"
)

func (r *runner) printOutput(resource runtime.Object) error {
	var (
		err     error
		printer printers.ResourcePrinter
	)

	switch {
	case output.IsOutputDefault(r.flag.print.OutputFormat):
		switch r.provider {
		case key.ProviderAWS:
			resource = provider.GetAWSTable(resource)
		case key.ProviderAzure:
			resource = provider.GetAzureTable(resource)
		}

		printOptions := printers.PrintOptions{
			WithNamespace: r.flag.AllNamespaces,
		}
		printer = printers.NewTablePrinter(printOptions)

	case output.IsOutputName(r.flag.print.OutputFormat):
		err = output.PrintResourceNames(r.stdout, resource)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil

	default:
		printer, err = r.flag.print.ToPrinter()
		if err != nil {
			return microerror.Mask(err)
		}
	}

	err = printer.PrintObj(resource, r.stdout)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) printNoResourcesOutput() {
	fmt.Fprintf(r.stdout, "No node pools found.\n")
	fmt.Fprintf(r.stdout, "To create a node pool, please check\n\n")
	fmt.Fprintf(r.stdout, "  kgs template nodepool --help\n")
}
