package catalog

import (
	"net/url"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagConfigMap       = "configmap"
	flagDescription     = "description"
	flagEnableLongNames = "enable-long-names"
	flagLogoURL         = "logo"
	flagName            = "name"
	flagNamespace       = "namespace"
	flagSecret          = "secret"
	flagURL             = "url"
)

type flag struct {
	ConfigMap       string
	Description     string
	EnableLongNames bool
	LogoURL         string
	Name            string
	Namespace       string
	Secret          string
	URL             string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.ConfigMap, flagConfigMap, "", "Path to a configmap file.")
	cmd.Flags().StringVar(&f.Description, flagDescription, "", "Catalog description.")
	cmd.Flags().BoolVar(&f.EnableLongNames, flagEnableLongNames, false, "Allow long names.")
	cmd.Flags().StringVar(&f.LogoURL, flagLogoURL, "", "Catalog logo URL.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Catalog name.")
	cmd.Flags().StringVar(&f.Namespace, flagNamespace, "", "Namespace where the catalog will be created.")
	cmd.Flags().StringVar(&f.Secret, flagSecret, "", "Path to a secret file.")
	cmd.Flags().StringVar(&f.URL, flagURL, "", "Catalog storage URL.")

	_ = cmd.Flags().MarkHidden(flagEnableLongNames)
}

func (f *flag) Validate() error {

	if f.Description == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagDescription)
	}
	if f.LogoURL == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagLogoURL)
	}
	if _, err := url.ParseRequestURI(f.LogoURL); err != nil {
		return microerror.Maskf(invalidFlagError, "--%s must be a valid URL", flagLogoURL)
	}
	if f.Name == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagName)
	}
	if f.Namespace == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagNamespace)
	}
	if f.URL == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagURL)
	}
	if _, err := url.ParseRequestURI(f.URL); err != nil {
		return microerror.Maskf(invalidFlagError, "--%s must be a valid URL", flagURL)
	}

	return nil
}
