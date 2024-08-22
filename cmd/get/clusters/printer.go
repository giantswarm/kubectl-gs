package clusters

import (
	"fmt"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"

	"github.com/giantswarm/kubectl-gs/v5/cmd/get/clusters/provider"
	"github.com/giantswarm/kubectl-gs/v5/internal/key"
	"github.com/giantswarm/kubectl-gs/v5/pkg/data/domain/cluster"
	"github.com/giantswarm/kubectl-gs/v5/pkg/output"
)

func (r *runner) printOutput(clusterResource cluster.Resource) error {
	var (
		err      error
		printer  printers.ResourcePrinter
		resource runtime.Object
	)

	switch {
	case output.IsOutputDefault(r.flag.print.OutputFormat):
		switch r.provider {
		case key.ProviderAWS:
			resource = provider.GetAWSTable(clusterResource)
		case key.ProviderAzure:
			resource = provider.GetAzureTable(clusterResource)
		case key.ProviderOpenStack:
			resource = provider.GetOpenStackTable(clusterResource)
		default:
			resource = provider.GetCommonClusterTable(clusterResource)
		}

		printOptions := printers.PrintOptions{
			WithNamespace: r.flag.AllNamespaces,
		}
		printer = printers.NewTablePrinter(printOptions)

	case output.IsOutputName(r.flag.print.OutputFormat):
		resource = clusterResource.Object()
		err = output.PrintResourceNames(r.stdout, resource)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil

	default:
		resource = clusterResource.Object()
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
	fmt.Fprintf(r.stdout, "No clusters found.\n")
	fmt.Fprintf(r.stdout, "To create a cluster, please check\n\n")
	fmt.Fprintf(r.stdout, "  kubectl gs template cluster --help\n")
}
