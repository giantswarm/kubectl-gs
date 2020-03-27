package appcatalog

import (
	"net/url"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagConfigMap   = "configmap"
	flagDescription = "description"
	flagName        = "name"
	flagSecret      = "secret"
	flagURL         = "url"
)

type flag struct {
	ConfigMap   string
	Description string
	Name        string
	Secret      string
	URL         string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.ConfigMap, flagConfigMap, "", "Path to a configmap file.")
	cmd.Flags().StringVar(&f.Description, flagDescription, "", "App Catalog description.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "App Catalog name.")
	cmd.Flags().StringVar(&f.Secret, flagSecret, "", "Path to a secret file.")
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
	if _, err := url.ParseRequestURI(f.URL); err != nil {
		return microerror.Maskf(invalidFlagError, "--%s must be a valid URL", flagURL)
	}

	return nil
}
