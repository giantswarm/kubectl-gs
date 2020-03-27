package appcatalog

import (
	"context"
	"io"
	"os"
	"text/template"

	"github.com/ghodss/yaml"
	"github.com/giantswarm/kubectl-gs/internal/key"
	appcatalog "github.com/giantswarm/kubectl-gs/pkg/template/appcatalog"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
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
	config := appcatalog.Config{
		Description: r.flag.Description,
		ID:          key.GenerateID(),
		Name:        r.flag.Name,
		URL:         r.flag.URL,
	}

	appCatalogCR, err := appcatalog.NewAppCatalogCR(config)

	appCatalogCRYaml, err := yaml.Marshal(appCatalogCR)
	if err != nil {
		return microerror.Mask(err)
	}

	var configMapData map[string]string
	if r.flag.ConfigMap != "" {
		configMapData, err = readValuesFromFile(afero.NewOsFs(), r.flag.ConfigMap)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	configmapCR, err := appcatalog.NewConfigmapCR(config, configMapData)

	configmapCRYaml, err := yaml.Marshal(configmapCR)
	if err != nil {
		return microerror.Maskf(unmashalToMapFailedError, err.Error())
	}

	secretCR, err := appcatalog.NewSecretCR(config)

	secretCRYaml, err := yaml.Marshal(secretCR)
	if err != nil {
		return microerror.Maskf(unmashalToMapFailedError, err.Error())
	}

	type AppCatalogCROutput struct {
		AppCatalogCR string
		ConfigmapCR  string
		SecretCR     string
	}

	appCatalogCROutput := AppCatalogCROutput{
		AppCatalogCR: string(appCatalogCRYaml),
		ConfigmapCR:  string(configmapCRYaml),
		SecretCR:     string(secretCRYaml),
	}

	t := template.Must(template.New("appCatalogCR").Parse(key.AppCatalogCRTemplate))

	err = t.Execute(os.Stdout, appCatalogCROutput)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// readValuesFromFile reads a configmap from a YAML file.
func readValuesFromFile(fs afero.Fs, path string) (map[string]string, error) {
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	rawMap := map[string]interface{}{}
	err = yaml.Unmarshal(data, rawMap)
	if err != nil {
		return nil, microerror.Maskf(unmashalToMapFailedError, err.Error())
	}

	var configMapData map[string]string
	configMapData["values"] = string(data)

	return configMapData, nil

}
