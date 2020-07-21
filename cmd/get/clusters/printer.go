package clusters

import (
	"fmt"
	"io"
	"strings"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/meta"
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

	switch {
	case output.IsOutputDefault(r.flag.print.OutputFormat):
		switch r.provider {
		case key.ProviderAWS:
			resource = provider.GetAWSTable(resource)
		}

		printOptions := printers.PrintOptions{}
		printer = printers.NewTablePrinter(printOptions)

	case output.IsOutputName(r.flag.print.OutputFormat):
		err = r.printResourceName(resource)
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

func (r *runner) printNoResourcesOutput() error {
	_, err := io.WriteString(r.stdout, "No clusters found.\n")
	if err != nil {
		return microerror.Mask(err)
	}
	_, err = io.WriteString(r.stdout, "To create a cluster, please check\n\n")
	if err != nil {
		return microerror.Mask(err)
	}
	_, err = io.WriteString(r.stdout, "  kgs create cluster --help\n")
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) printResourceName(resource runtime.Object) error {
	list, err := extractResourcesOutOfList(resource)
	if err != nil {
		return microerror.Mask(err)
	}

	printer := &printers.NamePrinter{}
	var buf strings.Builder
	for _, item := range list {
		err = printer.PrintObj(item, &buf)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	_, err = fmt.Fprint(r.stdout, buf.String())
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func extractResourcesOutOfList(resource runtime.Object) ([]runtime.Object, error) {
	if !meta.IsListType(resource) {
		return []runtime.Object{resource}, nil
	}

	list, err := meta.ExtractList(resource)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var resources []runtime.Object
	for _, item := range list {
		list, err = extractResourcesOutOfList(item)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		resources = append(resources, list...)
	}

	return resources, nil
}
