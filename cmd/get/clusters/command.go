package clusters

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v2/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v2/pkg/middleware"
	"github.com/giantswarm/kubectl-gs/v2/pkg/middleware/renewtoken"
)

const (
	name  = "clusters <cluster-name>"
	alias = "cluster"

	shortDescription = "Display one or many clusters"
	longDescription  = `Display one or many clusters

Output columns:

- NAME: Unique identifier of the cluster.
- AGE: How long ago was the cluster created.
- CONDITION: Latest condition reported for the cluster. Can be "CREATING", "CREATED", "UPDATING", "UPDATED", "DELETING".
- RELEASE: Workload cluster release used by the cluster.
- ORGANIZATION: Organization owning the cluster.
- DESCRIPTION: User friendly description for the cluster.`

	examples = `  # List all clusters you have access to
  kubectl gs get clusters

  # Get one specific cluster by its name
  kubectl gs get clusters f83ir`
)

type Config struct {
	Logger     micrologger.Logger
	FileSystem afero.Fs

	ConfigFlags *genericclioptions.RESTClientGetter

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
	if config.ConfigFlags == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.ConfigFlags must not be empty", config)
	}
	if config.Stderr == nil {
		config.Stderr = os.Stderr
	}
	if config.Stdout == nil {
		config.Stdout = os.Stdout
	}

	f := &flag{}

	r := &runner{
		commonConfig: &commonconfig.CommonConfig{
			ConfigFlags: config.ConfigFlags,
		},
		flag:   f,
		logger: config.Logger,
		fs:     config.FileSystem,

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
			renewtoken.Middleware(*config.ConfigFlags),
		),
	}

	f.Init(c)

	return c, nil
}
