package provider

import (
	"bytes"
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	ExternalSNAT       bool
	ControlPlaneSubnet string
	PodsCIDR           string

	// Common.
	FileName       string
	ControlPlaneAZ []string
	Description    string
	Name           string
	Owner          string
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
				label.Organization:            config.Owner,
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

func runMutation(ctx context.Context, client k8sclient.Interface, input []byte) ([]byte, error) {
	obj := &unstructured.Unstructured{}
	dec := apiyaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, gvk, err := dec.Decode(input, nil, obj)
	if err != nil {
		return []byte{}, err
	}

	// Create a DiscoveryClient to get the GVR.
	dc, err := discovery.NewDiscoveryClientForConfig(client.RESTConfig())
	if err != nil {
		return []byte{}, err
	}

	// Mapping needed for the dynamic client.
	mapping, err := findGVR(gvk, dc)
	if err != nil {
		return []byte{}, err
	}

	// Create a DynamicClient to execute our request.
	dyn, err := dynamic.NewForConfig(client.RESTConfig())
	if err != nil {
		return []byte{}, err
	}

	namespace, namespaced, err := unstructured.NestedString(obj.Object, "metadata", "namespace")
	if err != nil {
		return []byte{}, err
	}

	var defaultedObj *unstructured.Unstructured
	// Execute our request as a `DryRun` - no resources will be persisted.
	if namespaced {
		defaultedObj, err = dyn.Resource(mapping.Resource).Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{DryRun: []string{"All"}})
		if err != nil {
			// TODO handle different kinds of errors (e.g. validation) here.
			return []byte{}, err
		}
	} else {
		defaultedObj, err = dyn.Resource(mapping.Resource).Create(ctx, obj, metav1.CreateOptions{DryRun: []string{"All"}})
		if err != nil {
			// TODO handle different kinds of errors (e.g. validation) here.
			return []byte{}, err
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
	buf := new(bytes.Buffer)
	err = dec.Encode(defaultedObj, buf)
	if err != nil {
		return []byte{}, err
	}

	// JSON to YAML.
	yamlOutput, err := yaml.JSONToYAML(buf.Bytes())
	if err != nil {
		return []byte{}, err
	}

	return yamlOutput, nil

}

func findGVR(gvk *schema.GroupVersionKind, dc *discovery.DiscoveryClient) (*meta.RESTMapping, error) {
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))
	return mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
}
