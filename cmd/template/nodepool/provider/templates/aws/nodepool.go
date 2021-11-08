package aws

import (
	"github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/apiextensions/v3/pkg/id"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
)

const (
	kindAWSMachineDeployment = "AWSMachineDeployment"
)

// +k8s:deepcopy-gen=false

type NodePoolCRsConfig struct {
	AvailabilityZones                   []string
	AWSInstanceType                     string
	ClusterID                           string
	MachineDeploymentID                 string
	Description                         string
	NodesMax                            int
	NodesMin                            int
	OnDemandBaseCapacity                int
	OnDemandPercentageAboveBaseCapacity int
	Owner                               string
	ReleaseComponents                   map[string]string
	ReleaseVersion                      string
	UseAlikeInstanceTypes               bool
}

// +k8s:deepcopy-gen=false

type NodePoolCRs struct {
	MachineDeployment    *apiv1alpha3.MachineDeployment
	AWSMachineDeployment *v1alpha3.AWSMachineDeployment
}

func NewNodePoolCRs(config NodePoolCRsConfig) (NodePoolCRs, error) {
	// Default some essentials in case certain information are not given. E.g.
	// the workload cluster ID may be provided by the user.
	{
		if config.ClusterID == "" {
			config.ClusterID = id.Generate()
		}
		if config.MachineDeploymentID == "" {
			config.MachineDeploymentID = id.Generate()
		}
	}

	awsMachineDeploymentCR := newAWSMachineDeploymentCR(config)
	machineDeploymentCR := newMachineDeploymentCR(awsMachineDeploymentCR, config)

	crs := NodePoolCRs{
		MachineDeployment:    machineDeploymentCR,
		AWSMachineDeployment: awsMachineDeploymentCR,
	}

	return crs, nil
}

func newAWSMachineDeploymentCR(c NodePoolCRsConfig) *v1alpha3.AWSMachineDeployment {
	return &v1alpha3.AWSMachineDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       kindAWSMachineDeployment,
			APIVersion: v1alpha3.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.MachineDeploymentID,
			Namespace: metav1.NamespaceDefault,
			Annotations: map[string]string{
				annotation.Docs: "https://docs.giantswarm.io/ui-api/management-api/crd/awsmachinedeployments.infrastructure.giantswarm.io/",
			},
			Labels: map[string]string{
				label.AWSOperatorVersion:     c.ReleaseComponents["aws-operator"],
				label.Cluster:                c.ClusterID,
				label.MachineDeployment:      c.MachineDeploymentID,
				label.Organization:           c.Owner,
				label.ReleaseVersion:         c.ReleaseVersion,
				apiv1alpha3.ClusterLabelName: c.ClusterID,
			},
		},
		Spec: v1alpha3.AWSMachineDeploymentSpec{
			NodePool: v1alpha3.AWSMachineDeploymentSpecNodePool{
				Description: c.Description,
				Machine: v1alpha3.AWSMachineDeploymentSpecNodePoolMachine{
					DockerVolumeSizeGB:  100,
					KubeletVolumeSizeGB: 100,
				},
				Scaling: v1alpha3.AWSMachineDeploymentSpecNodePoolScaling{
					Max: c.NodesMax,
					Min: c.NodesMin,
				},
			},
			Provider: v1alpha3.AWSMachineDeploymentSpecProvider{
				AvailabilityZones: c.AvailabilityZones,
				Worker: v1alpha3.AWSMachineDeploymentSpecProviderWorker{
					InstanceType:          c.AWSInstanceType,
					UseAlikeInstanceTypes: c.UseAlikeInstanceTypes,
				},
				InstanceDistribution: v1alpha3.AWSMachineDeploymentSpecInstanceDistribution{
					OnDemandBaseCapacity:                c.OnDemandBaseCapacity,
					OnDemandPercentageAboveBaseCapacity: &c.OnDemandPercentageAboveBaseCapacity,
				},
			},
		},
	}
}

func newMachineDeploymentCR(obj *v1alpha3.AWSMachineDeployment, c NodePoolCRsConfig) *apiv1alpha3.MachineDeployment {
	return &apiv1alpha3.MachineDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MachineDeployment",
			APIVersion: "cluster.x-k8s.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.MachineDeploymentID,
			Namespace: metav1.NamespaceDefault,
			Annotations: map[string]string{
				annotation.Docs: "https://docs.giantswarm.io/ui-api/management-api/crd/machinedeployments.cluster.x-k8s.io/",
			},
			Labels: map[string]string{
				label.Cluster:                c.ClusterID,
				label.ClusterOperatorVersion: c.ReleaseComponents["cluster-operator"],
				label.MachineDeployment:      c.MachineDeploymentID,
				label.Organization:           c.Owner,
				label.ReleaseVersion:         c.ReleaseVersion,
				apiv1alpha3.ClusterLabelName: c.ClusterID,
			},
		},
		Spec: apiv1alpha3.MachineDeploymentSpec{
			ClusterName: c.ClusterID,
			Template: apiv1alpha3.MachineTemplateSpec{
				Spec: apiv1alpha3.MachineSpec{
					ClusterName: c.ClusterID,
					InfrastructureRef: corev1.ObjectReference{
						APIVersion: obj.TypeMeta.APIVersion,
						Kind:       obj.TypeMeta.Kind,
						Name:       obj.GetName(),
						Namespace:  obj.GetNamespace(),
					},
				},
			},
		},
	}
}
