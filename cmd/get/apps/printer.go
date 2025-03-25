package apps

import (
	"fmt"

	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"

	"github.com/giantswarm/kubectl-gs/v5/pkg/data/domain/app"
	"github.com/giantswarm/kubectl-gs/v5/pkg/output"
)

func (r *runner) printOutput(appResource app.Resource) error {
	var (
		err      error
		printer  printers.ResourcePrinter
		resource runtime.Object
	)

	switch {
	case output.IsOutputDefault(r.flag.print.OutputFormat):
		resource = getTable(appResource)
		printOptions := printers.PrintOptions{
			WithNamespace: r.flag.AllNamespaces,
		}
		printer = printers.NewTablePrinter(printOptions)
	case output.IsOutputName(r.flag.print.OutputFormat):
		resource = appResource.Object()
		err = output.PrintResourceNames(r.stdout, resource)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil

	default:
		resource = appResource.Object()
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
	_, _ = fmt.Fprintf(r.stdout, "No App CRD found.\n")
	_, _ = fmt.Fprintf(r.stdout, "Please check you are accessing a management cluster\n\n")
}

func (r *runner) printNoResourcesOutput() {
	_, _ = fmt.Fprintf(r.stdout, "No apps found.\n")
	_, _ = fmt.Fprintf(r.stdout, "To create an app, please check\n\n")
	_, _ = fmt.Fprintf(r.stdout, "  kubectl gs template app --help\n")
}

func getTable(appResource app.Resource) *metav1.Table {
	// Creating a custom table resource.
	table := &metav1.Table{}

	table.ColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string"},
		{Name: "Version", Type: "string"},
		{Name: "Last Deployed", Type: "string", Format: "date-time"},
		{Name: "Status", Type: "string"},
		{Name: "Notes", Type: "string"},
	}

	switch c := appResource.(type) {
	case *app.App:
		table.Rows = append(table.Rows, getAppRow(*c))
	case *app.Collection:
		for _, appItem := range c.Items {
			table.Rows = append(table.Rows, getAppRow(appItem))
		}
	}

	return table
}

func getAppRow(a app.App) metav1.TableRow {
	if a.CR == nil {
		return metav1.TableRow{}
	}

	return metav1.TableRow{
		Cells: []interface{}{
			a.CR.Name,
			a.CR.Status.Version,
			output.TranslateTimestampSince(a.CR.Status.Release.LastDeployed),
			a.CR.Status.Release.Status,
			a.CR.Status.Release.Reason,
		},
		Object: runtime.RawExtension{
			Object: a.CR,
		},
	}
}
