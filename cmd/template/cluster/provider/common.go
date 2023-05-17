package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"strings"
	"text/template"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	k8smetadata "github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
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

	"github.com/giantswarm/kubectl-gs/v2/internal/key"
	"github.com/giantswarm/kubectl-gs/v2/pkg/app"
	templateapp "github.com/giantswarm/kubectl-gs/v2/pkg/template/app"
)

type AWSConfig struct {
	EKS                bool
	ExternalSNAT       bool
	ControlPlaneSubnet string

	// for CAPA
	AWSClusterRoleIdentityName string
	MachinePool                AWSMachinePoolConfig
	NetworkAZUsageLimit        int
	NetworkVPCCIDR             string
	ClusterType                string
	HttpProxy                  string
	HttpsProxy                 string
	NoProxy                    string
	APIMode                    string
	VPCMode                    string
	DNSMode                    string
	TopologyMode               string
	PrefixListID               string
	TransitGatewayID           string
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
	ControlPlane      GCPControlPlane
	MachineDeployment GCPMachineDeployment
}

type GCPControlPlane struct {
	ServiceAccount ServiceAccount
}

type ServiceAccount struct {
	Email  string
	Scopes []string
}

type GCPMachineDeployment struct {
	Name             string
	FailureDomain    string
	InstanceType     string
	Replicas         int
	RootVolumeSizeGB int
	CustomNodeLabels []string
	ServiceAccount   ServiceAccount
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
	ServicePriority   string

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
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "cluster.x-k8s.io/v1beta1",
		},
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

				// According to RFC https://github.com/giantswarm/rfc/tree/main/classify-cluster-priority
				// we use "highest" as the default service priority.
				label.ServicePriority: config.ServicePriority,
			},
			Annotations: map[string]string{
				annotation.ClusterDescription: config.Description,
			},
		},
		Spec: capi.ClusterSpec{
			InfrastructureRef: infrastructureRef,
		},
	}

	if val, ok := config.Labels[label.ServicePriority]; ok {
		cluster.ObjectMeta.Labels[label.ServicePriority] = val
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

func findNextPowerOfTwo(num int) int {
	log2OfNum := math.Ceil(math.Log2(float64(num)))
	return int(math.Pow(2, log2OfNum+1))
}

func defaultTo(value string, defaultValue string) string {
	if value != "" {
		return value
	}
	return defaultValue
}

// validateYAML validates the given yaml against the cluster specific app values schema
func ValidateYAML(ctx context.Context, logger micrologger.Logger, client k8sclient.Interface, clusterApp applicationv1alpha1.App, yaml map[string]interface{}) error {

	serviceConfig := app.Config{
		Client: client,
		Logger: logger,
	}
	service, err := app.New(serviceConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	_, resultJsonValidate, err := service.ValidateApp(ctx, &clusterApp, "", yaml)
	if err != nil {
		return microerror.Mask(err)
	}

	resultErrors := resultJsonValidate.Errors()
	var validationErrors []string
	if len(resultErrors) > 0 {
		for _, resultError := range resultErrors {
			validationErrors = append(validationErrors, fmt.Errorf("%s", resultError.Description()).Error())
		}
		// return all validation errors
		return microerror.Mask(fmt.Errorf(strings.Join(validationErrors, "; ")))
	}

	return nil
}

func GetClusterApp(ctx context.Context, client k8sclient.Interface, capiProvider, clusterAppCatalog, clusterAppVersion string) (applicationv1alpha1.App, error) {

	clusterApp := applicationv1alpha1.App{}

	clusterApp.Spec.Catalog = clusterAppCatalog

	clusterApp.Spec.Name = key.CAPIClusterApps(capiProvider)

	if clusterAppVersion == "" {
		latestAppVersion, err := getLatestVersion(ctx, client.CtrlClient(), clusterApp.Spec.Name, clusterApp.Spec.Catalog)
		if err != nil {
			return clusterApp, microerror.Mask(err)
		}
		clusterApp.Spec.Version = latestAppVersion
	} else {
		clusterApp.Spec.Version = clusterAppVersion
	}

	return clusterApp, nil

}

func GetDefaultApp(ctx context.Context, client k8sclient.Interface, capiProvider, clusterAppCatalog, clusterAppVersion string) (applicationv1alpha1.App, error) {

	clusterApp := applicationv1alpha1.App{}

	clusterApp.Spec.Catalog = clusterAppCatalog

	clusterApp.Spec.Name = key.CAPIDefaultApps(capiProvider)

	if clusterAppVersion == "" {
		latestAppVersion, err := getLatestVersion(ctx, client.CtrlClient(), clusterApp.Spec.Name, clusterApp.Spec.Catalog)
		if err != nil {
			return clusterApp, microerror.Mask(err)
		}
		clusterApp.Spec.Version = latestAppVersion
	} else {
		clusterApp.Spec.Version = clusterAppVersion
	}

	return clusterApp, nil

}

// templateClusterApp templates the Cluster app
func TemplateClusterApp(ctx context.Context, output io.Writer, provider, clusterName, clusterOrganization string, clusterApp applicationv1alpha1.App, clusterAppConfigValues map[string]interface{}) error {

	appCR, err := templateapp.NewAppCR(templateapp.Config{
		AppName:                 clusterName, // should be probably the name from the flag aka the name of my cluster
		Catalog:                 clusterApp.Spec.Catalog,
		InCluster:               true,
		Name:                    clusterApp.Spec.Name,                       // should be hardcoeded
		Namespace:               organizationNamespace(clusterOrganization), // should be the name of the org given by the flag
		Version:                 clusterApp.Spec.Version,
		UserConfigConfigMapName: fmt.Sprintf("%s-user-values", clusterName),
	})
	if err != nil {
		return err
	}

	// marshal the given raw
	clusterAppConfigYAML, err := yaml.Marshal(clusterAppConfigValues)
	if err != nil {
		return microerror.Mask(err)
	}

	clusterAppConfigMap, err := templateapp.NewConfigMap(templateapp.UserConfig{
		Name:      fmt.Sprintf("%s-user-values", clusterName),
		Namespace: organizationNamespace(clusterOrganization),
		Data:      string(clusterAppConfigYAML),
	})
	if err != nil {
		return microerror.Mask(err)
	}

	// marshal the generated cluster specific config map for later templating
	clusterAppConfigMapRaw, err := yaml.Marshal(clusterAppConfigMap)
	if err != nil {
		return microerror.Mask(err)
	}

	t := template.Must(template.New("appCR").Parse(key.AppCRTemplate))

	err = t.Execute(output, templateapp.AppCROutput{
		AppCR:               string(appCR),
		UserConfigConfigMap: string(clusterAppConfigMapRaw),
	})
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// templateClusterApp templates the Cluster app
func TemplateDefaultApp(ctx context.Context, output io.Writer, provider, clusterName, clusterOrganization string, clusterApp applicationv1alpha1.App, clusterAppConfigValues map[string]interface{}) error {

	appCR, err := templateapp.NewAppCR(templateapp.Config{
		Cluster:                 clusterName,
		AppName:                 fmt.Sprintf("%s-default-apps", clusterName),
		Catalog:                 clusterApp.Spec.Catalog,
		InCluster:               true,
		Name:                    clusterApp.Spec.Name,
		Namespace:               organizationNamespace(clusterOrganization),
		Version:                 clusterApp.Spec.Version,
		UserConfigConfigMapName: fmt.Sprintf("%s-default-apps-user-values", clusterName),
		UseClusterValuesConfig:  true,
		ExtraLabels: map[string]string{
			k8smetadata.ManagedBy: "cluster",
		},
	})
	if err != nil {
		return err
	}

	// marshal the given raw
	clusterAppConfigYAML, err := yaml.Marshal(clusterAppConfigValues)
	if err != nil {
		return microerror.Mask(err)
	}

	clusterAppConfigMap, err := templateapp.NewConfigMap(templateapp.UserConfig{
		Name:      fmt.Sprintf("%s-default-apps-user-values", clusterName),
		Namespace: organizationNamespace(clusterOrganization),
		Data:      string(clusterAppConfigYAML),
	})
	if err != nil {
		return microerror.Mask(err)
	}

	// marshal the generated cluster specific config map for later templating
	clusterAppConfigMapRaw, err := yaml.Marshal(clusterAppConfigMap)
	if err != nil {
		return microerror.Mask(err)
	}

	t := template.Must(template.New("appCR").Parse(key.AppCRTemplate))

	err = t.Execute(output, templateapp.AppCROutput{
		AppCR:               string(appCR),
		UserConfigConfigMap: string(clusterAppConfigMapRaw),
	})
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
