package autoupdate

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const (
	name  = "automatic-update"
	alias = "update"

	shortDescription = "Enable automatic updates for an app."
	longDescription  = `Enable automatic updates for an app.

automatic-update \
--app <app_to_configure_updates_for> \
[--version-range <semver_range>] \
--management-cluster <mc_code_name> \
--organization <org_name> \
--workload-cluster <wc_name>

It sets a semver range on the app's OCIRepository, replacing the version it
pins. Flux then pulls the newest chart version in the range, and the
HelmRelease upgrades the app to it. Apps deployed with an App CR, and apps
from a base, are refused.

It respects the Giantswarm's GitOps repository structure recommendation:
https://github.com/giantswarm/gitops-template/blob/main/docs/reference/repo-structure.md.

Steps it implements:
https://github.com/giantswarm/gitops-template/blob/main/docs/guides/automatic-updates.md`

	examples = `  # Enable automatic updates for hello-world app
    kubectl gs gitops add automatic-update \
    --app hello-world \
    --management-cluster demomc \
    --organization demoorg \
    --workload-cluster demowc

  # Only follow patch releases of hello-world 0.3
    kubectl gs gitops add automatic-update \
    --app hello-world \
    --version-range '0.3.x' \
    --management-cluster demomc \
    --organization demoorg \
    --workload-cluster demowc`
)

type Config struct {
	Logger     micrologger.Logger
	FileSystem afero.Fs

	Stderr io.Writer
	Stdout io.Writer
}

func New(config Config) (*cobra.Command, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.FileSystem == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.FileSystem must not be empty", config)
	}
	if config.Stderr == nil {
		config.Stderr = os.Stderr
	}
	if config.Stdout == nil {
		config.Stdout = os.Stdout
	}

	f := &flag{}

	r := &runner{
		flag:   f,
		logger: config.Logger,
		stderr: config.Stderr,
		stdout: config.Stdout,
	}

	c := &cobra.Command{
		Use:     name,
		Short:   shortDescription,
		Long:    longDescription,
		Example: examples,
		Aliases: []string{alias},
		RunE:    r.Run,
	}

	f.Init(c)

	return c, nil
}
