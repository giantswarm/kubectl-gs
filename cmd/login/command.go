package login

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v4/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v4/pkg/middleware"
	"github.com/giantswarm/kubectl-gs/v4/pkg/middleware/renewtoken"
)

const (
	name = `login <arg1> <arg2> [flags]

Arguments <arg1> and <arg2> are optional and can take several forms.
No arguments means that the currently selected context will be used.

Use <arg1> alone for:

  - the URL of the cluster Kubernetes endpoint
  - the URL of the Giant Swarm web UI
  - A Giant Swarm management cluster with an existing context
  - a previously generated context name
  
Use <arg1> <arg2> for

  -  A Giant Swarm management cluster and a Giant Swarm workload cluster with an existing context
  `
	shortDescription = "Ensures an authenticated context for a Giant Swarm management or workload cluster"
	longDescription  = `Ensure an authenticated context for a Giant Swarm management or workload cluster

Use this command to set up a kubectl context to work with:
  (1) a management cluster, using OIDC authentication
  (2) a workload cluster, using OIDC authentication
  (3) a workload cluster, using client certificate auth. Not supported on kvm.

Note that (3) implies (1). When creating a workload cluster client certificate,
management cluster access will be set up as well, if that is not yet done.

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

  kubectl gs login gs-mymc

  kubectl gs login mymc

Workload cluster:

  kubectl gs login https://api.example.g8s.test.eu-west-1.aws.gigantic.io

  kubectl gs login gs-mymc-mywc

  kubectl gs login mymc mywc

Workload cluster client certificate:

  kubectl gs login mymc \
    --` + flagWCName + ` gir0y \
    --` + flagWCOrganization + ` acme \
    --` + flagWCCertGroups + ` admins \
    --` + flagWCCertTTL + ` 3h
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
