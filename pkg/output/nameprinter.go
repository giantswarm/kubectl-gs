package output

import (
	"io"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
)

// PrintResourceNames prints the name of a resource, or if it's
// a list, it will print the names of the nested resources.
//
// It can print deeply nested list objects names.
func PrintResourceNames(out io.Writer, resource runtime.Object) error {
	list, err := extractResourcesOutOfList(resource)
	if err != nil {
		return microerror.Mask(err)
	}

	printer := &printers.NamePrinter{}
	for _, item := range list {
		err = printer.PrintObj(item, out)
		if err != nil {
			return microerror.Mask(err)
		}
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
