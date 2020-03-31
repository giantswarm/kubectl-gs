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
	v1 "k8s.io/api/core/v1"
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
	var userConfigSecretCRYaml []byte
	var userConfigConfigMapCRYaml []byte
	var kubeSecretCR *v1.Secret
	var userSecretCR *v1.Secret
	var userConfigMapCR *v1.ConfigMap
	var err error
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
		kubeSecretCR, err = app.NewSecretCR(secretConfig)

		kubeConfigSecretCRYaml, err = yaml.Marshal(kubeSecretCR)
		if err != nil {
			return microerror.Mask(err)
		}
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
		return microerror.Mask(err)
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
		return microerror.Mask(err)
	}

	{
		userConfigSecretData, err := key.ReadSecretYamlFromFile(afero.NewOsFs(), r.flag.Secret)
		if err != nil {
			return microerror.Mask(err)
		}

		secretConfig := app.SecretConfig{
			Data:      userConfigSecretData,
			Name:      key.GenerateAssetName(r.flag.Name, "userconfig", appAssetID),
			Namespace: r.flag.Namespace,
		}
		userSecretCR, err = app.NewSecretCR(secretConfig)

		userConfigSecretCRYaml, err = yaml.Marshal(userSecretCR)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		var configMapData string
		if r.flag.flagUserConfigMap != "" {
			configMapData, err = key.ReadConfigMapYamlFromFile(afero.NewOsFs(), r.flag.flagUserConfigMap)
			if err != nil {
				return microerror.Mask(err)
			}
		}
		configMapConfig := app.ConfigMapConfig{
			Data:      configMapData,
			Name:      key.GenerateAssetName(r.flag.Name, "userconfig", appAssetID),
			Namespace: r.flag.Namespace,
		}
		userConfigMapCR, err = app.NewConfigmapCR(configMapConfig)

		userConfigConfigMapCRYaml, err = yaml.Marshal(userConfigMapCR)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	appConfig := app.Config{
		Catalog:                 r.flag.Catalog,
		ID:                      appAssetID,
		KubeConfigContext:       r.flag.KubeConfigContext,
		KubeConfigSecretName:    kubeSecretCR.GetName(),
		UserConfigSecretName:    userSecretCR.GetName(),
		UserConfigConfigMapName: userConfigMapCR.GetName(),
		Name:                    r.flag.Name,
		Namespace:               r.flag.Namespace,
		Version:                 r.flag.Version,
	}
	appCR, err := app.NewAppCR(appConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	appCRYaml, err := yaml.Marshal(appCR)
	if err != nil {
		return microerror.Mask(err)
	}

	type AppCROutput struct {
		AppCR                 string
		ConfigmapCR           string
		KubeConfigSecretCR    string
		UserConfigSecretCR    string
		UserConfigConfigMapCR string
		SecretCR              string
	}

	appCROutput := AppCROutput{
		AppCR:                 string(appCRYaml),
		ConfigmapCR:           string(configmapCRYaml),
		KubeConfigSecretCR:    string(kubeConfigSecretCRYaml),
		UserConfigConfigMapCR: string(userConfigConfigMapCRYaml),
		UserConfigSecretCR:    string(userConfigSecretCRYaml),
		SecretCR:              string(secretCRYaml),
	}

	t := template.Must(template.New("appCatalogCR").Parse(key.AppCRTemplate))

	err = t.Execute(os.Stdout, appCROutput)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
