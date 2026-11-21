package org

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const (
	name  = "organization"
	alias = "org"

	shortDescription = "Adds a new Organization to your GitOps directory structure"
	longDescription  = `Adds a new Organization to your GitOps directory structure.

org \
--name <org_name> \
--management-cluster <mc_name> \
[--repository-name <gitops_repo_name>]

It writes the Organization in <org_name>.yaml, protected from deletion, and the
organization's namespace Kustomization in namespace.yaml. The organisation's
kustomization.yaml lists only these two for the management cluster's
Kustomization. The namespace Kustomization applies namespace/ into the
organization's namespace, org-<org_name>, once organization-operator has
created it: namespace/kustomization.yaml lists ../workload-clusters, so the
workload clusters' Kustomizations go into the organisation's namespace.

It respects the Giantswarm's GitOps repository structure recommendation:
https://github.com/giantswarm/gitops-template/blob/main/docs/reference/repo-structure.md.

Steps it implements:
https://github.com/giantswarm/gitops-template/blob/main/docs/tutorials/basic/getting-started.md#2-add-an-organisation

What it writes:
https://github.com/giantswarm/gitops-template/blob/main/docs/how-to/kubectl-gs.md#organisation`

	examples = `  # Add dummy Organization to dummy Management Cluster
  kubectl gs gitops add org \
  --name dummy \
  --management-cluster dummy`
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
		fs:     config.FileSystem,
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
