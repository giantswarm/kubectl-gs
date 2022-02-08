package provider

import (
	"bytes"
	"context"
	"io"
	"text/template"

	corev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
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
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/yaml"
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
	NodePoolID        string
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

func newCAPIV1Alpha3MachinePoolCR(config NodePoolCRsConfig, infrastructureRef *corev1.ObjectReference, bootstrapConfigRef *corev1.ObjectReference) *expcapiv1alpha3.MachinePool {
	mp := &expcapiv1alpha3.MachinePool{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MachinePool",
			APIVersion: "exp.cluster.x-k8s.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.NodePoolID,
			Namespace: config.Namespace,
			Labels: map[string]string{
				label.Cluster:                 config.ClusterName,
				capiv1alpha3.ClusterLabelName: config.ClusterName,
				label.MachinePool:             config.NodePoolID,
				label.Organization:            config.Organization,
			},
			Annotations: map[string]string{
				annotation.MachinePoolName: config.Description,
			},
		},
		Spec: expcapiv1alpha3.MachinePoolSpec{
			ClusterName:    config.ClusterName,
			Replicas:       toInt32Ptr(int32(config.NodesMin)),
			FailureDomains: config.AvailabilityZones,
			Template: capiv1alpha3.MachineTemplateSpec{
				Spec: capiv1alpha3.MachineSpec{
					ClusterName:       config.ClusterName,
					InfrastructureRef: *infrastructureRef,
					Bootstrap: capiv1alpha3.Bootstrap{
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
			Name:      config.NodePoolID,
			Namespace: config.Namespace,
			Labels: map[string]string{
				label.Cluster:                 config.ClusterName,
				capiv1alpha3.ClusterLabelName: config.ClusterName,
			},
		},
		Spec: corev1alpha1.SparkSpec{},
	}

	return spark
}

func toInt32Ptr(i int32) *int32 {
	return &i
}

func runMutation(ctx context.Context, client k8sclient.Interface, templateData interface{}, input []string, output io.Writer) error {
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
	for _, t := range input {
		// Add separators to make the entire file valid yaml and allow easy appending.
		_, err = output.Write([]byte("---\n"))
		if err != nil {
			return microerror.Mask(err)
		}
		te := template.Must(template.New("resource").Parse(t))
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
