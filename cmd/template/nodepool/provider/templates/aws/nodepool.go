package aws

import (
	"github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1alpha3 "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/giantswarm/kubectl-gs/internal/key"
)

const (
	kindAWSMachineDeployment = "AWSMachineDeployment"
)

// +k8s:deepcopy-gen=false

type NodePoolCRsConfig struct {
	AvailabilityZones                   []string
	AWSInstanceType                     string
	ClusterName                         string
	MachineDeploymentName               string
	Description                         string
	EnableLongNames                     bool
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
	// the workload cluster name may be provided by the user.
	{
		if config.ClusterName == "" {
			generatedName, err := key.GenerateName(config.EnableLongNames)
			if err != nil {
				return NodePoolCRs{}, microerror.Mask(err)
			}

			config.ClusterName = generatedName
		}

		if config.MachineDeploymentName == "" {
			generatedName, err := key.GenerateName(config.EnableLongNames)
			if err != nil {
				return NodePoolCRs{}, microerror.Mask(err)
			}

			config.MachineDeploymentName = generatedName
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
			Name:      c.MachineDeploymentName,
			Namespace: metav1.NamespaceDefault,
			Annotations: map[string]string{
				annotation.Docs: "https://docs.giantswarm.io/ui-api/management-api/crd/awsmachinedeployments.infrastructure.giantswarm.io/",
			},
			Labels: map[string]string{
				label.AWSOperatorVersion:     c.ReleaseComponents["aws-operator"],
				label.Cluster:                c.ClusterName,
				label.MachineDeployment:      c.MachineDeploymentName,
				label.Organization:           c.Owner,
				label.ReleaseVersion:         c.ReleaseVersion,
				apiv1alpha3.ClusterLabelName: c.ClusterName,
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
			APIVersion: "cluster.x-k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.MachineDeploymentName,
			Namespace: metav1.NamespaceDefault,
			Annotations: map[string]string{
				annotation.Docs: "https://docs.giantswarm.io/ui-api/management-api/crd/machinedeployments.cluster.x-k8s.io/",
			},
			Labels: map[string]string{
				label.Cluster:                c.ClusterName,
				label.ClusterOperatorVersion: c.ReleaseComponents["cluster-operator"],
				label.MachineDeployment:      c.MachineDeploymentName,
				label.Organization:           c.Owner,
				label.ReleaseVersion:         c.ReleaseVersion,
				apiv1alpha3.ClusterLabelName: c.ClusterName,
			},
		},
		Spec: apiv1alpha3.MachineDeploymentSpec{
			ClusterName: c.ClusterName,
			Template: apiv1alpha3.MachineTemplateSpec{
				Spec: apiv1alpha3.MachineSpec{
					ClusterName: c.ClusterName,
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
