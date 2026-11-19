package app

import (
	"fmt"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v6/internal/key"
)

const (
	flagAzureSubscriptionID = "azure-subscription-id"
	flagPreventDeletion     = "prevent-deletion"
	flagProvider            = "provider"
	flagRegion              = "region"
)

type flag struct {
	Provider            string
	Region              string
	AzureSubscriptionID string
	PreventDeletion     bool
}

func supportedProviders() []string {
	return []string{
		key.ProviderCAPA,
		key.ProviderCAPZ,
		key.ProviderVSphere,
	}
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Provider, flagProvider, "", fmt.Sprintf("Installation infrastructure provider, supported values: %s", strings.Join(supportedProviders(), ", ")))
	cmd.Flags().StringVar(&f.Region, flagRegion, "", "AWS/Azure region where cluster will be created")
	cmd.Flags().BoolVar(&f.PreventDeletion, flagPreventDeletion, true, "Prevent clusters built on the base from getting deleted, set to false to allow it (see https://docs.giantswarm.io/tutorials/fleet-management/deletion-prevention/)")

	// Azure only
	cmd.Flags().StringVar(&f.AzureSubscriptionID, flagAzureSubscriptionID, "", "Azure subscription ID")
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
