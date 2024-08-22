package wcluster

import (
	"context"
	"io"
	"strconv"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/structure/common"
	structure "github.com/giantswarm/kubectl-gs/v5/internal/gitops/structure/workload-cluster"
	commonkey "github.com/giantswarm/kubectl-gs/v5/internal/key"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
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

	config := common.StructureConfig{
		ClusterBase:       r.flag.Base,
		Release:           r.flag.Release,
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

	localPathFlag := cmd.InheritedFlags().Lookup("local-path")
	if localPathFlag != nil {
		creatorConfig.Path = localPathFlag.Value.String()
	}

	creator := creator.NewCreator(*creatorConfig)

	err = creator.Create()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
