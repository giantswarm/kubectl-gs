package catalogs

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/pkg/middleware"
	"github.com/giantswarm/kubectl-gs/pkg/middleware/renewtoken"
)

const (
	name  = "catalogs <catalog-name>"
	alias = "catalog"

	shortDescription = "Display app catalogs and available apps"
	longDescription  = `Display app catalogs and available apps

Output columns:

- NAME: Name of the catalog.
- NAMESPACE: Namespace of the catalog.
- URL: URL for the Helm chart repository.
- AGE: How long ago was the catalog created.
- DESCRIPTION: Helm chart description.

Getting a catalog by name will display the latest versions of the apps
in this catalog according to semantic versioning.

Output columns:

- CATALOG: Name of the catalog.
- APP NAME: Name of the app.
- APP VERSION: Upstream version of the app.
- VERSION: Latest version of the app.
- AGE: How long ago was the app release created.
- DESCRIPTION: Helm chart description.`

	examples = `  # List all public app catalogs
  kubectl gs get catalogs

  # List all available apps for a catalog
  kubectl gs get catalog giantswarm`
)

type Config struct {
	Logger     micrologger.Logger
	FileSystem afero.Fs

	CommonConfig *commonconfig.CommonConfig

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
	if config.CommonConfig == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CommonConfig must not be empty", config)
	}
	if config.Stderr == nil {
		config.Stderr = os.Stderr
	}
	if config.Stdout == nil {
		config.Stdout = os.Stdout
	}

	f := &flag{}

	r := &runner{
		commonConfig: config.CommonConfig,
		flag:         f,
		logger:       config.Logger,
		fs:           config.FileSystem,

		stderr: config.Stderr,
		stdout: config.Stdout,
	}

	c := &cobra.Command{
		Use:     name,
		Short:   shortDescription,
		Long:    longDescription,
		Example: examples,
		Aliases: []string{alias},
		Args:    cobra.MaximumNArgs(1),
		RunE:    r.Run,
		PreRunE: middleware.Compose(
			renewtoken.Middleware(config.CommonConfig.ToRawKubeConfigLoader().ConfigAccess()),
		),
	}

	f.Init(c)

	return c, nil
}
