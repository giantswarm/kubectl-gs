package app

import (
	"fmt"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v2/internal/key"
)

const (
	flagProvider = "provider"
)

type flag struct {
	Provider string
}

func supportedProviders() []string {
	return []string{
		key.ProviderCAPA,
		key.ProviderGCP,
	}
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Provider, flagProvider, "", fmt.Sprintf("Installation infrastructure provider, supported values: %s", strings.Join(supportedProviders(), ", ")))
}

func (f *flag) Validate() error {
	isValidProvider := false
	for _, p := range supportedProviders() {
		if f.Provider == p {
			isValidProvider = true
			break
		}
	}
	if !isValidProvider {
		return microerror.Maskf(invalidFlagError, "--%s must be one of: %s", flagProvider, strings.Join(supportedProviders(), ", "))
	}

	return nil
}
