package nodepools

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giantswarm/kubectl-gs/pkg/middleware"
	"github.com/giantswarm/kubectl-gs/pkg/middleware/renewtoken"
)

const (
	name  = "nodepools [nodepool-name]"
	alias = "nodepool"

	shortDescription = "Display one or many node pools"
	longDescription  = `Display one or many node pools

Output columns:

- NAME: Unique identifier of the node pool.
- CLUSTER NAME: Unique identifier of the cluster that the node pool belongs to.
- AGE: How long ago was the node pool created.
- CONDITION: Latest condition reported for the node pool.
- NODES MIN/MAX: Node pool autoscaler settings (if supported).
- NODES DESIRED: The total number of nodes that the node pool should have.
- NODES READY: The number of nodes in the node pool that are actually ready.
- DESCRIPTION: User friendly description for the node pool.`

	examples = `  # List all node pools you have access to
  kubectl gs get nodepools

  # Get one specific nodepool by its name
  kubectl gs get nodepool 3f01a`
)

type Config struct {
	Logger     micrologger.Logger
	FileSystem afero.Fs

	K8sConfigAccess clientcmd.ConfigAccess

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
	if config.K8sConfigAccess == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sConfigAccess must not be empty", config)
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
			renewtoken.Middleware(config.K8sConfigAccess),
		),
	}

	f.Init(c)

	return c, nil
}
