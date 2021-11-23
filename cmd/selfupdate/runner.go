package selfupdate

import (
	"context"
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

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

		updaterService, err = selfupdate.New(config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	latestVersion, err := updaterService.GetLatest()
	if selfupdate.IsHasNewVersion(err) {
		if r.flag.DryRun {
			_, _ = color.New(color.Bold, color.FgYellow).Fprintln(r.stdout, "There's a new version available!")
			fmt.Fprintln(r.stdout, "Please update by running \"kubectl gs selfupdate\".")

			return nil
		}

		fmt.Fprintf(r.stdout, "Update to %s has been started.\n", latestVersion)
		fmt.Fprintln(r.stdout, "Fetching latest built binary...")

		err = updaterService.InstallLatest()
		if err != nil {
			return microerror.Mask(err)
		}

		_, _ = color.New(color.Bold, color.FgGreen).Fprintln(r.stdout, "Updated successfully.")

		return nil
	} else if selfupdate.IsVersionNotFound(err) {
		return microerror.Maskf(updateCheckFailedError, "Checking for the latest version failed or your platform is unsupported.")
	} else if err != nil {
		return microerror.Mask(err)
	}

	fmt.Fprintln(r.stdout, "You are already using the latest version.")

	return nil
}
