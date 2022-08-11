package wcluster

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	name  = "workload-cluster"
	alias = "wc"

	shortDescription = "Adds a new Workload Cluster to your GitOps directory structure"
	longDescription  = `Adds a new Workload Cluster to your GitOps directory structure.

workload-cluster \
[--base <path_to_base] \
[--cluster-release <cluster_version>] \
[--cluster-user-config <path_to_values.yaml>] \
[--default-apps-release <default_apps_version>] \
[--default-apps-user-config <path_to_values.yaml>] \
--name <wc_id> \
--management-cluster <mc_code_name> \
--organization <org_name> \
--repository <gitops_repo_name> \

It respects the Giantswarm's GitOps repository structure recommendation:
https://github.com/giantswarm/gitops-template/blob/main/docs/repo_structure.md.

Steps it implements:
https://github.com/giantswarm/gitops-template/blob/main/docs/add_wc.md`

	examples = `  # Add dummy Workload Cluster structure, without cluster definition
  kubectl gs gitops add wc \
  --name dummy \
  --management-cluster mymc \
  --organization myorg \
  --repository-name gitops-demo

  # Add dummy Workload Cluster structure with definition from base
  kubectl gs gitops add wc \
  --name dummy \
  --management-cluster mymc \
  --organization myorg \
  --repository-name gitops-demo \
  --base bases/clusters/openstack \
  --cluster-release 0.13.0 \
  --default-apps-release 0.6.0

  # Add dummy Workload Cluster structure with definition from base and extra configuration
  kubectl gs gitops add wc \
  --name dummy \
  --management-cluster mymc \
  --organization myorg \
  --repository-name gitops-demo \
  --base bases/clusters/openstack \
  --cluster-release 0.13.0 \
  --default-apps-release 0.6.0 \
  --cluster-user-config /tmp/cluster_user_config.yaml \
  --default-apps-user-config /tmp/default_apps_user_config.yaml`
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
