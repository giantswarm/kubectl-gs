package appcatalogs

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"

	"github.com/giantswarm/kubectl-gs/pkg/data/domain/appcatalog"
	"github.com/giantswarm/kubectl-gs/pkg/output"
	"github.com/giantswarm/microerror"
)

func (r *runner) printOutput(appCatalogResource appcatalog.Resource) error {
	var (
		err      error
		printer  printers.ResourcePrinter
		resource runtime.Object
	)

	switch {
	case output.IsOutputDefault(r.flag.print.OutputFormat):
		resource = appCatalogResource.Object()
		printOptions := printers.PrintOptions{
			WithNamespace: r.flag.AllNamespaces,
		}
		printer = printers.NewTablePrinter(printOptions)

	case output.IsOutputName(r.flag.print.OutputFormat):
		resource = appCatalogResource.Object()
		err = output.PrintResourceNames(r.stdout, resource)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil

	default:
		resource = appCatalogResource.Object()
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
	fmt.Fprintf(r.stdout, "No appcatalogs found.\n")
	fmt.Fprintf(r.stdout, "To create an appcatalog, please check\n\n")
	fmt.Fprintf(r.stdout, "  kubectl gs template appcatalog --help\n")
}
