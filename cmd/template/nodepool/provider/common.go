package provider

import (
	corev1alpha1 "github.com/giantswarm/apiextensions/v6/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
)

type NodePoolCRsConfig struct {
	// AWS only.
	AWSInstanceType                     string
	MachineDeploymentSubnet             string
	OnDemandBaseCapacity                int
	OnDemandPercentageAboveBaseCapacity int
	UseAlikeInstanceTypes               bool
	EKS                                 bool

	// Azure only.
	VMSize            string
	AzureUseSpotVms   bool
	AzureSpotMaxPrice float32

	// Common.
	FileName          string
	NodePoolName      string
	AvailabilityZones []string
	ClusterName       string
	Description       string
	NodesMax          int
	NodesMin          int
	Organization      string
	ReleaseComponents map[string]string
	ReleaseVersion    string
	Namespace         string
}

func newcapiMachinePoolCR(config NodePoolCRsConfig, infrastructureRef *corev1.ObjectReference, bootstrapConfigRef *corev1.ObjectReference) *capiexp.MachinePool {
	mp := &capiexp.MachinePool{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MachinePool",
			APIVersion: "exp.cluster.x-k8s.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.NodePoolName,
			Namespace: config.Namespace,
			Labels: map[string]string{
				label.Cluster:              config.ClusterName,
				capi.ClusterLabelName:      config.ClusterName,
				label.MachinePool:          config.NodePoolName,
				label.Organization:         config.Organization,
				label.ReleaseVersion:       config.ReleaseVersion,
				label.AzureOperatorVersion: config.ReleaseComponents["azure-operator"],
			},
			Annotations: map[string]string{
				annotation.MachinePoolName: config.Description,
			},
		},
		Spec: capiexp.MachinePoolSpec{
			ClusterName:    config.ClusterName,
			Replicas:       toInt32Ptr(int32(config.NodesMin)),
			FailureDomains: config.AvailabilityZones,
			Template: capi.MachineTemplateSpec{
				Spec: capi.MachineSpec{
					ClusterName:       config.ClusterName,
					InfrastructureRef: *infrastructureRef,
					Bootstrap: capi.Bootstrap{
						ConfigRef: bootstrapConfigRef,
					},
				},
			},
		},
	}

	return mp
}

func newSparkCR(config NodePoolCRsConfig) *corev1alpha1.Spark {
	spark := &corev1alpha1.Spark{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Spark",
			APIVersion: "core.giantswarm.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.NodePoolName,
			Namespace: config.Namespace,
			Labels: map[string]string{
				label.Cluster:         config.ClusterName,
				capi.ClusterLabelName: config.ClusterName,
				label.ReleaseVersion:  config.ReleaseVersion,
			},
		},
		Spec: corev1alpha1.SparkSpec{},
	}

	return spark
}

func toInt32Ptr(i int32) *int32 {
	return &i
}
