package releases

import (
	"fmt"
	"sort"
	"strings"

	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"

	"github.com/giantswarm/kubectl-gs/pkg/data/domain/release"
	"github.com/giantswarm/kubectl-gs/pkg/output"
)

const (
	naValue = "n/a"
)

type PrintOptions struct {
	Name string
}

func (r *runner) printOutput(rResource release.Resource) error {
	var err error
	var printer printers.ResourcePrinter
	var resource runtime.Object

	switch {
	case output.IsOutputDefault(r.flag.print.OutputFormat):
		table := &metav1.Table{
			ColumnDefinitions: []metav1.TableColumnDefinition{
				{Name: "Version", Type: "string"},
				{Name: "Status", Type: "string"},
				{Name: "Created", Type: "string", Format: "date-time"},
				{Name: "Kubernetes", Type: "string"},
				{Name: "Container Linux", Type: "string"},
				{Name: "CoreDNS", Type: "string"},
				{Name: "Calico", Type: "string"},
			},
		}

		switch r := rResource.(type) {
		case *release.Release:
			table.Rows = append(table.Rows, getTableRow(*r))
		case *release.ReleaseCollection:
			// Sort ASC by release version.
			sort.Slice(r.Items, func(i, j int) bool {
				return strings.Compare(r.Items[j].CR.Name, r.Items[i].CR.Name) > 0
			})
			for _, release := range r.Items {
				table.Rows = append(table.Rows, getTableRow(release))
			}
		}

		resource = table
		printOptions := printers.PrintOptions{
			WithNamespace: false,
		}
		printer = printers.NewTablePrinter(printOptions)

	case output.IsOutputName(r.flag.print.OutputFormat):
		switch res := rResource.(type) {
		case *release.Release:
			resource = res.Object()
		case *release.ReleaseCollection:
			resource = res.Object()
		}

		err = output.PrintResourceNames(r.stdout, resource)
		if err != nil {
			return microerror.Mask(err)
		}
		return nil

	default:
		switch res := rResource.(type) {
		case *release.Release:
			resource = res.Object()
		case *release.ReleaseCollection:
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
	fmt.Fprintf(r.stdout, "No releases found.\n")
}

func getTableRow(release release.Release) metav1.TableRow {
	if release.CR == nil {
		return metav1.TableRow{}
	}

	status := "inactive"
	if release.CR.Status.Ready {
		status = "active"
	}

	kubernetesVersion := naValue
	containerLinuxVersion := naValue
	coreDNSVersion := naValue
	calicoVersion := naValue

	for _, component := range release.CR.Spec.Components {
		if component.Name == "kubernetes" {
			kubernetesVersion = component.Version
		}
		if component.Name == "containerlinux" {
			containerLinuxVersion = component.Version
		}
		if component.Name == "coredns" {
			coreDNSVersion = component.Version
		}
		if component.Name == "calico" {
			calicoVersion = component.Version
		}
	}

	for _, app := range release.CR.Spec.Apps {
		if app.Name == "coredns" {
			coreDNSVersion = app.ComponentVersion
		}
	}

	return metav1.TableRow{
		Cells: []interface{}{
			release.CR.GetName(),
			status,
			release.CR.CreationTimestamp.UTC(),
			kubernetesVersion,
			containerLinuxVersion,
			coreDNSVersion,
			calicoVersion,
		},
		Object: runtime.RawExtension{
			Object: release.CR,
		},
	}
}
