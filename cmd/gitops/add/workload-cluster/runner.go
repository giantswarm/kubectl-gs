package wcluster

import (
	"context"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
	structure "github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/workload-cluster"
	commonkey "github.com/giantswarm/kubectl-gs/v6/internal/key"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
	fs     afero.Fs
	fetch  fetchFunc
	stdout io.Writer
	stderr io.Writer
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

	localPath := "."
	localPathFlag := cmd.InheritedFlags().Lookup("local-path")
	if localPathFlag != nil {
		localPath = localPathFlag.Value.String()
	}

	clusterAppVersion, err := r.clusterAppVersion(ctx, localPath)
	if err != nil {
		return microerror.Mask(err)
	}

	config := common.StructureConfig{
		ClusterBase:       r.flag.Base,
		Release:           r.flag.Release,
		ClusterAppVersion: clusterAppVersion,
		ManagementCluster: r.flag.ManagementCluster,
		WorkloadCluster:   r.flag.Name,
		Organization:      r.flag.Organization,
		SkipMAPI:          r.flag.SkipMAPI,
		RepositoryName:    r.flag.RepositoryName,
	}

	if r.flag.ClusterUserConfig != "" {
		config.ClusterUserConfig, err = commonkey.ReadConfigMapYamlFromFile(
			afero.NewOsFs(),
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

// clusterAppVersion returns the version of the cluster App. It is
// --cluster-app-version when set. Otherwise it is the version of the base's
// cluster chart that --release ships, from giantswarm/releases.
func (r *runner) clusterAppVersion(ctx context.Context, localPath string) (string, error) {
	if r.flag.Base == "" || r.flag.ClusterAppVersion != "" {
		return r.flag.ClusterAppVersion, nil
	}

	chartName, err := clusterChartName(r.fs, filepath.Join(localPath, r.flag.Base))
	if err != nil {
		return "", microerror.Mask(err)
	}

	version, err := releaseClusterAppVersion(ctx, r.fetch, chartName, r.flag.Release)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return version, nil
}
