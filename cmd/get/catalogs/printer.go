package catalogs

import (
	"fmt"

	applicationv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"

	catalogdata "github.com/giantswarm/kubectl-gs/pkg/data/domain/catalog"
	"github.com/giantswarm/kubectl-gs/pkg/output"
)

func (r *runner) printOutput(catalogResource catalogdata.Resource, maxColWidth uint) error {
	var (
		err      error
		printer  printers.ResourcePrinter
		resource runtime.Object
	)

	switch {
	case output.IsOutputDefault(r.flag.print.OutputFormat):
		resource = getTable(catalogResource, maxColWidth)
		printOptions := printers.PrintOptions{}
		printer = printers.NewTablePrinter(printOptions)
	case output.IsOutputName(r.flag.print.OutputFormat):
		resource = catalogResource.Object()
		err = output.PrintResourceNames(r.stdout, resource)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil

	default:
		resource = catalogResource.Object()
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

func (r *runner) printNoMatchOutput() {
	fmt.Fprintf(r.stdout, "No Catalog CRD found.\n")
	fmt.Fprintf(r.stdout, "Please check you are accessing a management cluster\n\n")
}

func (r *runner) printNoResourcesOutput() {
	fmt.Fprintf(r.stdout, "No catalogs found.\n")
	fmt.Fprintf(r.stdout, "To create a catalog, please check\n\n")
	fmt.Fprintf(r.stdout, "  kubectl gs template catalog --help\n")
}

func getAppCatalogEntryDescription(description string, maxColWidth uint) string {
	if uint(len(description)) > maxColWidth {
		return fmt.Sprintf("%s...", description[:maxColWidth])
	}

	return description
}

func getAppCatalogEntryRow(ace applicationv1alpha1.AppCatalogEntry, maxColWidth uint) metav1.TableRow {
	return metav1.TableRow{
		Cells: []interface{}{
			ace.Spec.Catalog.Name,
			ace.Spec.AppName,
			ace.Spec.Version,
			ace.Spec.AppVersion,
			output.TranslateTimestampSince(ace.CreationTimestamp),
			getAppCatalogEntryDescription(ace.Spec.Chart.Description, maxColWidth),
		},
	}
}

func getCatalogEntryTable(catalogResource *catalogdata.Catalog, maxColWidth uint) *metav1.Table {
	// Creating a custom table resource.
	table := &metav1.Table{}

	table.ColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "Catalog", Type: "string"},
		{Name: "App Name", Type: "string"},
		{Name: "Version", Type: "string"},
		{Name: "Upstream Version", Type: "string"},
		{Name: "Age", Type: "string", Format: "date-time"},
		{Name: "Description", Type: "string", Format: "string"},
	}

	for _, ace := range catalogResource.Entries.Items {
		table.Rows = append(table.Rows, getAppCatalogEntryRow(ace, maxColWidth))
	}

	return table
}

func getCatalogRow(a catalogdata.Catalog) metav1.TableRow {
	if a.CR == nil {
		return metav1.TableRow{}
	}

	return metav1.TableRow{
		Cells: []interface{}{
			a.CR.Name,
			a.CR.Namespace,
			a.CR.Spec.Storage.URL,
			output.TranslateTimestampSince(a.CR.CreationTimestamp),
		},
		Object: runtime.RawExtension{
			Object: a.CR,
		},
	}
}

func getCatalogTable(catalogResource catalogdata.Resource) *metav1.Table {
	// Creating a custom table resource.
	table := &metav1.Table{}

	table.ColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string"},
		{Name: "Namespace", Type: "string"},
		{Name: "Catalog URL", Type: "string"},
		{Name: "Age", Type: "string", Format: "date-time"},
	}

	switch c := catalogResource.(type) {
	case *catalogdata.Catalog:
		table.Rows = append(table.Rows, getCatalogRow(*c))
	case *catalogdata.Collection:
		for _, catalogItem := range c.Items {
			table.Rows = append(table.Rows, getCatalogRow(catalogItem))
		}
	}

	return table
}

func getTable(catalogResource catalogdata.Resource, maxColWidth uint) *metav1.Table {
	switch c := catalogResource.(type) {
	case *catalogdata.Catalog:
		return getCatalogEntryTable(c, maxColWidth)
	case *catalogdata.Collection:
		return getCatalogTable(c)
	}

	return nil
}
