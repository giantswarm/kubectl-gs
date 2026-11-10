package wcluster

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	semver "github.com/Masterminds/semver/v3"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	clustercommon "github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/common"
	clusterprovider "github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/provider"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
	structure "github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/workload-cluster"
	commonkey "github.com/giantswarm/kubectl-gs/v6/internal/key"
	"github.com/giantswarm/kubectl-gs/v6/internal/ociregistry"
)

type runner struct {
	flag      *flag
	logger    micrologger.Logger
	fs        afero.Fs
	ociClient ociregistry.Client
	stdout    io.Writer
	stderr    io.Writer
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	var err error

	config := common.StructureConfig{
		ClusterBase: r.flag.Base,
		// Release charts are tagged without the `v` prefix.
		Release:           strings.TrimPrefix(r.flag.Release, "v"),
		ManagementCluster: r.flag.ManagementCluster,
		WorkloadCluster:   r.flag.Name,
		Organization:      r.flag.Organization,
		SkipMAPI:          r.flag.SkipMAPI,
		RepositoryName:    r.flag.RepositoryName,
	}

	localPath := ""
	localPathFlag := cmd.InheritedFlags().Lookup("local-path")
	if localPathFlag != nil {
		localPath = localPathFlag.Value.String()
	}

	if r.flag.Base != "" {
		baseDir := filepath.Join(localPath, r.flag.Base)
		exists, err := afero.DirExists(r.fs, baseDir)
		if err != nil {
			return microerror.Mask(err)
		}
		if !exists {
			return microerror.Maskf(invalidFlagsError, "--%s %q not found under %q", flagBase, r.flag.Base, localPath)
		}

		chart, err := clusterChartName(r.fs, baseDir)
		if err != nil {
			return microerror.Mask(err)
		}
		if chart == "" {
			_, _ = fmt.Fprintf(r.stderr, "Base %q holds no release chart, so --%s is not checked against the %s registry.\n", r.flag.Base, flagRelease, clustercommon.GSOCIRegistry)
		} else {
			err = r.checkReleaseChart(ctx, chart, config.Release)
			if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	if r.flag.ClusterUserConfig != "" {
		config.ClusterUserConfig, err = commonkey.ReadConfigMapYamlFromFile(
			r.fs,
			r.flag.ClusterUserConfig,
		)
		if err != nil {
			return microerror.Mask(err)
		}

		config.ClusterUserConfig = strings.TrimSpace(config.ClusterUserConfig)
	}

	creatorConfig, err := structure.NewWorkloadCluster(config)
	if err != nil {
		return microerror.Mask(err)
	}

	creatorConfig.Fs = r.fs
	creatorConfig.Stdout = r.stdout

	dryRunFlag := cmd.InheritedFlags().Lookup("dry-run")
	if dryRunFlag != nil {
		creatorConfig.DryRun, _ = strconv.ParseBool(dryRunFlag.Value.String())
	}

	creatorConfig.Path = localPath

	creator := creator.NewCreator(*creatorConfig)

	err = creator.Create()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// checkReleaseChart fails when the release chart can't deploy the release,
// because the chart isn't published for it or can't run as a HelmRelease.
func (r *runner) checkReleaseChart(ctx context.Context, chart, release string) error {
	version, err := semver.NewVersion(release)
	if err != nil {
		return microerror.Maskf(invalidFlagsError, "--%s %q is not a valid version: %s", flagRelease, release, err)
	}
	if chart == clusterprovider.ReleaseVsphereRepoName && version.LessThan(clustercommon.MinVsphereReleaseChartVersion) {
		return microerror.Maskf(invalidFlagsError, "--%s %s can't be deployed as a HelmRelease, the %s chart needs at least %s", flagRelease, release, chart, clustercommon.MinVsphereReleaseChartVersion)
	}

	ociClient := r.ociClient
	if ociClient == nil {
		ociClient, err = ociregistry.NewClient(ociregistry.ClientOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		defer ociClient.Close(ctx)
	}

	available, err := clustercommon.ReleaseChartAvailable(ctx, ociClient, chart, release)
	if err != nil {
		return microerror.Mask(err)
	}
	if !available {
		return microerror.Maskf(invalidFlagsError, "--%s %s is not published as a %s chart in the %s registry", flagRelease, release, chart, clustercommon.GSOCIRegistry)
	}

	return nil
}
