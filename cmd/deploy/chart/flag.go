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
	flagChartName        = "chart-name"
	flagOrganization     = "organization"
	flagCluster          = "target-cluster"
	flagTargetNS         = "target-namespace"
	flagOCIURLPrefix     = "oci-url-prefix"
	flagVersion          = "version"
	flagValuesFile       = "values-file"
	flagName             = "name"
	flagAutoUpgrade      = "auto-upgrade"
	flagInterval         = "interval"
	flagRegistryUsername = "registry-username"
	flagRegistryPassword = "registry-password"

	defaultOCIURLPrefix = "oci://gsoci.azurecr.io/charts/giantswarm/"
	defaultInterval     = "10m"
)

var validAutoUpgradeValues = []string{"all", "minor", "patch"}

type flag struct {
	ChartName        string
	Organization     string
	Cluster          string
	TargetNS         string
	OCIURLPrefix     string
	Version          string
	ValuesFile       string
	Name             string
	AutoUpgrade      string
	Interval         string
	RegistryUsername string
	RegistryPassword string
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
	cmd.Flags().StringVar(&f.RegistryUsername, flagRegistryUsername, "", "Username for private OCI registry authentication.")
	cmd.Flags().StringVar(&f.RegistryPassword, flagRegistryPassword, "", "Password or token for private OCI registry authentication.")
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
	if f.Cluster == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagCluster)
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

	// Normalize OCI URL prefix.
	f.OCIURLPrefix = normalizeOCIURLPrefix(f.OCIURLPrefix)

	// Validate resource name length.
	resourceName := f.Name
	if resourceName == "" {
		resourceName = fmt.Sprintf("%s-%s", f.Cluster, f.ChartName)
	}
	if len(resourceName) > 253 {
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
