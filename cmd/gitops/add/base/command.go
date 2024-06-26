package app

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v3/pkg/commonconfig"
)

const (
	name = "base"

	shortDescription = "Adds a new Cluster base template to your GitOps directory structure"
	longDescription  = `Adds a new Cluster base template to your GitOps directory structure.

base \
--provider <provider>

It respects the Giantswarm's GitOps repository structure recommendation:
https://github.com/giantswarm/gitops-template/blob/main/docs/repo_structure.md.`

	examples = `  # Add a CAPA base to your GitOps directory structure
  kubectl gs gitops add base --provider capa

  # Add a CAPG base to your GitOps directory structure
  kubectl gs gitops add base --provider gcp

  # Add a CAPO base to your GitOps directory structure
  kubectl gs gitops add base --provider openstack`
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
		stderr: config.Stderr,
		stdout: config.Stdout,
	}

	c := &cobra.Command{
		Use:     name,
		Short:   shortDescription,
		Long:    longDescription,
		Example: examples,
		RunE:    r.Run,
	}

	f.Init(c)

	return c, nil
}
