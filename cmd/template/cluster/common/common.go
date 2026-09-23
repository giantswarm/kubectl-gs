package common

import (
	"context"
	"fmt"
	"strings"

	semver "github.com/Masterminds/semver/v3"
	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/kubectl-gs/v6/internal/deploychart"
	"github.com/giantswarm/kubectl-gs/v6/internal/ociregistry"
)

var invalidFlagError = &microerror.Error{
	Kind: "invalidFlagError",
}

type AWSConfig struct {
	AWSClusterRoleIdentityName                     string
	MachinePool                                    AWSMachinePoolConfig
	NetworkAZUsageLimit                            int
	NetworkVPCCIDR                                 string
	ClusterType                                    string
	HttpProxy                                      string
	HttpsProxy                                     string
	NoProxy                                        string
	APIMode                                        string
	VPCMode                                        string
	TopologyMode                                   string
	PrefixListID                                   string
	TransitGatewayID                               string
	ControlPlaneLoadBalancerIngressAllowCIDRBlocks []string
	PublicSubnetMask                               int
	PrivateSubnetMask                              int
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

type AzureConfig struct {
	SubscriptionID           string
	ClusterIdentityName      string
	ClusterIdentityNamespace string
}

type CloudDirectorConfig struct {
	VipSubnet               string
	CredentialsSecretName   string
	ControlPlane            CloudDirectorControlPlane
	NetworkName             string
	Worker                  CloudDirectorMachineTemplate
	ServiceLoadBalancerCIDR string
	SvcLbIpPoolName         string
	HttpProxy               string
	HttpsProxy              string
	NoProxy                 string
	Org                     string
	Ovdc                    string
	Site                    string
	OvdcNetwork             string
}

type CloudDirectorControlPlane struct {
	Replicas        int
	MachineTemplate CloudDirectorMachineTemplate
}

type CloudDirectorMachineTemplate struct {
	DiskSizeGB   int
	SizingPolicy string
	Replicas     int
}

type VSphereConfig struct {
	ControlPlane            VSphereControlPlane
	CredentialsSecretName   string
	NetworkName             string
	Worker                  VSphereMachineTemplate
	ResourcePool            string
	ServiceLoadBalancerCIDR string
	SvcLbIpPoolName         string
}

type VSphereMachineTemplate struct {
	DiskGiB   int
	MemoryMiB int
	NumCPUs   int
	Replicas  int
}

type VSphereControlPlane struct {
	Ip         string
	IpPoolName string
	VSphereMachineTemplate
}

type ServiceAccount struct {
	Email  string
	Scopes []string
}

type MachineConfig struct {
	BootFromVolume bool
	DiskSize       int
	Flavor         string
	Image          string
}

type AppConfig struct {
	ClusterCatalog string
	ClusterVersion string
}

type ClusterConfig struct {
	ManagementCluster string
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
	PreventDeletion   bool

	Region                   string
	BastionInstanceType      string
	BastionReplicas          int
	ControlPlaneInstanceType string

	// UseReleaseChart tells whether the cluster App CR should point to the
	// release-<provider> chart instead of the cluster-<provider> chart.
	UseReleaseChart bool

	App           AppConfig
	AWS           AWSConfig
	Azure         AzureConfig
	VSphere       VSphereConfig
	CloudDirector CloudDirectorConfig
}

type OIDC struct {
	IssuerURL     string
	CAFile        string
	ClientID      string
	UsernameClaim string
	GroupsClaim   string
}

func GetLatestVersion(ctx context.Context, ctrlClient client.Client, app, catalog string) (string, error) {
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
		// Advice about logging into a management cluster is added centrally,
		// see pkg/errorprinter.
		return "", fmt.Errorf("failed to get the latest version of https://github.com/giantswarm/%s from the %s catalog: %w", app, catalog, microerror.Mask(err))
	} else if len(catalogEntryList.Items) != 1 {
		return "", microerror.Maskf(invalidFlagError, "version not specified for %s and latest release couldn't be uniquely determined in %s catalog", app, catalog)
	}

	return catalogEntryList.Items[0].Spec.Version, nil
}

func OrganizationNamespace(org string) string {
	return fmt.Sprintf("org-%s", org)
}

// BuildClusterFluxResources builds the OCIRepository and HelmRelease that
// deploy a release-<provider> chart via Flux, in place of the App CR used
// for pre-release-chart releases. userConfigMapName must point at a
// ConfigMap whose data key is "values" (see UserConfigMapName).
// extraValuesFrom are appended after it, so they take precedence.
func BuildClusterFluxResources(config ClusterConfig, releaseChart, userConfigMapName string, extraValuesFrom ...deploychart.ValuesFromReference) (ociRepoYAML, helmReleaseYAML []byte, err error) {
	namespace := OrganizationNamespace(config.Organization)

	ociRepo := deploychart.BuildOCIRepository(deploychart.OCIRepositoryOptions{
		Name:        config.Name,
		Namespace:   namespace,
		ClusterName: config.Name,
		URL:         fmt.Sprintf("oci://%s/%s%s", GSOCIRegistry, GSOCIChartsRepoPrefix, releaseChart),
		Version:     config.ReleaseVersion,
		Interval:    "10m",
		Timeout:     "60s",
	})

	helmRelease := deploychart.BuildHelmRelease(deploychart.HelmReleaseOptions{
		Name:               config.Name,
		Namespace:          namespace,
		ClusterName:        config.Name,
		ChartName:          config.Name,
		TargetNamespace:    namespace,
		Interval:           "5m",
		Timeout:            "10m",
		ManagementCluster:  true,
		ServiceAccountName: "automation",
		StorageNamespace:   namespace,
		InstallRemediation: &deploychart.RemediationPolicy{
			Retries: 10,
		},
		UpgradeRemediation: &deploychart.RemediationPolicy{
			Retries:              10,
			RemediateLastFailure: true,
			Strategy:             "rollback",
		},
		ValuesFrom: append([]deploychart.ValuesFromReference{
			{Kind: "ConfigMap", Name: userConfigMapName, ValuesKey: "values"},
		}, extraValuesFrom...),
	})

	ociRepoYAML, err = deploychart.MarshalManifest(ociRepo)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	helmReleaseYAML, err = deploychart.MarshalManifest(helmRelease)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	return ociRepoYAML, helmReleaseYAML, nil
}

func UserConfigMapName(app string) string {
	return fmt.Sprintf("%s-userconfig", app)
}

func DefaultTo(value string, defaultValue string) string {
	if value != "" {
		return value
	}
	return defaultValue
}

const (
	// ReleaseVersionMajorThreshold is the major version threshold above which
	// we assume the version is a Release version (not an original chart version).
	// Release versions (e.g., 35.0.0) use release-<provider> chart names.
	// Original chart versions (e.g., 7.2.5) use cluster-<provider> chart names.
	ReleaseVersionMajorThreshold = 35
)

// IsReleaseVersion determines if the given version is a Release version.
// Release versions have major version >= 35 and use release-<provider> chart names.
// Original chart versions have lower major versions and use cluster-<provider> chart names.
func IsReleaseVersion(version string) bool {
	v, err := semver.NewVersion(version)
	if err != nil {
		return false
	}

	return v.Major() >= ReleaseVersionMajorThreshold
}

// GSOCIRegistry and GSOCIChartsRepoPrefix locate Giant Swarm's public Helm
// chart OCI repository.
const (
	GSOCIRegistry         = "gsoci.azurecr.io"
	GSOCIChartsRepoPrefix = "charts/giantswarm/"
)

// ReleaseChartAvailable checks whether the given version of a
// release-<provider> chart is published in the gsoci OCI registry.
//
// Release CRs created by hand for testing purposes (e.g. `35.0.0-andreas`,
// copied from a released one) have no matching release-<provider> chart, so
// pulling that chart would fail. In such cases the cluster-<provider> chart
// has to be used instead, which resolves the Release CR at runtime.
func ReleaseChartAvailable(ctx context.Context, ociClient ociregistry.Client, app, version string) (bool, error) {
	exists, err := ociClient.TagExists(ctx, GSOCIRegistry, GSOCIChartsRepoPrefix+app, strings.TrimPrefix(version, "v"))
	if err != nil {
		return false, microerror.Mask(err)
	}

	return exists, nil
}
