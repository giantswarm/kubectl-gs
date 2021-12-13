package provider

import (
	"bytes"
	"context"
	"io"
	"text/template"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/yaml"
)

type ClusterCRsConfig struct {
	// AWS only.
	EKS                bool
	ExternalSNAT       bool
	ControlPlaneSubnet string

	// OpenStack only.
	Cloud                string   // OPENSTACK_CLOUD
	CloudConfig          string   // <no equivalent env var>>
	DNSNameservers       []string // OPENSTACK_DNS_NAMESERVERS
	ExternalNetworkID    string   // <no equivalent env var>
	FailureDomain        string   // OPENSTACK_FAILURE_DOMAIN
	ImageName            string   // OPENSTACK_IMAGE_NAME
	NodeCIDR             string   // <no equivalent env var>
	NodeMachineFlavor    string   // OPENSTACK_NODE_MACHINE_FLAVOR
	RootVolumeDiskSize   string   // <no equivalent env var>
	RootVolumeSourceType string   // <no equivalent env var>
	RootVolumeSourceUUID string   // <no equivalent env var>

	// Common.
	FileName       string
	ControlPlaneAZ []string
	Description    string
	Name           string
	Organization   string
	ReleaseVersion string
	Labels         map[string]string
	Namespace      string
	PodsCIDR       string
}

type templateConfig struct {
	Name string
	Data string
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

func runMutation(ctx context.Context, templateData interface{}, templates []templateConfig, output io.Writer) error {
	var err error

	for _, t := range templates {
		// Add separators to make the entire file valid yaml and allow easy appending.
		_, err = output.Write([]byte("---\n"))
		if err != nil {
			return microerror.Mask(err)
		}
		te := template.Must(template.New(t.Name).Parse(t.Data))
		var buf bytes.Buffer
		// Template from our inputs.
		err = te.Execute(&buf, templateData)
		if err != nil {
			return microerror.Mask(err)
		}
		// JSON to YAML.
		mutated, err := yaml.JSONToYAML(buf.Bytes())
		if err != nil {
			return microerror.Mask(err)
		}
		// Write the yaml to our file.
		_, err = output.Write(mutated)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	return nil
}
