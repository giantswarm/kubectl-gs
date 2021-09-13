package provider

import (
	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

type ClusterCRsConfig struct {
	// AWS only.
	ExternalSNAT       bool
	ControlPlaneSubnet string
	PodsCIDR           string

	// Common.
	FileName       string
	ControlPlaneAZ []string
	Description    string
	Name           string
	Organization   string
	ReleaseVersion string
	Labels         map[string]string
	Namespace      string
}

func newCAPIV1Alpha3ClusterCR(config ClusterCRsConfig, infrastructureRef *corev1.ObjectReference) *capiv1alpha3.Cluster {
	cluster := &capiv1alpha3.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "cluster.x-k8s.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			Labels: map[string]string{
				label.Cluster:                 config.Name,
				capiv1alpha3.ClusterLabelName: config.Name,
				label.Organization:            config.Organization,
				label.ReleaseVersion:          config.ReleaseVersion,
			},
			Annotations: map[string]string{
				annotation.ClusterDescription: config.Description,
			},
		},
		Spec: capiv1alpha3.ClusterSpec{
			InfrastructureRef: infrastructureRef,
		},
	}

	return cluster
}
