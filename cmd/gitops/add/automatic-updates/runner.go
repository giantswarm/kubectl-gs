package autoupdate

import (
	"context"
	//"fmt"
	"io"
	"strconv"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/structure/common"
	structure "github.com/giantswarm/kubectl-gs/v5/internal/gitops/structure/updates"
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
		AppName:              r.flag.App,
		AppVersionRepository: r.flag.VersionRepository,
		ManagementCluster:    r.flag.ManagementCluster,
		Organization:         r.flag.Organization,
		RepositoryName:       r.flag.Repository,
		SkipMAPI:             r.flag.SkipMAPI,
		WorkloadCluster:      r.flag.WorkloadCluster,
	}

	creatorConfig, err := structure.NewAutomaticUpdate(config)
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
