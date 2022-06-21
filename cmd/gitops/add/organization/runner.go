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
	orgConfig := structure.OrgConfig{
		Name: r.flag.Name,
	}
	orgDir, err := structure.NewOrganization(orgConfig)
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
		orgDir.Print(localPath)
		return nil
	}

	err = orgDir.Write(localPath)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
