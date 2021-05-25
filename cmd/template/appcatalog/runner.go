package appcatalog

import (
	"context"
	"io"
	"os"
	"text/template"

	"github.com/ghodss/yaml"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/kubectl-gs/internal/key"
	templateappcatalog "github.com/giantswarm/kubectl-gs/pkg/template/appcatalog"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
	stdout io.Writer
	stderr io.Writer
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	var catalogConfigMap *corev1.ConfigMap
	var catalogSecret *corev1.Secret
	var catalogConfigMapYaml []byte
	var catalogSecretYaml []byte
	var err error

	config := templateappcatalog.Config{
		Description: r.flag.Description,
		LogoURL:     r.flag.LogoURL,
		ID:          key.GenerateID(),
		Name:        r.flag.Name,
		Namespace:   metav1.NamespaceDefault,
		URL:         r.flag.URL,
	}

	if r.flag.ConfigMap != "" {
		var configMapData string

		configMapData, err = key.ReadConfigMapYamlFromFile(afero.NewOsFs(), r.flag.ConfigMap)
		if err != nil {
			return microerror.Mask(err)
		}

		config.CatalogConfigMapName = key.GenerateAssetName(r.flag.Name, config.ID)
		catalogConfigMap, err = templateappcatalog.NewConfigMap(config, configMapData)
		if err != nil {
			return microerror.Mask(err)
		}

		catalogConfigMapYaml, err = yaml.Marshal(catalogConfigMap)
		if err != nil {
			return microerror.Maskf(unmashalToMapFailedError, err.Error())
		}
	}

	if r.flag.Secret != "" {
		var secretData []byte

		secretData, err = key.ReadSecretYamlFromFile(afero.NewOsFs(), r.flag.Secret)
		if err != nil {
			return microerror.Mask(err)
		}

		config.CatalogSecretName = key.GenerateAssetName(r.flag.Name, config.ID)
		catalogSecret, err = templateappcatalog.NewSecret(config, secretData)
		if err != nil {
			return microerror.Mask(err)
		}

		catalogSecretYaml, err = yaml.Marshal(catalogSecret)
		if err != nil {
			return microerror.Maskf(unmashalToMapFailedError, err.Error())
		}
	}

	appCatalogCR, err := templateappcatalog.NewAppCatalogCR(config)
	if err != nil {
		return microerror.Mask(err)
	}

	appCatalogCRYaml, err := yaml.Marshal(appCatalogCR)
	if err != nil {
		return microerror.Mask(err)
	}

	type AppCatalogCROutput struct {
		AppCatalogCR     string
		CatalogConfigMap string
		CatalogSecret    string
	}

	appCatalogCROutput := AppCatalogCROutput{
		AppCatalogCR:     string(appCatalogCRYaml),
		CatalogConfigMap: string(catalogConfigMapYaml),
		CatalogSecret:    string(catalogSecretYaml),
	}

	t := template.Must(template.New("appCatalogCR").Parse(key.AppCatalogCRTemplate))

	err = t.Execute(os.Stdout, appCatalogCROutput)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
