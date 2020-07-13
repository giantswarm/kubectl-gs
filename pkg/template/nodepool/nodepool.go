package nodepool

import (
	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	infrastructurev1alpha2scheme "github.com/giantswarm/apiextensions/pkg/clientset/versioned/scheme"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
	apiv1alpha2 "sigs.k8s.io/cluster-api/api/v1alpha2"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/internal/label"
)

type Config struct {
	AvailabilityZones                   []string
	AWSInstanceType                     string
	ClusterID                           string
	Name                                string
	NodesMax                            int
	NodesMin                            int
	OnDemandBaseCapacity                int
	OnDemandPercentageAboveBaseCapacity int
	Owner                               string
	ReleaseComponents                   map[string]string
	ReleaseVersion                      string
}

func NewMachineDeploymentCRs(config Config) (*apiv1alpha2.MachineDeployment, *infrastructurev1alpha2.AWSMachineDeployment, error) {

	machineDeploymentID := key.GenerateID()

	awsMachineDeploymentCR, err := newAWSMachineDeploymentCR(config.ClusterID, machineDeploymentID, config)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	machineDeploymentCR, err := newMachineDeploymentCR(awsMachineDeploymentCR, config.ClusterID, machineDeploymentID, config)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	return machineDeploymentCR, awsMachineDeploymentCR, nil
}

func newMachineDeploymentCR(obj interface{}, clusterID, machineDeploymentID string, c Config) (*apiv1alpha2.MachineDeployment, error) {
	runtimeObj, _ := obj.(runtime.Object)

	infrastructureCRRef, err := reference.GetReference(infrastructurev1alpha2scheme.Scheme, runtimeObj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	machineDeploymentCR := &apiv1alpha2.MachineDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MachineDeployment",
			APIVersion: "cluster.x-k8s.io/v1alpha2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      machineDeploymentID,
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				label.Cluster:                clusterID,
				label.ClusterOperatorVersion: c.ReleaseComponents["cluster-operator"],
				label.MachineDeployment:      machineDeploymentID,
				label.Organization:           c.Owner,
				label.ReleaseVersion:         c.ReleaseVersion,
			},
		},
		Spec: apiv1alpha2.MachineDeploymentSpec{
			Template: apiv1alpha2.MachineTemplateSpec{
				Spec: apiv1alpha2.MachineSpec{
					InfrastructureRef: *infrastructureCRRef,
				},
			},
		},
	}

	return machineDeploymentCR, nil
}

func newAWSMachineDeploymentCR(clusterID, machineDeploymentID string, c Config) (*infrastructurev1alpha2.AWSMachineDeployment, error) {
	awsMachineDeploymentCR := &infrastructurev1alpha2.AWSMachineDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AWSMachineDeployment",
			APIVersion: "infrastructure.giantswarm.io/v1alpha2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      machineDeploymentID,
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				label.AWSOperatorVersion: c.ReleaseComponents["aws-operator"],
				label.Cluster:            clusterID,
				label.MachineDeployment:  machineDeploymentID,
				label.Organization:       c.Owner,
				label.ReleaseVersion:     c.ReleaseVersion,
			},
		},
		Spec: infrastructurev1alpha2.AWSMachineDeploymentSpec{
			NodePool: infrastructurev1alpha2.AWSMachineDeploymentSpecNodePool{
				Description: c.Name,
				Machine: infrastructurev1alpha2.AWSMachineDeploymentSpecNodePoolMachine{
					DockerVolumeSizeGB:  100,
					KubeletVolumeSizeGB: 100,
				},
				Scaling: infrastructurev1alpha2.AWSMachineDeploymentSpecNodePoolScaling{
					Max: c.NodesMax,
					Min: c.NodesMin,
				},
			},
			Provider: infrastructurev1alpha2.AWSMachineDeploymentSpecProvider{
				AvailabilityZones: c.AvailabilityZones,
				Worker: infrastructurev1alpha2.AWSMachineDeploymentSpecProviderWorker{
					InstanceType: c.AWSInstanceType,
				},
				InstanceDistribution: infrastructurev1alpha2.AWSMachineDeploymentSpecInstanceDistribution{
					OnDemandBaseCapacity:                c.OnDemandBaseCapacity,
					OnDemandPercentageAboveBaseCapacity: &c.OnDemandPercentageAboveBaseCapacity,
				},
			},
		},
	}

	return awsMachineDeploymentCR, nil
}
