package login

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v6/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v6/pkg/middleware"
	"github.com/giantswarm/kubectl-gs/v6/pkg/middleware/renewtoken"
)

const (
	name = `login <arg1> <arg2> [flags]

Arguments <arg1> and <arg2> are optional and can take several forms.

Use <arg1> alone for:

  - the URL of the cluster Kubernetes API endpoint
  - the URL of the Giant Swarm web UI
  - a Giant Swarm management cluster with an existing context
  - a previously generated context name

Use <arg1> <arg2> for

  -  A Giant Swarm management cluster and a Giant Swarm workload cluster with an existing context
  `
	shortDescription = "Ensures an authenticated context for a Giant Swarm management or workload cluster"
	longDescription  = `Ensure an authenticated context for a Giant Swarm management or workload cluster

Use this command to set up a kubectl context to work with:
  (1) a management cluster (MC), using OIDC authentication via Dex
  (2) a workload cluster (WC), using OIDC authentication via Dex
  (3) a workload cluster (WC), using client certificate authentication
  (4) a workload cluster (WC), using AWS IAM authentication (EKS only)
  (5) a workload cluster (WC), using direct OIDC authentication (Kubernetes
      structured authentication)

The appropriate workload-cluster mode (2, 3, 4 or 5) is selected automatically
based on the cluster's configuration. (3) implies (1): when creating a workload
cluster client certificate, management cluster access will be set up as well,
if that is not yet done.

For (5), the OIDC issuer URL, client ID and API server CA are auto-discovered
from the management cluster. The flags --` + flagWCOIDCIssuer + `, --` + flagWCOIDCClientID + ` and
--` + flagWCOIDCCAFile + ` can be used to override the discovered values.

Security notes:

Please be aware that the recommended way to authenticate users for
workload clusters is OIDC. Client certificates should only be a
temporary replacement.

When creating client certificates, we recommend to use short
expiration periods (--` + flagWCCertTTL + `) and specific group names
(--` + flagWCCertGroups + `).`
	examples = `
Management cluster:

  kubectl gs login https://api.g8s.test.eu-west-1.aws.gigantic.io

  kubectl gs login gs-mymc # "gs-mymc" is the context name for the management cluster

  kubectl gs login mymc

Workload cluster (re-using an existing context):

  kubectl gs login https://api.example.g8s.test.eu-west-1.aws.gigantic.io

  kubectl gs login gs-mymc-mywc

  kubectl gs login mymc mywc

Workload cluster (the appropriate authentication method - direct OIDC,
client certificate, or AWS IAM - is selected automatically based on the
cluster's configuration):

  kubectl gs login mymc \
    --` + flagWCName + ` mywc \
    --` + flagWCOrganization + ` acme

Workload cluster, forcing client certificate creation parameters:

  kubectl gs login mymc \
    --` + flagWCName + ` mywc \
    --` + flagWCOrganization + ` acme \
    --` + flagWCCertGroups + ` admins \
    --` + flagWCCertTTL + ` 3h

Workload cluster direct OIDC with explicit overrides (skips management
cluster lookup of issuer URL, client ID and API server CA):

  kubectl gs login mymc \
    --` + flagWCName + ` mywc \
    --` + flagWCOrganization + ` acme \
    --` + flagWCOIDCIssuer + ` https://login.example.com/tenant-id \
    --` + flagWCOIDCClientID + ` my-client-id \
    --` + flagWCOIDCCAFile + ` /path/to/ca.crt
`
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
		flag:   f,
		logger: config.Logger,
		fs:     config.FileSystem,

		commonConfig: &commonconfig.CommonConfig{
			ConfigFlags: config.ConfigFlags,
		},
		stderr: config.Stderr,
		stdout: config.Stdout,
	}

	c := &cobra.Command{
		Use:     name,
		Short:   shortDescription,
		Long:    longDescription,
		Example: examples,
		RunE:    r.Run,
		PreRunE: middleware.Compose(
			renewtoken.Middleware(*config.ConfigFlags),
		),
	}

	f.Init(c)

	return c, nil
}
