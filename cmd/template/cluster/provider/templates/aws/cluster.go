package aws

import (
	"github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/giantswarm/kubectl-gs/v2/internal/key"
)

const (
	defaultControlPlaneInstanceType = "m5.xlarge"
	kindAWSCluster                  = "AWSCluster"
	kindAWSControlPlane             = "AWSControlPlane"
	kindG8sControlPlane             = "G8sControlPlane"
)

// +k8s:deepcopy-gen=false

type ClusterCRsConfig struct {
	ClusterName              string
	ControlPlaneName         string
	Credential               string
	Domain                   string
	ExternalSNAT             bool
	ControlPlaneAZ           []string
	ControlPlaneInstanceType string
	Description              string
	PodsCIDR                 string
	Owner                    string
	Region                   string
	ReleaseComponents        map[string]string
	ReleaseVersion           string
	Labels                   map[string]string
	NetworkPool              string
	ServicePriority          string
}

// +k8s:deepcopy-gen=false

type ClusterCRs struct {
	Cluster         *capi.Cluster
	AWSCluster      *v1alpha3.AWSCluster
	G8sControlPlane *v1alpha3.G8sControlPlane
	AWSControlPlane *v1alpha3.AWSControlPlane
}

func NewClusterCRs(config ClusterCRsConfig) (ClusterCRs, error) {
	// Default some essentials in case certain information are not given. E.g.
	// the workload cluster name may be provided by the user.
	{
		if config.ClusterName == "" {
			generatedName, err := key.GenerateName()
			if err != nil {
				return ClusterCRs{}, microerror.Mask(err)
			}

			config.ClusterName = generatedName
		}

		if config.ControlPlaneName == "" {
			generatedName, err := key.GenerateName()
			if err != nil {
				return ClusterCRs{}, microerror.Mask(err)
			}

			config.ControlPlaneName = generatedName
		}

		if config.ControlPlaneInstanceType == "" {
			config.ControlPlaneInstanceType = defaultControlPlaneInstanceType
		}
	}

	awsClusterCR := newAWSClusterCR(config)
	clusterCR := newClusterCR(awsClusterCR, config)
	awsControlPlaneCR := newAWSControlPlaneCR(config)
	g8sControlPlaneCR := newG8sControlPlaneCR(awsControlPlaneCR, config)

	crs := ClusterCRs{
		Cluster:         clusterCR,
		AWSCluster:      awsClusterCR,
		G8sControlPlane: g8sControlPlaneCR,
		AWSControlPlane: awsControlPlaneCR,
	}

	return crs, nil
}

func newAWSClusterCR(c ClusterCRsConfig) *v1alpha3.AWSCluster {
	awsClusterCR := &v1alpha3.AWSCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       kindAWSCluster,
			APIVersion: v1alpha3.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.ClusterName,
			Namespace: metav1.NamespaceDefault,
			Annotations: map[string]string{
				annotation.Docs: "https://docs.giantswarm.io/use-the-api/management-api/crd/awsclusters.infrastructure.giantswarm.io/",
			},
			Labels: map[string]string{
				label.AWSOperatorVersion: c.ReleaseComponents["aws-operator"],
				label.Cluster:            c.ClusterName,
				label.Organization:       c.Owner,
				label.ReleaseVersion:     c.ReleaseVersion,
				capi.ClusterNameLabel:    c.ClusterName,
			},
		},
		Spec: v1alpha3.AWSClusterSpec{
			Cluster: v1alpha3.AWSClusterSpecCluster{
				Description: c.Description,
				DNS: v1alpha3.AWSClusterSpecClusterDNS{
					Domain: c.Domain,
				},
				OIDC: v1alpha3.AWSClusterSpecClusterOIDC{},
			},
			Provider: v1alpha3.AWSClusterSpecProvider{
				CredentialSecret: v1alpha3.AWSClusterSpecProviderCredentialSecret{
					Name:      c.Credential,
					Namespace: "giantswarm",
				},
				Pods: v1alpha3.AWSClusterSpecProviderPods{
					CIDRBlock:    c.PodsCIDR,
					ExternalSNAT: &c.ExternalSNAT,
				},
				Nodes: v1alpha3.AWSClusterSpecProviderNodes{
					NetworkPool: c.NetworkPool,
				},
				Region: c.Region,
			},
		},
	}

	// Single master node
	if len(c.ControlPlaneAZ) == 1 {
		awsClusterCR.Spec.Provider.Master = v1alpha3.AWSClusterSpecProviderMaster{
			AvailabilityZone: c.ControlPlaneAZ[0],
			InstanceType:     c.ControlPlaneInstanceType,
		}
	}

	return awsClusterCR
}

func newAWSControlPlaneCR(c ClusterCRsConfig) *v1alpha3.AWSControlPlane {
	return &v1alpha3.AWSControlPlane{
		TypeMeta: metav1.TypeMeta{
			Kind:       kindAWSControlPlane,
			APIVersion: v1alpha3.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.ControlPlaneName,
			Namespace: metav1.NamespaceDefault,
			Annotations: map[string]string{
				annotation.Docs: "https://docs.giantswarm.io/use-the-api/management-api/crd/awscontrolplanes.infrastructure.giantswarm.io/",
			},
			Labels: map[string]string{
				label.AWSOperatorVersion: c.ReleaseComponents["aws-operator"],
				label.Cluster:            c.ClusterName,
				label.ControlPlane:       c.ControlPlaneName,
				label.Organization:       c.Owner,
				label.ReleaseVersion:     c.ReleaseVersion,
				capi.ClusterNameLabel:    c.ClusterName,
			},
		},
		Spec: v1alpha3.AWSControlPlaneSpec{
			AvailabilityZones: c.ControlPlaneAZ,
			InstanceType:      c.ControlPlaneInstanceType,
		},
	}
}

func newClusterCR(obj *v1alpha3.AWSCluster, c ClusterCRsConfig) *capi.Cluster {
	clusterLabels := map[string]string{}
	{
		for key, value := range c.Labels {
			clusterLabels[key] = value
		}

		gsLabels := map[string]string{
			label.ClusterOperatorVersion: c.ReleaseComponents["cluster-operator"],
			label.Cluster:                c.ClusterName,
			capi.ClusterNameLabel:        c.ClusterName,
			label.Organization:           c.Owner,
			label.ReleaseVersion:         c.ReleaseVersion,

			// According to RFC https://github.com/giantswarm/rfc/tree/main/classify-cluster-priority
			// we use "highest" as the default service priority.
			label.ServicePriority: c.ServicePriority,
		}

		for key, value := range gsLabels {
			clusterLabels[key] = value
		}
	}

	clusterCR := &capi.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "cluster.x-k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.ClusterName,
			Namespace: metav1.NamespaceDefault,
			Annotations: map[string]string{
				annotation.Docs: "https://docs.giantswarm.io/use-the-api/management-api/crd/clusters.cluster.x-k8s.io/",
			},
			Labels: clusterLabels,
		},
		Spec: capi.ClusterSpec{
			InfrastructureRef: &corev1.ObjectReference{
				APIVersion: obj.TypeMeta.APIVersion,
				Kind:       obj.TypeMeta.Kind,
				Name:       obj.GetName(),
				Namespace:  obj.GetNamespace(),
			},
		},
	}

	if val, ok := c.Labels[label.ServicePriority]; ok {
		clusterCR.ObjectMeta.Labels[label.ServicePriority] = val
	}

	return clusterCR
}

func newG8sControlPlaneCR(obj *v1alpha3.AWSControlPlane, c ClusterCRsConfig) *v1alpha3.G8sControlPlane {
	return &v1alpha3.G8sControlPlane{
		TypeMeta: metav1.TypeMeta{
			Kind:       kindG8sControlPlane,
			APIVersion: v1alpha3.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.ControlPlaneName,
			Namespace: metav1.NamespaceDefault,
			Annotations: map[string]string{
				annotation.Docs: "https://docs.giantswarm.io/use-the-api/management-api/crd/g8scontrolplanes.infrastructure.giantswarm.io/",
			},
			Labels: map[string]string{
				label.ClusterOperatorVersion: c.ReleaseComponents["cluster-operator"],
				label.Cluster:                c.ClusterName,
				label.ControlPlane:           c.ControlPlaneName,
				label.Organization:           c.Owner,
				label.ReleaseVersion:         c.ReleaseVersion,
				capi.ClusterNameLabel:        c.ClusterName,
			},
		},
		Spec: v1alpha3.G8sControlPlaneSpec{
			Replicas: len(c.ControlPlaneAZ),
			InfrastructureRef: corev1.ObjectReference{
				APIVersion: obj.TypeMeta.APIVersion,
				Kind:       obj.TypeMeta.Kind,
				Name:       obj.GetName(),
				Namespace:  obj.GetNamespace(),
			},
		},
	}
}
