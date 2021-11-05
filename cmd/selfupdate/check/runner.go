package check

import (
	"context"
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/selfupdate"

	"github.com/giantswarm/kubectl-gs/pkg/project"
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

	var updaterService *selfupdate.Updater
	{
		config := selfupdate.Config{
			CurrentVersion: project.Version(),
			RepositoryURL:  project.Source(),
		}

		config.CacheDir, err = key.GetCacheDir()
		if err != nil {
			return microerror.Mask(err)
		}

		updaterService, err = selfupdate.New(config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	_, err = updaterService.GetLatest()
	if selfupdate.IsHasNewVersion(err) {
		color.New(color.Bold, color.FgYellow).Fprintln(r.stdout, "There's a new version available!")
		fmt.Fprintln(r.stdout, "Please update by running \"kubectl gs selfupdate execute\".")

		return nil
	} else if selfupdate.IsVersionNotFound(err) {
		return microerror.Maskf(updateCheckFailedError, "Checking for the latest version failed or your platform is unsupported.")
	} else if err != nil {
		return microerror.Mask(err)
	}

	color.New(color.Bold, color.FgGreen).Fprintln(r.stdout, "You are already using the latest version.")
	fmt.Fprintln(r.stdout, "There are no newer versions available.")

	return nil
}
