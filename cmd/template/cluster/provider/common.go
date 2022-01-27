package provider

import (
	"bytes"
	"context"
	"io"
	"text/template"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apiyaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
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

	// App settings
	ClusterAppCatalog           string
	ClusterAppUserConfigMap     string
	ClusterAppVersion           string
	DefaultAppsAppCatalog       string
	DefaultAppsAppUserConfigMap string
	DefaultAppsAppVersion       string
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

func runMutation(ctx context.Context, client k8sclient.Interface, templateData interface{}, templates []templateConfig, output io.Writer) error {
	var err error

	// Create a DiscoveryClient to get the GVR.
	dc, err := discovery.NewDiscoveryClientForConfig(client.RESTConfig())
	if err != nil {
		return microerror.Mask(err)
	}

	// Create a DynamicClient to execute our request.
	dyn, err := dynamic.NewForConfig(client.RESTConfig())
	if err != nil {
		return microerror.Mask(err)
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))
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

		// Transform to unstructured.Unstructured.
		obj := &unstructured.Unstructured{}
		dec := apiyaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		_, gvk, err := dec.Decode(buf.Bytes(), nil, obj)
		if err != nil {
			return microerror.Mask(err)
		}
		// Mapping needed for the dynamic client.
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return microerror.Mask(err)
		}
		// Get namespace information from object.
		namespace, namespaced, err := unstructured.NestedString(obj.Object, "metadata", "namespace")
		if err != nil {
			return microerror.Mask(err)
		}
		var defaultedObj *unstructured.Unstructured
		// Execute our request as a `DryRun` - no resources will be persisted.
		if namespaced {
			defaultedObj, err = dyn.Resource(mapping.Resource).Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{DryRun: []string{"All"}})
			if err != nil {
				// TODO handle different kinds of errors (e.g. validation) here.
				return microerror.Mask(err)
			}
		} else {
			defaultedObj, err = dyn.Resource(mapping.Resource).Create(ctx, obj, metav1.CreateOptions{DryRun: []string{"All"}})
			if err != nil {
				// TODO handle different kinds of errors (e.g. validation) here.
				return microerror.Mask(err)
			}
		}
		// Strip `managedFields` for better readability.
		unstructured.RemoveNestedField(defaultedObj.Object, "metadata", "managedFields")
		// Strip some metadata fields for better UX.
		unstructured.RemoveNestedField(defaultedObj.Object, "metadata", "creationTimestamp")
		unstructured.RemoveNestedField(defaultedObj.Object, "metadata", "generation")
		unstructured.RemoveNestedField(defaultedObj.Object, "metadata", "selfLink")
		unstructured.RemoveNestedField(defaultedObj.Object, "metadata", "uid")
		// Unstructured to JSON.
		buf = *new(bytes.Buffer)
		err = dec.Encode(defaultedObj, &buf)
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
