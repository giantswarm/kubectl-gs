package login

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
	name = `login <management-cluster> [flags]

The <management-cluster> argument can take several forms:

  - the URL of the management cluster Kubernetes endpoint
  - the URL of the Giant Swarm web UI
  - a previously generated context name, with or without the prefix "gs-"`
	shortDescription = "Ensures an authenticated context for a Giant Swarm management or workload cluster"
	longDescription  = `Ensure an authenticated context for a Giant Swarm management or workload cluster

Use this command to set up a kubectl context to work with:
  (1) a management cluster, using OIDC authentication
  (2) a workload cluster, using client certificate auth

Note that (2) implies (1). When setting up workload cluster access,
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

  kubectl gs login https://g8s.test.eu-west-1.aws.gigantic.io

  kubectl gs login gs-mymc

  kubectl gs login mymc

Workload cluster:

  kubectl gs login mymc \
    --workload-cluster gir0y \
    --organization acme \
    --certificate-o admins \
    --certificate-ttl 3h
`
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
		return nil, microerror.Maskf(invalidConfigError, "%T.ConfigAccess must not be empty", config)
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

		k8sConfigAccess: config.K8sConfigAccess,

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
			renewtoken.Middleware(config.K8sConfigAccess),
		),
	}

	f.Init(c)

	return c, nil
}
