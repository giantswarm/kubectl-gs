package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"text/template"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	apiyaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type AWSConfig struct {
	EKS                bool
	ExternalSNAT       bool
	ControlPlaneSubnet string

	// for CAPA
	Role                string
	MachinePool         AWSMachinePoolConfig
	NetworkAZUsageLimit int
	NetworkVPCCIDR      string
	SSHSSOPublicKey     string
}

type AWSMachinePoolConfig struct {
	Name             string
	MinSize          int
	MaxSize          int
	AZs              []string
	InstanceType     string
	RootVolumeSizeGB int
	CustomNodeLabels []string
}

type GCPConfig struct {
	Project           string
	FailureDomains    []string
	MachineDeployment GCPMachineDeployment
}

type GCPMachineDeployment struct {
	Name             string
	FailureDomain    string
	InstanceType     string
	Replicas         int
	RootVolumeSizeGB int
	CustomNodeLabels []string
}

type MachineConfig struct {
	BootFromVolume bool
	DiskSize       int
	Flavor         string
	Image          string
}

type OpenStackConfig struct {
	Cloud          string
	CloudConfig    string
	DNSNameservers []string

	ExternalNetworkID string
	NodeCIDR          string
	NetworkName       string
	SubnetName        string

	Bastion      MachineConfig
	ControlPlane MachineConfig
	Worker       MachineConfig

	WorkerFailureDomain string
	WorkerReplicas      int
}

type AppConfig struct {
	ClusterCatalog     string
	ClusterVersion     string
	DefaultAppsCatalog string
	DefaultAppsVersion string
}

type ClusterConfig struct {
	KubernetesVersion string
	FileName          string
	ControlPlaneAZ    []string
	Description       string
	Name              string
	Organization      string
	ReleaseVersion    string
	ReleaseComponents map[string]string
	Labels            map[string]string
	Namespace         string
	PodsCIDR          string
	OIDC              OIDC

	Region                   string
	BastionInstanceType      string
	BastionReplicas          int
	ControlPlaneInstanceType string

	App       AppConfig
	AWS       AWSConfig
	GCP       GCPConfig
	OpenStack OpenStackConfig
}

type OIDC struct {
	IssuerURL     string
	CAFile        string
	ClientID      string
	UsernameClaim string
	GroupsClaim   string
}

type templateConfig struct {
	Name string
	Data string
}

func newcapiClusterCR(config ClusterConfig, infrastructureRef *corev1.ObjectReference) *capi.Cluster {
	cluster := &capi.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			Labels: map[string]string{
				label.Cluster:                config.Name,
				capi.ClusterLabelName:        config.Name,
				label.Organization:           config.Organization,
				label.ReleaseVersion:         config.ReleaseVersion,
				label.AzureOperatorVersion:   config.ReleaseComponents["azure-operator"],
				label.ClusterOperatorVersion: config.ReleaseComponents["cluster-operator"],
			},
			Annotations: map[string]string{
				annotation.ClusterDescription: config.Description,
			},
		},
		Spec: capi.ClusterSpec{
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

func getLatestVersion(ctx context.Context, ctrlClient client.Client, app, catalog string) (string, error) {
	var catalogEntryList applicationv1alpha1.AppCatalogEntryList
	err := ctrlClient.List(ctx, &catalogEntryList, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"app.kubernetes.io/name":            app,
			"application.giantswarm.io/catalog": catalog,
			"latest":                            "true",
		}),
		Namespace: "giantswarm",
	})

	if err != nil {
		return "", microerror.Mask(err)
	} else if len(catalogEntryList.Items) != 1 {
		message := fmt.Sprintf("version not specified for %s and latest release couldn't be determined in %s catalog", app, catalog)
		return "", microerror.Maskf(invalidFlagError, message)
	}

	return catalogEntryList.Items[0].Spec.Version, nil
}

func organizationNamespace(org string) string {
	return fmt.Sprintf("org-%s", org)
}

func userConfigMapName(app string) string {
	return fmt.Sprintf("%s-userconfig", app)
}
