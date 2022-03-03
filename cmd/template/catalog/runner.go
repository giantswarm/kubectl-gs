package catalog

import (
	"context"
	"io"
	"os"
	"text/template"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/internal/key"
	templatecatalog "github.com/giantswarm/kubectl-gs/pkg/template/catalog"
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

	catalogID, err := key.GenerateName(r.flag.AllowLongNames)
	if err != nil {
		return microerror.Mask(err)
	}

	config := templatecatalog.Config{
		Description: r.flag.Description,
		LogoURL:     r.flag.LogoURL,
		ID:          catalogID,
		Name:        r.flag.Name,
		Namespace:   r.flag.Namespace,
		URL:         r.flag.URL,
	}

	if r.flag.ConfigMap != "" {
		var configMapData string

		configMapData, err = key.ReadConfigMapYamlFromFile(afero.NewOsFs(), r.flag.ConfigMap)
		if err != nil {
			return microerror.Mask(err)
		}

		config.CatalogConfigMapName = key.GenerateAssetName(r.flag.Name, config.ID)
		configmapCR, err := templatecatalog.NewConfigMap(config, configMapData)
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
		secretCR, err := templatecatalog.NewSecret(config, secretData)
		if err != nil {
			return microerror.Mask(err)
		}

		secretYaml, err = yaml.Marshal(secretCR)
		if err != nil {
			return microerror.Maskf(unmashalToMapFailedError, err.Error())
		}
	}

	type CatalogCROutput struct {
		CatalogCR string
		ConfigMap string
		Secret    string
	}

	catalogCR, err := templatecatalog.NewCatalogCR(config)
	if err != nil {
		return microerror.Mask(err)
	}

	catalogCRYaml, err := yaml.Marshal(catalogCR)
	if err != nil {
		return microerror.Mask(err)
	}

	catalogCROutput := CatalogCROutput{
		CatalogCR: string(catalogCRYaml),
		ConfigMap: string(configMapYaml),
		Secret:    string(secretYaml),
	}

	t := template.Must(template.New("catalogCR").Parse(key.CatalogCRTemplate))

	err = t.Execute(os.Stdout, catalogCROutput)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
