package cluster

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/giantswarm/k8sclient/v8/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/common"
	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/flags"
	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/provider"
	"github.com/giantswarm/kubectl-gs/v6/internal/key"
	"github.com/giantswarm/kubectl-gs/v6/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v6/pkg/labels"
)

type runner struct {
	commonConfig *commonconfig.CommonConfig
	flag         *flags.Flag
	logger       micrologger.Logger
	stdout       io.Writer
	stderr       io.Writer
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Sorting is required before validation for uniqueness.
	sort.Slice(r.flag.ControlPlaneAZ, func(i, j int) bool {
		return r.flag.ControlPlaneAZ[i] < r.flag.ControlPlaneAZ[j]
	})

	err := r.flag.Validate(cmd)
	if err != nil {
		return microerror.Mask(err)
	}

	client, err := r.commonConfig.GetClient(r.logger)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.run(ctx, client)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, client k8sclient.Interface) error {
	output := r.stdout
	if r.flag.Output != "" {
		outFile, err := os.Create(r.flag.Output)
		if err != nil {
			return microerror.Mask(err)
		}

		defer func() { _ = outFile.Close() }()
		output = outFile
	}

	config, err := r.getClusterConfig()
	if err != nil {
		return microerror.Mask(err)
	}

	config.UseReleaseChart = r.useReleaseChart(ctx, client, config)

	switch r.flag.Provider {
	case key.ProviderCAPA:
		err = provider.WriteCAPATemplate(ctx, client, output, config)
		if err != nil {
			return microerror.Mask(err)
		}
	case key.ProviderCAPZ:
		err = provider.WriteCAPZTemplate(ctx, client, output, config)
		if err != nil {
			return microerror.Mask(err)
		}
	case key.ProviderAKS:
		err = provider.WriteAKSTemplate(ctx, client, output, config)
		if err != nil {
			return microerror.Mask(err)
		}
	case key.ProviderEKS:
		err = provider.WriteEKSTemplate(ctx, client, output, config)
		if err != nil {
			return microerror.Mask(err)
		}
	case key.ProviderVSphere:
		err = provider.WriteVSphereTemplate(ctx, client, output, config)
		if err != nil {
			return microerror.Mask(err)
		}
	case key.ProviderCloudDirector:
		err = provider.WriteCloudDirectorTemplate(ctx, client, output, config)
		if err != nil {
			return microerror.Mask(err)
		}
	default:
		return microerror.Mask(templateFlagNotImplemented)
	}

	return nil
}

// useReleaseChart tells whether the cluster App CR should point to the
// release-<provider> chart (the flow used since release v35) or to the
// cluster-<provider> chart.
//
// The release-<provider> chart has the release version baked in, so it only
// works for releases which are actually published as a chart. Developers
// commonly create a Release CR by hand to test a change (e.g. a copy of
// `aws-35.0.0` named `aws-35.0.0-andreas`). For those, no release-<provider>
// chart exists and we have to fall back to the cluster-<provider> chart, which
// resolves the Release CR at runtime.
func (r *runner) useReleaseChart(ctx context.Context, client k8sclient.Interface, config common.ClusterConfig) bool {
	if !common.IsReleaseVersion(config.ReleaseVersion) {
		return false
	}

	clusterChart, releaseChart := providerChartNames(r.flag.Provider)
	if releaseChart == "" {
		return false
	}

	// An explicitly requested cluster-<provider> chart version is a clear
	// opt-in to the cluster-<provider> chart, as used for testing.
	if config.App.ClusterVersion != "" {
		return false
	}

	available, err := common.ReleaseChartAvailable(ctx, client.CtrlClient(), releaseChart, config.App.ClusterCatalog, config.ReleaseVersion)
	if err != nil {
		// We cannot tell, so we stick to the default flow for releases.
		r.warnf("Warning: could not check whether the %s chart exists in the %s catalog: %s\n", releaseChart, config.App.ClusterCatalog, err)
		return true
	}

	if !available {
		r.warnf(
			"Warning: no %s chart with version %s found in the %s catalog, using the %s chart instead. Make sure the Release CR for %s exists in the management cluster.\n",
			releaseChart, config.ReleaseVersion, config.App.ClusterCatalog, clusterChart, config.ReleaseVersion,
		)
		return false
	}

	return true
}

func (r *runner) warnf(format string, a ...interface{}) {
	if r.stderr == nil {
		return
	}

	_, _ = fmt.Fprintf(r.stderr, format, a...)
}

// providerChartNames returns the names of the cluster-<provider> and
// release-<provider> charts for the given provider.
func providerChartNames(providerName string) (clusterChart string, releaseChart string) {
	switch providerName {
	case key.ProviderCAPA:
		return provider.ClusterAWSRepoName, provider.ReleaseAWSRepoName
	case key.ProviderCAPZ:
		return provider.ClusterAzureRepoName, provider.ReleaseAzureRepoName
	case key.ProviderAKS:
		return provider.ClusterAKSRepoName, provider.ReleaseAKSRepoName
	case key.ProviderEKS:
		return provider.ClusterEKSRepoName, provider.ReleaseEKSRepoName
	case key.ProviderVSphere:
		return provider.ClusterVsphereRepoName, provider.ReleaseVsphereRepoName
	case key.ProviderCloudDirector:
		return provider.ClusterCloudDirectorRepoName, provider.ReleaseCloudDirectorRepoName
	default:
		return "", ""
	}
}

func (r *runner) getClusterConfig() (common.ClusterConfig, error) {
	config := common.ClusterConfig{
		ControlPlaneAZ:           r.flag.ControlPlaneAZ,
		ControlPlaneInstanceType: r.flag.ControlPlaneInstanceType,
		Description:              r.flag.Description,
		KubernetesVersion:        r.flag.KubernetesVersion,
		ManagementCluster:        r.flag.ManagementCluster,
		Organization:             r.flag.Organization,
		PodsCIDR:                 r.flag.PodsCIDR,
		ReleaseVersion:           r.flag.Release,
		Namespace:                metav1.NamespaceDefault,
		Region:                   r.flag.Region,
		ServicePriority:          r.flag.ServicePriority,
		PreventDeletion:          r.flag.PreventDeletion,

		App:           r.flag.App,
		AWS:           r.flag.AWS,
		Azure:         r.flag.Azure,
		OIDC:          r.flag.OIDC,
		VSphere:       r.flag.VSphere,
		CloudDirector: r.flag.CloudDirector,
	}

	if r.flag.GenerateName {
		generatedName, err := key.GenerateName()
		if err != nil {
			return common.ClusterConfig{}, microerror.Mask(err)
		}

		config.Name = generatedName
	} else {
		config.Name = r.flag.Name
	}

	if config.Name == "" {
		return common.ClusterConfig{}, errors.New("logic error in name assignment")
	}

	// Remove leading 'v' from release flag input.
	config.ReleaseVersion = strings.TrimLeft(config.ReleaseVersion, "v")

	var err error
	config.Labels, err = labels.Parse(r.flag.Label)
	if err != nil {
		return common.ClusterConfig{}, microerror.Mask(err)
	}

	config.Namespace = key.OrganizationNamespaceFromName(config.Organization)

	return config, nil
}
