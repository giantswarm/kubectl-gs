package apps

import (
	"fmt"

	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/printers"

	"github.com/giantswarm/kubectl-gs/pkg/data/domain/app"
	"github.com/giantswarm/kubectl-gs/pkg/output"
	"github.com/giantswarm/kubectl-gs/pkg/pluralize"
)

func (r *runner) printOutput(results app.ValidationResults) error {
	var (
		err     error
		printer printers.ResourcePrinter
	)

	switch {
	case output.IsOutputDefault(&r.flag.OutputFormat):
		printOptions := printers.PrintOptions{}
		printer = printers.NewTablePrinter(printOptions)

	case output.IsOutputReport(&r.flag.OutputFormat):
		err = PrintReport(results)
		if err != nil {
			return microerror.Mask(err)
		}
		return nil

	default:
		return microerror.Maskf(invalidFlagError, "Unknown output format: %s", r.flag.OutputFormat)
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
		{Name: "RESULT", Type: "string"},
	}

	for _, result := range results {
		table.Rows = append(table.Rows, getAppRow(result))
	}

	return table
}

func getAppRow(validationResult *app.ValidationResult) metav1.TableRow {
	var errorCount string

	pluralizedError := pluralize.Pluralize("error", len(validationResult.ValidationErrors))
	errorCount = fmt.Sprintf("%d %s", len(validationResult.ValidationErrors), pluralizedError)

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
