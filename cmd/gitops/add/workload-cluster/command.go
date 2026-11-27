package wcluster

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const (
	name  = "workload-cluster"
	alias = "wc"

	shortDescription = "Adds a new workload cluster to your GitOps directory structure"
	longDescription  = `Adds a new workload cluster to your GitOps directory structure.

workload-cluster \
[--base <path_to_base>] \
[--release <version>] \
[--cluster-user-config <path_to_values.yaml>] \
--name <wc_id> \
--management-cluster <mc_code_name> \
--organization <org_name> \
--repository-name <gitops_repo_name> \

It writes the cluster's directory, workload-clusters/<wc_id>, and the Flux
Kustomizations that apply it, next to it. The organisation's namespace
Kustomization applies them into the organisation's namespace, where they run
as its automation ServiceAccount, and they depend on it. The organisation
needs its namespace.yaml first, which
` + "`kubectl gs gitops add organization`" + ` writes. The cluster's
kustomization.yaml lists what they apply, and never out-of-band/, which goes
straight into the workload cluster.

With --base, the cluster's definition goes into cluster/, and the cluster gets
two Kustomizations: <wc_id>.yaml applies cluster/, and <wc_id>-apps.yaml applies
the apps in apps/ once the cluster is ready. Without --base, a single
Kustomization, <wc_id>.yaml, applies the apps. Running the command again with
--base adds the definition to that single Kustomization.

A workload cluster still laid out as before structure version 3 is refused,
as Flux would not apply what the command writes there. Run
` + "`kubectl gs gitops upgrade`" + ` first.

It respects the Giantswarm's GitOps repository structure recommendation:
https://github.com/giantswarm/gitops-template/blob/main/docs/reference/repo-structure.md#flux-kustomizations.

Steps it implements:
https://github.com/giantswarm/gitops-template/blob/main/docs/tutorials/basic/getting-started.md#3-create-a-workload-cluster

What it writes:
https://github.com/giantswarm/gitops-template/blob/main/docs/how-to/kubectl-gs.md#workload-cluster

With --base, the base deploys the cluster as a Flux HelmRelease of the
release-<provider> chart, and --release sets the chart version through the
${release} variable. The release has to be published as that chart in the
gsoci.azurecr.io registry. A base deploying an App CR, itself or through the
base below an overlay, is refused.`

	examples = `  # Add dummy workload cluster structure, without cluster definition
  kubectl gs gitops add wc \
  --name dummy \
  --management-cluster mymc \
  --organization myorg \
  --repository-name gitops-demo

  # Add dummy workload cluster structure with definition from base
  kubectl gs gitops add wc \
  --name dummy \
  --management-cluster mymc \
  --organization myorg \
  --repository-name gitops-demo \
  --base bases/clusters/capa \
  --release 36.1.1

  # Add dummy workload cluster structure with definition from base and extra configuration
  kubectl gs gitops add wc \
  --name dummy \
  --management-cluster mymc \
  --organization myorg \
  --repository-name gitops-demo \
  --base bases/clusters/capa \
  --release 36.1.1 \
  --cluster-user-config /tmp/cluster_user_config.yaml`
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
		RunE:    r.Run,
	}

	f.Init(c)

	return c, nil
}
