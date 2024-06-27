package orgs

import (
	"fmt"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/kubectl-gs/v3/pkg/output"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"

	"github.com/giantswarm/kubectl-gs/v3/pkg/data/domain/organization"
)

type PrintOptions struct {
	Name string
}

func (r *runner) printOutput(orgResource organization.Resource) error {
	var err error
	var printer printers.ResourcePrinter
	var resource runtime.Object

	switch {
	case output.IsOutputDefault(r.flag.print.OutputFormat):
		table := &metav1.Table{
			ColumnDefinitions: []metav1.TableColumnDefinition{
				{Name: "Name", Type: "string"},
				{Name: "Org Namespace", Type: "string"},
				{Name: "Age", Type: "string", Format: "date-time"},
			},
		}

		switch o := orgResource.(type) {
		case *organization.Organization:
			table.Rows = append(table.Rows, getTableRow(*o))
		case *organization.Collection:
			sort.Slice(o.Items, func(i, j int) bool {
				return strings.Compare(o.Items[i].Organization.Name, o.Items[j].Organization.Name) <= 0
			})
			for _, org := range o.Items {
				table.Rows = append(table.Rows, getTableRow(org))
			}
		}

		resource = table
		printOptions := printers.PrintOptions{
			NoHeaders:        false,
			WithNamespace:    false,
			WithKind:         true,
			Wide:             true,
			ShowLabels:       false,
			AllowMissingKeys: true,
		}

		printer = printers.NewTablePrinter(printOptions)
	case output.IsOutputName(r.flag.print.OutputFormat):
		switch res := orgResource.(type) {
		case *organization.Organization:
			resource = res.Object()
		case *organization.Collection:
			resource = res.Object()
		}

		err = output.PrintResourceNames(r.stdout, resource)
		if err != nil {
			return microerror.Mask(err)
		}
		return nil
	default:
		switch res := orgResource.(type) {
		case *organization.Organization:
			resource = res.Object()
		case *organization.Collection:
			resource = res.Object()
		}

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
	fmt.Fprintf(r.stdout, "No organizations found.\n")
	fmt.Fprintf(r.stdout, "No resources of type organizations.security.giantswarm.io available. To create one, please check\n\n")
	fmt.Fprintf(r.stdout, "  https://docs.giantswarm.io/use-the-api/management-api/crd/organizations.security.giantswarm.io/\n")
}

func getTableRow(org organization.Organization) metav1.TableRow {
	return metav1.TableRow{
		Cells: []interface{}{
			org.Organization.Name,
			org.Organization.Status.Namespace,
			output.TranslateTimestampSince(org.Organization.CreationTimestamp),
		},
		Object: runtime.RawExtension{
			Object: org.Organization,
		},
	}
}
