package app

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
	"github.com/giantswarm/kubectl-gs/pkg/annotations"
	"github.com/giantswarm/kubectl-gs/pkg/labels"
	templateapp "github.com/giantswarm/kubectl-gs/pkg/template/app"
)

type Runner struct {
	Flag   *Flag
	Logger micrologger.Logger
	Stdout io.Writer
	Stderr io.Writer
}

func (r *Runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.Flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	var userConfigConfigMapYaml []byte
	var userConfigSecretYaml []byte
	var err error

	appName := r.Flag.AppName
	if appName == "" {
		appName = r.Flag.Name
	}

	appConfig := templateapp.Config{
		AppName:           appName,
		Catalog:           r.Flag.Catalog,
		Cluster:           r.Flag.Cluster,
		DefaultingEnabled: r.Flag.DefaultingEnabled,
		InCluster:         r.Flag.InCluster,
		Name:              r.Flag.Name,
		Namespace:         r.Flag.Namespace,
		Version:           r.Flag.Version,
	}

	var assetName string
	if r.Flag.InCluster {
		assetName = key.GenerateAssetName(appName, "userconfig")
	} else {
		assetName = key.GenerateAssetName(appName, "userconfig", r.Flag.Cluster)
	}

	var namespace string
	if r.Flag.InCluster {
		namespace = r.Flag.Namespace
	} else {
		namespace = r.Flag.Cluster
	}

	if r.Flag.FlagUserSecret != "" {
		userConfigSecretData, err := key.ReadSecretYamlFromFile(afero.NewOsFs(), r.Flag.FlagUserSecret)
		if err != nil {
			return microerror.Mask(err)
		}

		secretConfig := templateapp.SecretConfig{
			Data:      userConfigSecretData,
			Name:      assetName,
			Namespace: namespace,
		}
		userSecret, err := templateapp.NewSecret(secretConfig)
		if err != nil {
			return microerror.Mask(err)
		}
		appConfig.UserConfigSecretName = userSecret.GetName()

		userConfigSecretYaml, err = yaml.Marshal(userSecret)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	if r.Flag.FlagUserConfigMap != "" {
		var configMapData string
		if r.Flag.FlagUserConfigMap != "" {
			configMapData, err = key.ReadConfigMapYamlFromFile(afero.NewOsFs(), r.Flag.FlagUserConfigMap)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		configMapConfig := templateapp.ConfigMapConfig{
			Data:      configMapData,
			Name:      assetName,
			Namespace: namespace,
		}
		userConfigMap, err := templateapp.NewConfigMap(configMapConfig)
		if err != nil {
			return microerror.Mask(err)
		}
		appConfig.UserConfigConfigMapName = userConfigMap.GetName()

		userConfigConfigMapYaml, err = yaml.Marshal(userConfigMap)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	namespaceAnnotations, err := annotations.Parse(r.Flag.FlagNamespaceConfigAnnotations)
	if err != nil {
		return microerror.Mask(err)
	}
	appConfig.NamespaceConfigAnnotations = namespaceAnnotations

	namespaceLabels, err := labels.Parse(r.Flag.FlagNamespaceConfigLabels)
	if err != nil {
		return microerror.Mask(err)
	}
	appConfig.NamespaceConfigLabels = namespaceLabels

	appCRYaml, err := templateapp.NewAppCR(appConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	type AppCROutput struct {
		AppCR               string
		UserConfigSecret    string
		UserConfigConfigMap string
	}

	appCROutput := AppCROutput{
		AppCR:               string(appCRYaml),
		UserConfigConfigMap: string(userConfigConfigMapYaml),
		UserConfigSecret:    string(userConfigSecretYaml),
	}

	t := template.Must(template.New("appCR").Parse(key.AppCRTemplate))

	err = t.Execute(os.Stdout, appCROutput)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
