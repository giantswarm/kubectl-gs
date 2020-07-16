package cluster

import (
	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	infrastructurev1alpha2scheme "github.com/giantswarm/apiextensions/pkg/clientset/versioned/scheme"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/reference"
	apiv1alpha2 "sigs.k8s.io/cluster-api/api/v1alpha2"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/internal/label"
)

const (
	defaultMasterInstanceType = "m5.xlarge"
)

type Config struct {
	ClusterID         string
	ControlPlaneID    string
	Credential        string
	Domain            string
	ExternalSNAT      bool
	MasterAZ          []string
	Name              string
	PodsCIDR          string
	Owner             string
	Region            string
	ReleaseComponents map[string]string
	ReleaseVersion    string
	Labels            map[string]string
}

type CRs struct {
	Cluster         *apiv1alpha2.Cluster
	AWSCluster      *infrastructurev1alpha2.AWSCluster
	G8sControlPlane *infrastructurev1alpha2.G8sControlPlane
	AWSControlPlane *infrastructurev1alpha2.AWSControlPlane
}

// NewCRs returns the custom resources to represent the given cluster.
func NewCRs(config Config) (CRs, error) {
	// Default some essentials in case certain information are not given. E.g.
	// the Tenant Cluster ID may be provided by the user.
	{
		if config.ClusterID == "" {
			config.ClusterID = key.GenerateID()
		}
		if config.ControlPlaneID == "" {
			config.ControlPlaneID = key.GenerateID()
		}
	}

	awsClusterCR := newAWSClusterCR(config)
	clusterCR, err := newClusterCR(awsClusterCR, config)
	if err != nil {
		return CRs{}, microerror.Mask(err)
	}

	awsControlPlaneCR := newAWSControlPlaneCR(config)
	g8sControlPlaneCR, err := newG8sControlPlaneCR(awsControlPlaneCR, config)
	if err != nil {
		return CRs{}, microerror.Mask(err)
	}

	crs := CRs{
		Cluster:         clusterCR,
		AWSCluster:      awsClusterCR,
		G8sControlPlane: g8sControlPlaneCR,
		AWSControlPlane: awsControlPlaneCR,
	}

	return crs, nil
}

func newAWSClusterCR(c Config) *infrastructurev1alpha2.AWSCluster {
	awsClusterCR := &infrastructurev1alpha2.AWSCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AWSCluster",
			APIVersion: "infrastructure.giantswarm.io/v1alpha2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.ClusterID,
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				label.AWSOperatorVersion: c.ReleaseComponents["aws-operator"],
				label.Cluster:            c.ClusterID,
				label.Organization:       c.Owner,
				label.ReleaseVersion:     c.ReleaseVersion,
			},
		},
		Spec: infrastructurev1alpha2.AWSClusterSpec{
			Cluster: infrastructurev1alpha2.AWSClusterSpecCluster{
				Description: c.Name,
				DNS: infrastructurev1alpha2.AWSClusterSpecClusterDNS{
					Domain: c.Domain,
				},
				OIDC: infrastructurev1alpha2.AWSClusterSpecClusterOIDC{},
			},
			Provider: infrastructurev1alpha2.AWSClusterSpecProvider{
				CredentialSecret: infrastructurev1alpha2.AWSClusterSpecProviderCredentialSecret{
					Name:      c.Credential,
					Namespace: "giantswarm",
				},
				Pods: infrastructurev1alpha2.AWSClusterSpecProviderPods{
					CIDRBlock:    c.PodsCIDR,
					ExternalSNAT: &c.ExternalSNAT,
				},
				Region: c.Region,
			},
		},
	}

	// Single master node
	if len(c.MasterAZ) == 1 {
		awsClusterCR.Spec.Provider.Master = infrastructurev1alpha2.AWSClusterSpecProviderMaster{
			AvailabilityZone: c.MasterAZ[0],
			InstanceType:     defaultMasterInstanceType,
		}
	}

	return awsClusterCR
}

func newAWSControlPlaneCR(c Config) *infrastructurev1alpha2.AWSControlPlane {
	return &infrastructurev1alpha2.AWSControlPlane{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AWSControlPlane",
			APIVersion: "infrastructure.giantswarm.io/v1alpha2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.ControlPlaneID,
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				label.AWSOperatorVersion: c.ReleaseComponents["aws-operator"],
				label.Cluster:            c.ClusterID,
				label.ControlPlane:       c.ControlPlaneID,
				label.Organization:       c.Owner,
				label.ReleaseVersion:     c.ReleaseVersion,
			},
		},
		Spec: infrastructurev1alpha2.AWSControlPlaneSpec{
			AvailabilityZones: c.MasterAZ,
			InstanceType:      defaultMasterInstanceType,
		},
	}
}

func newClusterCR(obj *infrastructurev1alpha2.AWSCluster, c Config) (*apiv1alpha2.Cluster, error) {
	infrastructureCRRef, err := reference.GetReference(infrastructurev1alpha2scheme.Scheme, obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	clusterLabels := map[string]string{}
	{
		for key, value := range c.Labels {
			clusterLabels[key] = value
		}

		gsLabels := map[string]string{
			label.ClusterOperatorVersion: c.ReleaseComponents["cluster-operator"],
			label.Cluster:                c.ClusterID,
			label.Organization:           c.Owner,
			label.ReleaseVersion:         c.ReleaseVersion,
		}

		for key, value := range gsLabels {
			clusterLabels[key] = value
		}
	}

	clusterCR := &apiv1alpha2.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "cluster.x-k8s.io/v1alpha2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.ClusterID,
			Namespace: metav1.NamespaceDefault,
			Labels:    clusterLabels,
		},
		Spec: apiv1alpha2.ClusterSpec{
			InfrastructureRef: infrastructureCRRef,
		},
	}

	return clusterCR, nil
}

func newG8sControlPlaneCR(obj *infrastructurev1alpha2.AWSControlPlane, c Config) (*infrastructurev1alpha2.G8sControlPlane, error) {
	infrastructureCRRef, err := reference.GetReference(infrastructurev1alpha2scheme.Scheme, obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	cr := &infrastructurev1alpha2.G8sControlPlane{
		TypeMeta: metav1.TypeMeta{
			Kind:       "G8sControlPlane",
			APIVersion: "infrastructure.giantswarm.io/v1alpha2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.ControlPlaneID,
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				label.ClusterOperatorVersion: c.ReleaseComponents["cluster-operator"],
				label.Cluster:                c.ClusterID,
				label.ControlPlane:           c.ControlPlaneID,
				label.Organization:           c.Owner,
				label.ReleaseVersion:         c.ReleaseVersion,
			},
		},
		Spec: infrastructurev1alpha2.G8sControlPlaneSpec{
			Replicas:          len(c.MasterAZ),
			InfrastructureRef: *infrastructureCRRef,
		},
	}

	return cr, nil
}
