package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/project"
	"github.com/giantswarm/kubectl-gs/pkg/selfupdate"
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

func (r *runner) PersistentPostRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.persistentPostRun(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	err := cmd.Help()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) persistentPostRun(ctx context.Context, cmd *cobra.Command, args []string) error {
	// Let's not run this before the `selfupdate` command,
	// to not prevent being able to update or check for new versions.
	if cmd.Name() == "selfupdate" {
		return nil
	}

	if r.flag.disableVersionCheck {
		// User wants to risk their life and use an older version.
		// Not my problem anymore.
		return nil
	}

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

	latestVersion, err := updaterService.GetLatest()
	if selfupdate.IsHasNewVersion(err) {
		if key.IsTTY() {
			fmt.Fprintf(r.stderr, "\n")
		}

		_, _ = color.New(color.Bold, color.FgYellow).Fprintf(r.stderr, "You are running an outdated version of %s. The latest version is %s.\n", project.Name(), latestVersion)
		fmt.Fprintln(r.stderr, "Please update by running \"kubectl gs selfupdate\".")

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
