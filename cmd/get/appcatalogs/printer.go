package appcatalogs

import (
	"fmt"

	applicationv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"

	"github.com/giantswarm/kubectl-gs/pkg/data/domain/appcatalog"
	"github.com/giantswarm/kubectl-gs/pkg/output"
)

func (r *runner) printOutput(appCatalogResource appcatalog.Resource) error {
	var (
		err      error
		printer  printers.ResourcePrinter
		resource runtime.Object
	)

	switch {
	case output.IsOutputDefault(r.flag.print.OutputFormat):
		resource = getTable(appCatalogResource)
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

func getAppCatalogEntryRow(ace applicationv1alpha1.AppCatalogEntry) metav1.TableRow {
	return metav1.TableRow{
		Cells: []interface{}{
			ace.Spec.Catalog.Name,
			ace.Spec.AppName,
			ace.Spec.AppVersion,
			ace.Spec.Version,
			ace.CreationTimestamp,
		},
	}
}

func getAppCatalogEntryTable(appCatalogResource appcatalog.Resource) *metav1.Table {
	// Creating a custom table resource.
	table := &metav1.Table{}

	table.ColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "Catalog", Type: "string"},
		{Name: "App Name", Type: "string"},
		{Name: "App Version", Type: "string"},
		{Name: "Version", Type: "string"},
		{Name: "Age", Type: "string"},
	}

	switch c := appCatalogResource.(type) {
	case *appcatalog.Collection:
		if len(c.Items) == 1 {
			item := c.Items[0]

			for _, ace := range item.Entries.Items {
				table.Rows = append(table.Rows, getAppCatalogEntryRow(ace))
			}
		}
	}

	return table
}

func getAppCatalogRow(a appcatalog.AppCatalog) metav1.TableRow {
	if a.CR == nil {
		return metav1.TableRow{}
	}

	return metav1.TableRow{
		Cells: []interface{}{
			a.CR.Name,
			a.CR.Spec.Storage.URL,
			a.CR.CreationTimestamp,
		},
		Object: runtime.RawExtension{
			Object: a.CR,
		},
	}
}

func getAppCatalogTable(appCatalogResource appcatalog.Resource) *metav1.Table {
	// Creating a custom table resource.
	table := &metav1.Table{}

	table.ColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string"},
		{Name: "Catalog URL", Type: "string"},
		{Name: "Age", Type: "string", Format: "date-time"},
	}

	switch c := appCatalogResource.(type) {
	case *appcatalog.Collection:
		for _, appCatalogItem := range c.Items {
			table.Rows = append(table.Rows, getAppCatalogRow(appCatalogItem))
		}
	}

	return table
}

func getTable(appCatalogResource appcatalog.Resource) *metav1.Table {
	switch c := appCatalogResource.(type) {
	case *appcatalog.Collection:
		if len(c.Items) == 1 {
			catalog := c.Items[0]
			if len(catalog.Entries.Items) > 0 {
				return getAppCatalogEntryTable(appCatalogResource)
			}
		}

		return getAppCatalogTable(appCatalogResource)
	}

	return nil
}
