package app

import (
	applicationv1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Config struct {
	Catalog                 string
	Cluster                 string
	UserConfigConfigMapName string
	UserConfigSecretName    string
	Name                    string
	Namespace               string
	Version                 string
}

type SecretConfig struct {
	Data      map[string][]byte
	Name      string
	Namespace string
}

type ConfigMapConfig struct {
	Data      string
	Name      string
	Namespace string
}

func NewAppCR(config Config) (*applicationv1alpha1.App, error) {
	userConfig := applicationv1alpha1.AppSpecUserConfig{}

	if config.UserConfigConfigMapName != "" {
		userConfig.ConfigMap = applicationv1alpha1.AppSpecUserConfigConfigMap{
			Name:      config.UserConfigConfigMapName,
			Namespace: config.Cluster,
		}
	}

	if config.UserConfigSecretName != "" {
		userConfig.Secret = applicationv1alpha1.AppSpecUserConfigSecret{
			Name:      config.UserConfigSecretName,
			Namespace: config.Cluster,
		}
	}

	appCR := &applicationv1alpha1.App{
		TypeMeta: metav1.TypeMeta{
			Kind:       "App",
			APIVersion: "application.giantswarm.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Cluster,
			Labels: map[string]string{
				"app-operator.giantswarm.io/version": "1.0.0",
			},
		},
		Spec: applicationv1alpha1.AppSpec{
			Catalog:   config.Catalog,
			Name:      config.Name,
			Namespace: config.Namespace,
			Config: applicationv1alpha1.AppSpecConfig{
				ConfigMap: applicationv1alpha1.AppSpecConfigConfigMap{
					Name:      config.Cluster + "-cluster-values",
					Namespace: config.Cluster,
				},
			},
			KubeConfig: applicationv1alpha1.AppSpecKubeConfig{
				Context: applicationv1alpha1.AppSpecKubeConfigContext{
					Name: config.Cluster + "-kubeconfig",
				},
				InCluster: false,
				Secret: applicationv1alpha1.AppSpecKubeConfigSecret{
					Name:      config.Cluster + "-kubeconfig",
					Namespace: config.Cluster,
				},
			},
			UserConfig: userConfig,
			Version:    config.Version,
		},
	}

	return appCR, nil
}

func NewConfigmapCR(config ConfigMapConfig) (*apiv1.ConfigMap, error) {

	configMapCR := &apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			Labels:    map[string]string{},
		},
		Data: map[string]string{
			"values": config.Data,
		},
	}

	return configMapCR, nil
}

func NewSecretCR(config SecretConfig) (*apiv1.Secret, error) {

	secretCR := &apiv1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			Labels:    map[string]string{},
		},
		Data: config.Data,
	}

	return secretCR, nil
}
