package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
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
	stderr io.Writer
	stdout io.Writer
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

	// Since organization may be provided in a mixed-cased form,
	// make sure it is lowercased.
	organization := strings.ToLower(r.flag.Organization)

	targetNamespace := r.flag.TargetNamespace
	if targetNamespace == "" {
		targetNamespace = r.flag.Namespace
	}

	clusterName := r.flag.ClusterName
	if clusterName == "" {
		clusterName = r.flag.Cluster
	}

	appConfig := templateapp.Config{
		AppName:           appName,
		Catalog:           r.flag.Catalog,
		CatalogNamespace:  r.flag.CatalogNamespace,
		Cluster:           clusterName,
		DefaultingEnabled: r.flag.DefaultingEnabled,
		InCluster:         r.flag.InCluster,
		Name:              r.flag.Name,
		Namespace:         targetNamespace,
		Organization:      organization,
		Version:           r.flag.Version,
	}

	var assetName string
	if r.flag.InCluster {
		assetName = key.GenerateAssetName(appName, "userconfig")
	} else {
		assetName = key.GenerateAssetName(appName, "userconfig", clusterName)
	}

	// If organization is passed to the command use it as the indicator for
	// templating App CR in org namespace.
	var namespace string
	if r.flag.InCluster {
		namespace = targetNamespace
	} else if appConfig.Organization != "" {
		namespace = fmt.Sprintf("org-%s", appConfig.Organization)
	} else {
		namespace = clusterName
	}

	if r.flag.flagUserSecret != "" {
		secretConfig := templateapp.UserConfig{
			Path:      r.flag.flagUserSecret,
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
		configMapConfig := templateapp.UserConfig{
			Path:      r.flag.flagUserConfigMap,
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

	appCROutput := templateapp.AppCROutput{
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
