package cluster

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
	name = "cluster --name <cluster-name> --namespace <cluster-namespace> --release-version <release-version> --scheduled-time <scheduled-time> --provider <provider>"

	shortDescription = "Schedule a cluster update."
	longDescription  = `Schedule a cluster update.

Updates given cluster with the provided values.

Options:
  --name <name>               		Name of the cluster to update.
  --namespace <namespace>     		Namespace of the cluster.
  --release-version <release-version>   Update the cluster to a release version. The release version must be higher than the current release version.
  --scheduled-time <scheduled-time>     Scheduled time when cluster should be updated. The value has to be in RFC822 Format and UTC time zone.
  --provider <provider> 		Name of the provider.`

	examples = `  # Display this help
kubectl gs update cluster --help

# Update cluster version
kubectl gs update cluster --name abcd1 --namespace my-org --release-version 16.1.0 --scheduled-time "30 Jan 22 12:00 UTC" --provider aws`
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

		stderr: config.Stderr,
		stdout: config.Stdout,
	}

	c := &cobra.Command{
		Use:     name,
		Short:   shortDescription,
		Long:    longDescription,
		Example: examples,
		Args:    cobra.ExactValidArgs(0),
		RunE:    r.Run,
		PreRunE: middleware.Compose(
			renewtoken.Middleware(config.K8sConfigAccess),
		),
	}

	f.Init(c)

	return c, nil
}
