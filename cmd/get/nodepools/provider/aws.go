package provider

import (
	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capiv1alpha2 "sigs.k8s.io/cluster-api/api/v1alpha2"
)

func GetAWSTable(resource runtime.Object) *metav1.Table {
	resCollection, ok := resource.(*v1.List)
	if !ok {
		return nil
	}
	table := &metav1.Table{}

	table.ColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "ID", Type: "string"},
	}

	for _, nodePool := range resCollection.Items {
		np, ok := nodePool.Object.(*v1.List)
		if !ok || len(np.Items) != 2 {
			continue
		}

		md, ok := np.Items[0].Object.(*capiv1alpha2.MachineDeployment)
		if !ok {
			continue
		}

		awsMD, ok := np.Items[1].Object.(*infrastructurev1alpha2.AWSMachineDeployment)
		if !ok {
			continue
		}

		table.Rows = append(table.Rows, getAWSNodePoolRow(md, awsMD))
	}

	return table
}

func getAWSNodePoolRow(
	md *capiv1alpha2.MachineDeployment,
	awsMD *infrastructurev1alpha2.AWSMachineDeployment,
) metav1.TableRow {
	return metav1.TableRow{
		Cells: []interface{}{
			md.GetName(),
		},
		Object: runtime.RawExtension{
			Object: md,
		},
	}
}
