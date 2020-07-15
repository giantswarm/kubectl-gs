package clusters

import (
	"io"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"

	"github.com/giantswarm/kubectl-gs/cmd/get/clusters/provider"
	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/output"
)

func (r *runner) printOutput(resource runtime.Object) error {
	var (
		err     error
		printer printers.ResourcePrinter
	)

	if output.IsOutputDefault(r.flag.print.OutputFormat) {
		switch r.provider {
		case key.ProviderAWS:
			resource = provider.GetAWSTable(resource)
		case key.ProviderAzure:
			resource = provider.GetAzureTable(resource)
		case key.ProviderKVM:
			resource = provider.GetKVMTable(resource)
		}

		printOptions := printers.PrintOptions{}
		printer = printers.NewTablePrinter(printOptions)
	} else {
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

func (r *runner) printNoResourcesOutput() error {
	_, err := io.WriteString(r.stdout, "No clusters found.\n")
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
