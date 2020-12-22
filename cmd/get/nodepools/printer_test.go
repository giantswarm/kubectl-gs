package nodepools

import (
	goflag "flag"
	"time"

	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/kubectl-gs/internal/label"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	capiv1alpha2 "sigs.k8s.io/cluster-api/api/v1alpha2"
)

var update = goflag.Bool("update", false, "update .golden reference test files")

func newAWSMachineDeployment(id, created, release, description string, nodesMin, nodesMax int) *infrastructurev1alpha2.AWSMachineDeployment {
	location, _ := time.LoadLocation("UTC")
	parsedCreationDate, _ := time.ParseInLocation(time.RFC3339, created, location)
	c := &infrastructurev1alpha2.AWSMachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              id,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(parsedCreationDate),
			Labels: map[string]string{
				label.ReleaseVersion: release,
				label.Organization:   "giantswarm",
			},
		},
		Spec: infrastructurev1alpha2.AWSMachineDeploymentSpec{
			NodePool: infrastructurev1alpha2.AWSMachineDeploymentSpecNodePool{
				Description: description,
				Scaling: infrastructurev1alpha2.AWSMachineDeploymentSpecNodePoolScaling{
					Min: nodesMin,
					Max: nodesMax,
				},
			},
		},
	}

	c.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Group:   infrastructurev1alpha2.SchemeGroupVersion.Group,
		Version: infrastructurev1alpha2.SchemeGroupVersion.Version,
		Kind:    infrastructurev1alpha2.NewAWSMachineDeploymentTypeMeta().Kind,
	})

	return c
}

func newCAPIv1alpha2MachineDeployment(id, created, release string, nodesDesired, nodesReady int) *capiv1alpha2.MachineDeployment {
	location, _ := time.LoadLocation("UTC")
	parsedCreationDate, _ := time.ParseInLocation(time.RFC3339, created, location)
	c := &capiv1alpha2.MachineDeployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cluster.x-k8s.io/v1alpha2",
			Kind:       "MachineDeployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              id,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(parsedCreationDate),
			Labels: map[string]string{
				label.ReleaseVersion: release,
				label.Organization:   "giantswarm",
			},
		},
		Status: capiv1alpha2.MachineDeploymentStatus{
			Replicas:      int32(nodesDesired),
			ReadyReplicas: int32(nodesReady),
		},
	}

	return c
}
