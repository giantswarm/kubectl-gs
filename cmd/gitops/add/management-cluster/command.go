package mcluster

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const (
	name  = "management-cluster"
	alias = "mc"

	shortDescription = "Adds a new Management Cluster to your GitOps directory structure"
	longDescription  = `Adds a new Management Cluster to your GitOps directory structure.

mc \
--name <mc_code_name> \
--repository-name <gitops_repo_name> \
[--gen-master-key]

It respects the Giantswarm's GitOps repository structure recommendation:
https://github.com/giantswarm/gitops-template/blob/main/docs/repo_structure.md.

Steps it implements:
https://github.com/giantswarm/gitops-template/blob/main/docs/add_mc.md`

	examples = `  # Add dummy Management Cluster
  kubectl gs gitops add mc \
  --name dummy \
  --repository-name gitops-demo

  # Add dummy Management Cluster with master GPG key
  kubectl gs gitops add mc \
  --name dummy \
  --repository-name gitops-demo
  --gen-master-key`
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
