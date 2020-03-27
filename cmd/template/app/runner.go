package app

import (
	"context"
	"io"
	"os"
	"text/template"

	"github.com/ghodss/yaml"
	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/template/app"
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
	var kubeConfigSecretCRYaml []byte
	appAssetID := key.GenerateID()

	if r.flag.KubeConfigSecret != "" {
		kubeConfigSecretData, err := key.ReadSecretYamlFromFile(afero.NewOsFs(), r.flag.Secret)
		if err != nil {
			return microerror.Mask(err)
		}

		secretConfig := app.SecretConfig{
			Data:      kubeConfigSecretData,
			Name:      key.GenerateAssetName(r.flag.Name, "kubeconfig", appAssetID),
			Namespace: r.flag.Namespace,
		}
		secretCR, err := app.NewSecretCR(secretConfig)

		kubeConfigSecretCRYaml, err = yaml.Marshal(secretCR)
		if err != nil {
			return microerror.Maskf(unmashalToMapFailedError, err.Error())
		}
	}

	appConfig := app.Config{
		Catalog:           r.flag.Catalog,
		ID:                appAssetID,
		KubeConfigContext: r.flag.KubeConfigContext,
		Name:              r.flag.Name,
		Namespace:         r.flag.Namespace,
	}
	appCR, err := app.NewAppCR(appConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	appCRYaml, err := yaml.Marshal(appCR)
	if err != nil {
		return microerror.Mask(err)
	}

	var configMapData string
	if r.flag.ConfigMap != "" {
		configMapData, err = key.ReadConfigMapYamlFromFile(afero.NewOsFs(), r.flag.ConfigMap)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	configMapConfig := app.ConfigMapConfig{
		Data:      configMapData,
		Name:      key.GenerateAssetName(r.flag.Name, appAssetID),
		Namespace: r.flag.Namespace,
	}
	configmapCR, err := app.NewConfigmapCR(configMapConfig)

	configmapCRYaml, err := yaml.Marshal(configmapCR)
	if err != nil {
		return microerror.Maskf(unmashalToMapFailedError, err.Error())
	}

	var secretData map[string][]byte
	if r.flag.Secret != "" {
		secretData, err = key.ReadSecretYamlFromFile(afero.NewOsFs(), r.flag.Secret)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	secretConfig := app.SecretConfig{
		Data:      secretData,
		Name:      key.GenerateAssetName(r.flag.Name, appAssetID),
		Namespace: r.flag.Namespace,
	}
	secretCR, err := app.NewSecretCR(secretConfig)

	secretCRYaml, err := yaml.Marshal(secretCR)
	if err != nil {
		return microerror.Maskf(unmashalToMapFailedError, err.Error())
	}

	type AppCROutput struct {
		AppCR              string
		ConfigmapCR        string
		KubeConfigSecretCR string
		SecretCR           string
	}

	appCROutput := AppCROutput{
		AppCR:              string(appCRYaml),
		ConfigmapCR:        string(configmapCRYaml),
		KubeConfigSecretCR: string(kubeConfigSecretCRYaml),
		SecretCR:           string(secretCRYaml),
	}

	t := template.Must(template.New("appCatalogCR").Parse(key.AppCRTemplate))

	err = t.Execute(os.Stdout, appCROutput)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
