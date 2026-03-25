package chart

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

// semverRe matches a basic semver string like "1.2.3" or "v1.2.3".
var semverRe = regexp.MustCompile(`^v?\d+\.\d+\.\d+$`)

const (
	flagChartName         = "chart-name"
	flagOrganization      = "organization"
	flagCluster           = "target-cluster"
	flagTargetNS          = "target-namespace"
	flagOCIURLPrefix      = "oci-url-prefix"
	flagVersion           = "version"
	flagValuesFile        = "values-file"
	flagName              = "name"
	flagAutoUpgrade       = "auto-upgrade"
	flagInterval          = "interval"
	flagRegistryUsername  = "registry-username"
	flagRegistryProvider  = "registry-provider"
	flagValuesFrom        = "values-from"
	flagManagementCluster = "management-cluster"
	flagDryRun            = "dry-run"

	envRegistryPassword = "KUBECTL_GS_REGISTRY_PASSWORD" //nolint:gosec // Not a credential, just the env var name.

	defaultOCIURLPrefix = "oci://gsoci.azurecr.io/charts/giantswarm/"
	defaultInterval     = "10m"
)

var validAutoUpgradeValues = []string{"all", "minor", "patch"}
var validRegistryProviders = []string{"aws", "azure", "gcp"}

type flag struct {
	ChartName         string
	Organization      string
	Cluster           string
	TargetNS          string
	OCIURLPrefix      string
	Version           string
	ValuesFile        string
	Name              string
	AutoUpgrade       string
	Interval          string
	RegistryUsername  string
	RegistryProvider  string
	ValuesFrom        []string
	ManagementCluster bool
	DryRun            bool
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.ChartName, flagChartName, "", "Name of the chart to deploy.")
	cmd.Flags().StringVar(&f.Organization, flagOrganization, "", "Giant Swarm organization name owning the target cluster.")
	cmd.Flags().StringVar(&f.Cluster, flagCluster, "", "Name of the target workload cluster.")
	cmd.Flags().StringVar(&f.TargetNS, flagTargetNS, "", "Target namespace in the workload cluster.")
	cmd.Flags().StringVar(&f.OCIURLPrefix, flagOCIURLPrefix, defaultOCIURLPrefix, "OCI URL prefix for the chart registry.")
	cmd.Flags().StringVar(&f.Version, flagVersion, "", "Chart version to deploy. If not specified, no version ref is set.")
	cmd.Flags().StringVar(&f.ValuesFile, flagValuesFile, "", "Path to a YAML file with chart values.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Override the resource name (default: <cluster>-<chart-name>).")
	cmd.Flags().StringVar(&f.AutoUpgrade, flagAutoUpgrade, "", "Auto-upgrade strategy: all, minor, or patch.")
	cmd.Flags().StringVar(&f.Interval, flagInterval, defaultInterval, "Reconciliation interval for OCIRepository and HelmRelease.")
	cmd.Flags().StringVar(&f.RegistryUsername, flagRegistryUsername, "", "Username for private OCI registry authentication. Password is read from "+envRegistryPassword+" or prompted interactively.")
	cmd.Flags().StringVar(&f.RegistryProvider, flagRegistryProvider, "", "Cloud provider for registry authentication via workload identity (aws, azure, gcp). When set, no registry Secret is created.")
	cmd.Flags().StringSliceVar(&f.ValuesFrom, flagValuesFrom, nil, "Reference to a ConfigMap or Secret containing chart values (format: ConfigMap/name or Secret/name). Can be specified multiple times.")
	cmd.Flags().BoolVar(&f.ManagementCluster, flagManagementCluster, false, "Deploy to the management cluster itself. Cluster name is derived from the current kubectl context.")
	cmd.Flags().BoolVar(&f.DryRun, flagDryRun, false, "Perform server-side validation without applying. Prints manifests to stdout.")
}

func (f *flag) Validate() error {
	if f.ChartName == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagChartName)
	}
	if strings.Contains(f.ChartName, "/") {
		return microerror.Maskf(invalidFlagError, "--%s must not contain '/'", flagChartName)
	}
	if f.Organization == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagOrganization)
	}
	if f.ManagementCluster && f.Cluster != "" {
		return microerror.Maskf(invalidFlagError, "--%s and --%s are mutually exclusive", flagManagementCluster, flagCluster)
	}
	if f.Cluster == "" && !f.ManagementCluster {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty (or use --%s)", flagCluster, flagManagementCluster)
	}
	if f.TargetNS == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagTargetNS)
	}

	if f.Version != "" && !semverRe.MatchString(f.Version) {
		return microerror.Maskf(invalidFlagError, "--%s must be a valid semver version (e.g. 1.2.3), got %q", flagVersion, f.Version)
	}

	if f.AutoUpgrade != "" {
		valid := false
		for _, v := range validAutoUpgradeValues {
			if f.AutoUpgrade == v {
				valid = true
				break
			}
		}
		if !valid {
			return microerror.Maskf(invalidFlagError, "--%s must be one of: %s", flagAutoUpgrade, strings.Join(validAutoUpgradeValues, ", "))
		}
	}

	if f.RegistryProvider != "" {
		valid := false
		for _, v := range validRegistryProviders {
			if f.RegistryProvider == v {
				valid = true
				break
			}
		}
		if !valid {
			return microerror.Maskf(invalidFlagError, "--%s must be one of: %s", flagRegistryProvider, strings.Join(validRegistryProviders, ", "))
		}
	}

	// Validate and normalize --values-from entries.
	for i, vf := range f.ValuesFrom {
		kind, name, ok := strings.Cut(vf, "/")
		if !ok || name == "" {
			return microerror.Maskf(invalidFlagError, "--%s %q must be in the format ConfigMap/name or Secret/name", flagValuesFrom, vf)
		}
		switch strings.ToLower(kind) {
		case "configmap":
			kind = "ConfigMap"
		case "secret":
			kind = "Secret"
		default:
			return microerror.Maskf(invalidFlagError, "--%s %q has invalid kind %q (must be ConfigMap or Secret)", flagValuesFrom, vf, kind)
		}
		f.ValuesFrom[i] = kind + "/" + name
	}

	// Normalize OCI URL prefix.
	f.OCIURLPrefix = normalizeOCIURLPrefix(f.OCIURLPrefix)

	// Validate resource name length (when we can compute it at flag-validation time).
	resourceName := f.Name
	if resourceName == "" && f.Cluster != "" {
		resourceName = fmt.Sprintf("%s-%s", f.Cluster, f.ChartName)
	}
	if resourceName != "" && len(resourceName) > 253 {
		return microerror.Maskf(invalidFlagError, "resource name %q exceeds maximum length of 253 characters", resourceName)
	}

	return nil
}

func normalizeOCIURLPrefix(prefix string) string {
	if !strings.HasPrefix(prefix, "oci://") {
		prefix = "oci://" + prefix
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}
	return prefix
}
