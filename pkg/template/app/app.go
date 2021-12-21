package app

import (
	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	applicationv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
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
	Version                    string
}

type SecretConfig struct {
	Data      []byte
	Name      string
	Namespace string
}

type ConfigMapConfig struct {
	Data      string
	Name      string
	Namespace string
}

func NewAppCR(config Config) ([]byte, error) {
	userConfig := applicationv1alpha1.AppSpecUserConfig{}

	var namespace string
	if config.InCluster {
		namespace = config.Namespace
	} else {
		namespace = config.Cluster
	}

	if config.UserConfigConfigMapName != "" {
		userConfig.ConfigMap = applicationv1alpha1.AppSpecUserConfigConfigMap{
			Name:      config.UserConfigConfigMapName,
			Namespace: namespace,
		}
	}

	if config.UserConfigSecretName != "" {
		userConfig.Secret = applicationv1alpha1.AppSpecUserConfigSecret{
			Name:      config.UserConfigSecretName,
			Namespace: namespace,
		}
	}

	appCR := &applicationv1alpha1.App{
		TypeMeta: metav1.TypeMeta{
			Kind:       "App",
			APIVersion: "application.giantswarm.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.AppName,
			Namespace: namespace,
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
				Namespace: config.Cluster,
			},
		}
		appCR.Spec.KubeConfig = applicationv1alpha1.AppSpecKubeConfig{
			Context: applicationv1alpha1.AppSpecKubeConfigContext{
				Name: config.Cluster + "-kubeconfig",
			},
			InCluster: false,
			Secret: applicationv1alpha1.AppSpecKubeConfigSecret{
				Name:      config.Cluster + "-kubeconfig",
				Namespace: config.Cluster,
			},
		}
	}

	return printAppCR(appCR, config.DefaultingEnabled)
}

func NewConfigMap(config ConfigMapConfig) (*corev1.ConfigMap, error) {
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
			"values": config.Data,
		},
	}

	return configMap, nil
}

func NewSecret(config SecretConfig) (*corev1.Secret, error) {
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
			"values": config.Data,
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
