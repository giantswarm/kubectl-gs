package app

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
	v1 "k8s.io/api/core/v1"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/template/app"
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
	var userSecretCR *v1.Secret
	var userConfigMapCR *v1.ConfigMap
	var userConfigConfigMapCRYaml []byte
	var userConfigSecretCRYaml []byte
	var err error

	appConfig := app.Config{
		Catalog:   r.flag.Catalog,
		Name:      r.flag.Name,
		Namespace: r.flag.Namespace,
		Cluster:   r.flag.Cluster,
		Version:   r.flag.Version,
	}

	if r.flag.flagUserSecret != "" {
		userConfigSecretData, err := key.ReadSecretYamlFromFile(afero.NewOsFs(), r.flag.flagUserSecret)
		if err != nil {
			return microerror.Mask(err)
		}

		secretConfig := app.SecretConfig{
			Data:      userConfigSecretData,
			Name:      key.GenerateAssetName(r.flag.Name, "userconfig", r.flag.Cluster),
			Namespace: r.flag.Cluster,
		}
		userSecretCR, err = app.NewSecretCR(secretConfig)
		appConfig.UserConfigSecretName = userSecretCR.GetName()

		userConfigSecretCRYaml, err = yaml.Marshal(userSecretCR)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	if r.flag.flagUserConfigMap != "" {
		var configMapData string
		if r.flag.flagUserConfigMap != "" {
			configMapData, err = key.ReadConfigMapYamlFromFile(afero.NewOsFs(), r.flag.flagUserConfigMap)
			if err != nil {
				return microerror.Mask(err)
			}
		}
		configMapConfig := app.ConfigMapConfig{
			Data:      configMapData,
			Name:      key.GenerateAssetName(r.flag.Name, "userconfig", r.flag.Cluster),
			Namespace: r.flag.Cluster,
		}
		userConfigMapCR, err = app.NewConfigmapCR(configMapConfig)
		appConfig.UserConfigConfigMapName = userConfigMapCR.GetName()

		userConfigConfigMapCRYaml, err = yaml.Marshal(userConfigMapCR)
		if err != nil {
			return microerror.Mask(err)
		}
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
		UserConfigSecretCR    string
		UserConfigConfigMapCR string
	}

	appCROutput := AppCROutput{
		AppCR:                 string(appCRYaml),
		UserConfigConfigMapCR: string(userConfigConfigMapCRYaml),
		UserConfigSecretCR:    string(userConfigSecretCRYaml),
	}

	t := template.Must(template.New("appCatalogCR").Parse(key.AppCRTemplate))

	err = t.Execute(os.Stdout, appCROutput)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
