package apps

import (
	"fmt"

	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/printers"

	"github.com/giantswarm/kubectl-gs/pkg/data/domain/app"
	"github.com/giantswarm/kubectl-gs/pkg/output"
)

func (r *runner) printOutput(results app.ValidationResults) error {
	var (
		err     error
		printer printers.ResourcePrinter
	)

	switch {
	case output.IsOutputDefault(r.flag.print.OutputFormat):
		printOptions := printers.PrintOptions{}
		printer = printers.NewTablePrinter(printOptions)

	case output.IsOutputReport(r.flag.print.OutputFormat):
		err = PrintReport(results)
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

	resource := getTable(results)

	err = printer.PrintObj(resource, r.stdout)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func getTable(results app.ValidationResults) *metav1.Table {
	table := &metav1.Table{}

	table.ColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "NAMESPACE", Type: "string"},
		{Name: "NAME", Type: "string"},
		{Name: "VERSION", Type: "string"},
		{Name: "ERRORS", Type: "string"},
	}

	for _, result := range results {
		table.Rows = append(table.Rows, getAppRow(result))
	}

	return table
}

func getAppRow(validationResult *app.ValidationResult) metav1.TableRow {
	var errorCount string

	errorCount = fmt.Sprintf("%d", len(validationResult.ValidationErrors))

	if app.IsNoSchema(validationResult.Err) {
		errorCount = "n/a: app has no values schema"
	} else if validationResult.Err != nil {
		errorCount = validationResult.Err.Error()
	}

	return metav1.TableRow{
		Cells: []interface{}{
			validationResult.App.Namespace,
			validationResult.App.Name,
			validationResult.App.Spec.Version,
			errorCount,
		},
	}
}
