package encryption

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const (
	name  = "encryption"
	alias = "enc"

	shortDescription = "Adds a new GPG key pair to the SOPS repository configuration."
	longDescription  = `Adds a new GPG key pair to the SOPS repository configuration.

enc \
--generate
--management-cluster <mc_code_name>
[--organization <org_name>]
[--workload-cluster <wc_name>]
[--target <sub_path>]

A management cluster's key, and an organisation's, go into the master key
Secret, sops-gpg-master, in MC_NAME/secrets, which secrets/kustomization.yaml
protects from deletion. An organisation's key encrypts its namespace/secrets
unless --target says otherwise: the organisation's namespace Kustomization,
namespace.yaml, applies and decrypts what it keeps there.

A workload cluster's key goes into its own Secret, sops-gpg-<wc_name>, in the
organization's namespace/secrets and namespace, where the cluster's
Kustomizations run. namespace/secrets/kustomization.yaml protects it from
deletion. It is encrypted with the master key, which the organisation's
namespace Kustomization decrypts it with, so the management cluster needs its
master key first: .sops.yaml gets a rule for the workload clusters' keys,
ahead of the other rules. Both of the cluster's Kustomizations decrypt with
the key.

It respects the Giantswarm's GitOps repository structure recommendation:
https://github.com/giantswarm/gitops-template/blob/main/docs/reference/repo-structure.md.

Steps it implements:
https://github.com/giantswarm/gitops-template/blob/main/docs/how-to/sops.md#add-a-workload-cluster-key-gpg`

	examples = `  # Configure Management Cluster encryption with a new key pair.
  # Configures SOPS for handling the "management-clusters/demomc/secrets/*.enc.yaml" files. # master key
  kubectl gs gitops add encryption \
  --generate \
  --management-cluster demomc

  # Configure Management Cluster encryption with a new key pair.
  # Configures SOPS for handling the "management-clusters/demomc/mydir/*.enc.yaml" files.
  kubectl gs gitops add encryption \
  --generate \
  --management-cluster demomc \
  --target mydir/

  # Configure Organization encryption with a new key pair.
  # Configures SOPS for handling the "management-clusters/demomc/organizations/demoorg/namespace/secrets/*.enc.yaml" files.
  kubectl gs gitops add encryption \
  --generate \
  --management-cluster demomc \
  --organization demoorg

  # Configure Workload Cluster encryption with a new key pair.
  # Configures SOPS for the "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/*.enc.yaml" files. # entire Workload Cluster
  kubectl gs gitops add encryption \
  --generate \
  --management-cluster demomc \
  --organization demoorg \
  --workload-cluster demowc \
  --target /

  # Configure Workload Cluster encryption with a new key pair.
  # Configures SOPS for handling the "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/myapp/*.enc.yaml" files. # encryption for a single app
  kubectl gs gitops add encryption \
  --generate \
  --management-cluster demomc \
  --organization demoorg \
  --workload-cluster demowc \
  --target apps/myapp/`
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
