package mcluster

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem"
	"github.com/giantswarm/kubectl-gs/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/internal/gitops/structure"
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
	config := structure.McConfig{
		Name:            r.flag.Name,
		RefreshInterval: r.flag.Interval.String(),
		RefreshTimeout:  r.flag.Timeout.String(),
		RepositoryName:  r.flag.RepositoryName,
		ServiceAccount:  r.flag.ServiceAccount,
	}
	dir, err := structure.NewManagementCluster(config)
	if err != nil {
		return microerror.Mask(err)
	}

	creatorConfig := filesystem.CreatorConfig{
		Directory: dir,
		Stdout:    r.stdout,
	}

	dryRunFlag := cmd.InheritedFlags().Lookup("dry-run")
	if dryRunFlag != nil {
		creatorConfig.DryRun, _ = strconv.ParseBool(dryRunFlag.Value.String())
	}

	localPathFlag := cmd.InheritedFlags().Lookup("local-path")
	if localPathFlag != nil {
		creatorConfig.Path = fmt.Sprintf("%s/", localPathFlag.Value.String())
	}
	creatorConfig.Path += key.DirectoryManagementClusters

	creator := filesystem.NewCreator(creatorConfig)
	creator.Create()

	return nil
}
