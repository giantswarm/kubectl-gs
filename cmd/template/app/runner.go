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
	var userConfigConfigMapYaml []byte
	var userConfigSecretYaml []byte
	var err error

	appName := r.flag.AppName
	if appName == "" {
		appName = r.flag.Name
	}

	appConfig := templateapp.Config{
		AppName:           appName,
		Catalog:           r.flag.Catalog,
		Cluster:           r.flag.Cluster,
		DefaultingEnabled: r.flag.DefaultingEnabled,
		InCluster:         r.flag.InCluster,
		Name:              r.flag.Name,
		Namespace:         r.flag.Namespace,
		Version:           r.flag.Version,
	}

	var assetName string
	if r.flag.InCluster {
		assetName = key.GenerateAssetName(appName, "userconfig")
	} else {
		assetName = key.GenerateAssetName(appName, "userconfig", r.flag.Cluster)
	}

	var namespace string
	if r.flag.InCluster {
		namespace = r.flag.Namespace
	} else {
		namespace = r.flag.Cluster
	}

	if r.flag.flagUserSecret != "" {
		userConfigSecretData, err := key.ReadSecretYamlFromFile(afero.NewOsFs(), r.flag.flagUserSecret)
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

	if r.flag.flagUserConfigMap != "" {
		var configMapData string
		if r.flag.flagUserConfigMap != "" {
			configMapData, err = key.ReadConfigMapYamlFromFile(afero.NewOsFs(), r.flag.flagUserConfigMap)
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

	namespaceAnnotations, err := annotations.Parse(r.flag.flagNamespaceConfigAnnotations)
	if err != nil {
		return microerror.Mask(err)
	}
	appConfig.NamespaceConfigAnnotations = namespaceAnnotations

	namespaceLabels, err := labels.Parse(r.flag.flagNamespaceConfigLabels)
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
