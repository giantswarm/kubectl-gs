package mcluster

import (
	"context"
	"io"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

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
	mcConfig := structure.McConfig{
		Name:            r.flag.Name,
		RefreshInterval: r.flag.Interval.String(),
		RefreshTimeout:  r.flag.Timeout.String(),
		RepositoryName:  r.flag.RepositoryName,
		ServiceAccount:  r.flag.ServiceAccount,
	}
	mcDir, err := structure.NewManagementCluster(mcConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	var dryRun string
	dryRunFlag := cmd.InheritedFlags().Lookup("dry-run")
	if dryRunFlag != nil {
		dryRun = dryRunFlag.Value.String()
	}

	var localPath string
	localPathFlag := cmd.InheritedFlags().Lookup("local-path")
	if localPathFlag != nil {
		localPath = localPathFlag.Value.String()
	}

	if dryRun == "true" {
		mcDir.Print(localPath)
		return nil
	}

	err = mcDir.Write(localPath)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
