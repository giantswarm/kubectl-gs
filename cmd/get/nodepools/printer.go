package nodepools

import (
	"fmt"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"

	"github.com/giantswarm/kubectl-gs/v2/cmd/get/nodepools/provider"
	"github.com/giantswarm/kubectl-gs/v2/internal/feature"
	"github.com/giantswarm/kubectl-gs/v2/internal/key"
	"github.com/giantswarm/kubectl-gs/v2/pkg/data/domain/nodepool"
	"github.com/giantswarm/kubectl-gs/v2/pkg/output"
)

type PrintOptions struct {
	Name string
}

func (r *runner) printOutput(npResource nodepool.Resource) error {
	var err error
	var printer printers.ResourcePrinter
	var resource runtime.Object

	switch {
	case output.IsOutputDefault(r.flag.print.OutputFormat):
		switch r.provider {
		case key.ProviderAWS:
			capabilities := feature.New(feature.ProviderAWS)
			resource = provider.GetAWSTable(npResource, capabilities)
		case key.ProviderCAPA:
			capabilities := feature.New(feature.ProviderCAPA)
			resource = provider.GetCAPATable(npResource, capabilities)
		case key.ProviderCAPZ:
			capabilities := feature.New(feature.ProviderCAPZ)
			resource = provider.GetCAPITable(npResource, capabilities)
		case key.ProviderAzure:
			capabilities := feature.New(feature.ProviderAzure)
			resource = provider.GetAzureTable(npResource, capabilities)
		case key.ProviderCloudDirector:
			capabilities := feature.New(feature.ProviderCloudDirector)
			resource = provider.GetCAPITable(npResource, capabilities)
		case key.ProviderVSphere:
			capabilities := feature.New(feature.ProviderVSphere)
			resource = provider.GetCAPITable(npResource, capabilities)
		}

		printOptions := printers.PrintOptions{
			WithNamespace: r.flag.AllNamespaces,
		}
		printer = printers.NewTablePrinter(printOptions)

	case output.IsOutputName(r.flag.print.OutputFormat):
		resource = npResource.Object()
		err = output.PrintResourceNames(r.stdout, resource)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil

	default:
		resource = npResource.Object()
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
	fmt.Fprintf(r.stdout, "  kubectl gs template nodepool --help\n")
}
