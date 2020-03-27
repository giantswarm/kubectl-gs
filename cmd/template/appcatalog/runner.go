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

	var configMapData string
	if r.flag.ConfigMap != "" {
		configMapData, err = readConfigMapYamlFromFile(afero.NewOsFs(), r.flag.ConfigMap)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	configmapCR, err := appcatalog.NewConfigmapCR(config, configMapData)

	configmapCRYaml, err := yaml.Marshal(configmapCR)
	if err != nil {
		return microerror.Maskf(unmashalToMapFailedError, err.Error())
	}

	var secretData map[string][]byte
	if r.flag.Secret != "" {
		secretData, err = readSecretYamlFromFile(afero.NewOsFs(), r.flag.Secret)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	secretCR, err := appcatalog.NewSecretCR(config, secretData)

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

// readConfigMapFromFile reads a configmap from a YAML file.
func readConfigMapYamlFromFile(fs afero.Fs, path string) (string, error) {
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return "", microerror.Mask(err)
	}

	rawMap := map[string]string{}
	err = yaml.Unmarshal(data, &rawMap)
	if err != nil {
		return "", microerror.Maskf(unmashalToMapFailedError, err.Error())
	}

	return string(data), nil
}

// readSecretFromFile reads a configmap from a YAML file.
func readSecretYamlFromFile(fs afero.Fs, path string) (map[string][]byte, error) {
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	rawMap := map[string][]byte{}
	err = yaml.Unmarshal(data, &rawMap)
	if err != nil {
		return nil, microerror.Maskf(unmashalToMapFailedError, err.Error())
	}

	return rawMap, nil
}
