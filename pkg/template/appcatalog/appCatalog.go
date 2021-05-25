package appcatalog

import (
	applicationv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Config struct {
	CatalogConfigMapName string
	CatalogSecretName    string
	Description          string
	ID                   string
	Name                 string
	Namespace            string
	LogoURL              string
	URL                  string
}

func NewAppCatalogCR(config Config) (*applicationv1alpha1.AppCatalog, error) {
	catalogConfig := applicationv1alpha1.AppCatalogSpecConfig{}

	if config.CatalogConfigMapName != "" {
		catalogConfig.ConfigMap = applicationv1alpha1.AppCatalogSpecConfigConfigMap{
			Name:      config.CatalogConfigMapName,
			Namespace: config.Namespace,
		}
	}

	if config.CatalogSecretName != "" {
		catalogConfig.Secret = applicationv1alpha1.AppCatalogSpecConfigSecret{
			Name:      config.CatalogSecretName,
			Namespace: config.Namespace,
		}
	}

	appCatalogCR := &applicationv1alpha1.AppCatalog{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AppCatalog",
			APIVersion: "application.giantswarm.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: config.Name,
			Labels: map[string]string{
				"app-operator.giantswarm.io/version":     "1.0.0",
				"application.giantswarm.io/catalog-type": "awesome",
			},
		},
		Spec: applicationv1alpha1.AppCatalogSpec{
			Config:      catalogConfig,
			Description: config.Description,
			LogoURL:     config.LogoURL,
			Storage: applicationv1alpha1.AppCatalogSpecStorage{
				URL:  config.URL,
				Type: "helm",
			},
			Title: config.Name,
		},
	}

	return appCatalogCR, nil
}

func NewConfigmapCR(config Config, data string) (*apiv1.ConfigMap, error) {
	configMapCR := &apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.CatalogConfigMapName,
			Namespace: config.Namespace,
		},
		Data: map[string]string{
			"values": data,
		},
	}

	return configMapCR, nil
}

func NewSecretCR(config Config, data []byte) (*apiv1.Secret, error) {
	secretCR := &apiv1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.CatalogSecretName,
			Namespace: config.Namespace,
		},
		Data: map[string][]byte{
			"values": data,
		},
	}

	return secretCR, nil
}
