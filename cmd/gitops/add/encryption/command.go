package encryption

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
	name  = "encryption"
	alias = "enc"

	shortDescription = "Adds a new GPG key pair to the SOPS repository configuration."
	longDescription  = `Adds a new GPG key pair to the SOPS repository configuration.

enc \
--fingerprint <fingerprint> | --generate
--management-cluster <mc_code_name>
[--organization <org_name>]
[--workload-cluster <wc_name>]
[--target <sub_path>]

It respects the Giantswarm's GitOps repository structure recommendation:
https://github.com/giantswarm/gitops-template/blob/main/docs/repo_structure.md.

Steps it implements:
https://github.com/giantswarm/gitops-template/blob/main/docs/add_wc_structure.md#create-flux-gpg-regular-key-pair-optional-step`

	examples = `  # Configure Management Cluster encryption with an existing key pair.
  # Configures SOPS for handling the "management-clusters/demomc/secrets/*.enc.yaml" files. # master key
  kubectl gs gitops add encryption \
  --fingerprint 123456789ABCDEF123456789ABCDEF123456789A \
  --management-cluster demomc

  # Configure Management Cluster encryption with a new key pair.
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
  # Configures SOPS for handling the "management-clusters/demomc/organizations/demoorg/secrets/*.enc.yaml" files.
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
