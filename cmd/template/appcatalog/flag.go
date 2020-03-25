package appcatalog

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagDescription = "description"
	flagName        = "name"
	flagURL         = "url"
)

type flag struct {
	Description string
	Name        string
	URL         string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Description, flagDescription, "", "App Catalog description.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "App Catalog name.")
	cmd.Flags().StringVar(&f.URL, flagURL, "", "Catalog storage URL.")
}

func (f *flag) Validate() error {

	if f.Description == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagDescription)
	}
	if f.Name == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagName)
	}
	if f.URL == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagURL)
	}

	return nil
}
