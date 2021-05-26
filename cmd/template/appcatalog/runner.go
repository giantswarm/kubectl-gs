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
	var configMapYaml []byte
	var secretYaml []byte
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
		configmapCR, err := templateappcatalog.NewConfigMap(config, configMapData)
		if err != nil {
			return microerror.Mask(err)
		}

		configMapYaml, err = yaml.Marshal(configmapCR)
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
		secretCR, err := templateappcatalog.NewSecret(config, secretData)
		if err != nil {
			return microerror.Mask(err)
		}

		secretYaml, err = yaml.Marshal(secretCR)
		if err != nil {
			return microerror.Maskf(unmashalToMapFailedError, err.Error())
		}
	}

	type AppCatalogCROutput struct {
		AppCatalogCR string
		ConfigMap    string
		Secret       string
	}

	appCatalogCR, err := templateappcatalog.NewAppCatalogCR(config)
	if err != nil {
		return microerror.Mask(err)
	}

	appCatalogCRYaml, err := yaml.Marshal(appCatalogCR)
	if err != nil {
		return microerror.Mask(err)
	}

	appCatalogCROutput := AppCatalogCROutput{
		AppCatalogCR: string(appCatalogCRYaml),
		ConfigMap:    string(configMapYaml),
		Secret:       string(secretYaml),
	}

	t := template.Must(template.New("appCatalogCR").Parse(key.AppCatalogCRTemplate))

	err = t.Execute(os.Stdout, appCatalogCROutput)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
