package app

import (
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	applicationv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/k8smetadata/pkg/label"

	"github.com/giantswarm/kubectl-gs/internal/key"
)

type Config struct {
	AppName                    string
	Catalog                    string
	Cluster                    string
	DefaultingEnabled          bool
	InCluster                  bool
	Name                       string
	Namespace                  string
	NamespaceConfigAnnotations map[string]string
	NamespaceConfigLabels      map[string]string
	UserConfigConfigMapName    string
	UserConfigSecretName       string
	Organization               string
	Version                    string
}

type UserConfig struct {
	Name      string
	Namespace string
	Path      string
}

type AppCROutput struct {
	AppCR               string
	UserConfigSecret    string
	UserConfigConfigMap string
}

func NewAppCR(config Config) ([]byte, error) {
	userConfig := applicationv1alpha1.AppSpecUserConfig{}
	appLabels := map[string]string{}

	// When templating App CR for an organization namespace add
	// `giantswarm.io/cluster` label for cluster selection.
	var crNamespace string
	if config.InCluster {
		crNamespace = config.Namespace
	} else if config.Organization != "" {
		crNamespace = fmt.Sprintf("org-%s", config.Organization)
		appLabels[label.Cluster] = config.Cluster
	} else {
		crNamespace = config.Cluster
	}

	if config.UserConfigConfigMapName != "" {
		userConfig.ConfigMap = applicationv1alpha1.AppSpecUserConfigConfigMap{
			Name:      config.UserConfigConfigMapName,
			Namespace: crNamespace,
		}
	}

	if config.UserConfigSecretName != "" {
		userConfig.Secret = applicationv1alpha1.AppSpecUserConfigSecret{
			Name:      config.UserConfigSecretName,
			Namespace: crNamespace,
		}
	}

	appCR := &applicationv1alpha1.App{
		TypeMeta: metav1.TypeMeta{
			Kind:       "App",
			APIVersion: "application.giantswarm.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.AppName,
			Namespace: crNamespace,
			Labels:    appLabels,
		},
		Spec: applicationv1alpha1.AppSpec{
			Catalog:   config.Catalog,
			Name:      config.Name,
			Namespace: config.Namespace,
			KubeConfig: applicationv1alpha1.AppSpecKubeConfig{
				InCluster: config.InCluster,
			},
			UserConfig: userConfig,
			Version:    config.Version,
			NamespaceConfig: applicationv1alpha1.AppSpecNamespaceConfig{
				Annotations: config.NamespaceConfigAnnotations,
				Labels:      config.NamespaceConfigLabels,
			},
		},
	}

	if config.InCluster {
		appCR.SetLabels(map[string]string{
			"app-operator.giantswarm.io/version": "0.0.0",
		})
	}

	if !config.DefaultingEnabled && !config.InCluster {
		appCR.SetLabels(map[string]string{
			"app-operator.giantswarm.io/version": "1.0.0",
		})

		appCR.Spec.Config = applicationv1alpha1.AppSpecConfig{
			ConfigMap: applicationv1alpha1.AppSpecConfigConfigMap{
				Name:      config.Cluster + "-cluster-values",
				Namespace: crNamespace,
			},
		}
		appCR.Spec.KubeConfig = applicationv1alpha1.AppSpecKubeConfig{
			Context: applicationv1alpha1.AppSpecKubeConfigContext{
				Name: config.Cluster + "-kubeconfig",
			},
			InCluster: false,
			Secret: applicationv1alpha1.AppSpecKubeConfigSecret{
				Name:      config.Cluster + "-kubeconfig",
				Namespace: crNamespace,
			},
		}
	}

	return printAppCR(appCR, config.DefaultingEnabled)
}

func NewConfigMap(config UserConfig) (*corev1.ConfigMap, error) {
	configMapData, err := key.ReadConfigMapYamlFromFile(afero.NewOsFs(), config.Path)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
		},
		Data: map[string]string{
			"values": configMapData,
		},
	}

	return configMap, nil
}

func NewSecret(config UserConfig) (*corev1.Secret, error) {
	userConfigSecretData, err := key.ReadSecretYamlFromFile(afero.NewOsFs(), config.Path)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
		},
		Data: map[string][]byte{
			"values": userConfigSecretData,
		},
	}

	return secret, nil
}

// printAppCR removes empty fields from the app CR YAML. This is needed because
// although the fields are optional we do not use struct pointers. This will
// be fixed in a future version of the App CRD.
func printAppCR(appCR *v1alpha1.App, defaultingEnabled bool) ([]byte, error) {
	appCRYaml, err := yaml.Marshal(appCR)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	rawAppCR := map[string]interface{}{}
	err = yaml.Unmarshal(appCRYaml, &rawAppCR)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	delete(rawAppCR, "status")

	metadata, ok := rawAppCR["metadata"].(map[string]interface{})
	if !ok {
		return nil, microerror.Maskf(executionFailedError, "failed to get metadata for app CR")
	}
	delete(metadata, "creationTimestamp")

	spec, ok := rawAppCR["spec"].(map[string]interface{})
	if !ok {
		return nil, microerror.Maskf(executionFailedError, "failed to get spec for app CR")
	}

	delete(spec, "install")
	if len(appCR.Spec.NamespaceConfig.Annotations) == 0 && len(appCR.Spec.NamespaceConfig.Labels) == 0 {
		delete(spec, "namespaceConfig")
	}

	if defaultingEnabled {
		delete(spec, "config")
		spec["kubeConfig"] = map[string]bool{
			"inCluster": appCR.Spec.KubeConfig.InCluster,
		}
	}

	if appCR.Spec.UserConfig.ConfigMap.Name == "" && appCR.Spec.UserConfig.Secret.Name == "" {
		delete(spec, "userConfig")
	} else {
		userConfig, ok := spec["userConfig"].(map[string]interface{})
		if !ok {
			return nil, microerror.Maskf(executionFailedError, "failed to get userConfig for app CR")
		}

		if appCR.Spec.UserConfig.ConfigMap.Name != "" && appCR.Spec.UserConfig.Secret.Name == "" {
			delete(userConfig, "secret")
		} else if appCR.Spec.UserConfig.ConfigMap.Name == "" && appCR.Spec.UserConfig.Secret.Name != "" {
			delete(userConfig, "configMap")
		}
	}

	outputYaml, err := yaml.Marshal(rawAppCR)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return outputYaml, nil
}
