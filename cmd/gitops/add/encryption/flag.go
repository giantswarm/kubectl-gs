package encryption

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagFingerprint       = "fingerprint"
	flagGenerate          = "generate"
	flagManagementCluster = "management-cluster"
)

type flag struct {
	Fingerprint string
	Generate    bool
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Fingerprint, flagFingerprint, "", "Fingerprint of the key pair to configure repository with.")
	cmd.Flags().BoolVar(&f.Generate, flagGenerate, false, "Generate new key pair.")
}

func (f *flag) Validate() error {
	if !f.Generate && f.Fingerprint == "" {
		return microerror.Maskf(invalidFlagsError, "either one --%s or %s must be specified", flagGenerate, flagFingerprint)
	}

	return nil
}
