package catalog

import (
	"fmt"
	"net/url"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagConfigMap       = "configmap"
	flagDescription     = "description"
	flagLogoURL         = "logo"
	flagName            = "name"
	flagNamespace       = "namespace"
	flagTargetNamespace = "target-namespace"
	flagSecret          = "secret"
	flagURL             = "url"
	flagURLType         = "type"
	flagVisibility      = "visibility"
)

type flag struct {
	ConfigMap       string
	Description     string
	LogoURL         string
	Name            string
	Namespace       string
	TargetNamespace string
	Secret          string
	URLs            []string
	URLTypes        []string
	Visibility      string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.ConfigMap, flagConfigMap, "", "Path to a configmap file.")
	cmd.Flags().StringVar(&f.Description, flagDescription, "", "Catalog description.")
	cmd.Flags().StringVar(&f.LogoURL, flagLogoURL, "", "Catalog logo URL.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Catalog name.")
	cmd.Flags().StringVar(&f.Namespace, flagNamespace, "", fmt.Sprintf("Namespace where the catalog will be created. Deprecated, use %s instead", flagTargetNamespace))
	cmd.Flags().StringVar(&f.TargetNamespace, flagTargetNamespace, "", "Namespace where the catalog will be created.")
	cmd.Flags().StringVar(&f.Secret, flagSecret, "", "Path to a secret file.")
	cmd.Flags().StringArrayVar(&f.URLs, flagURL, []string{}, "Catalog storage URL.")
	cmd.Flags().StringArrayVar(&f.URLTypes, flagURLType, []string{"helm"}, "Type of catalog storage.")
	cmd.Flags().StringVar(&f.Visibility, flagVisibility, "public", "Visibility label for whether catalog appears in the web UI.")

	_ = cmd.Flags().MarkDeprecated(flagNamespace, fmt.Sprintf("use --%s instead.", flagTargetNamespace))
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
	if f.Namespace == "" && f.TargetNamespace == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagTargetNamespace)
	}
	if len(f.URLs) == 0 {
		return microerror.Maskf(invalidFlagError, "at least one --%s must be defined", flagURL)
	}
	for _, u := range f.URLs {
		if _, err := url.ParseRequestURI(u); err != nil {
			return microerror.Maskf(invalidFlagError, "--%s must be a valid URL: %#q", flagURL, u)
		}
	}
	if len(f.URLTypes) == 0 {
		return microerror.Maskf(invalidFlagError, "at least one --%s must be defined", flagURLType)
	}
	if len(f.URLs) != len(f.URLTypes) {
		return microerror.Maskf(invalidFlagError,
			"number of --%s (%d) and --%s (%d) has to match",
			flagURL, len(f.URLs), flagURLType, len(f.URLTypes),
		)
	}

	return nil
}
