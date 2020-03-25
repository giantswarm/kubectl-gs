package appcatalog

import (
	applicationv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Config struct {
	Description string
	Name        string
	URL         string
}

func NewAppCatalogCR(config Config) (*applicationv1alpha1.AppCatalog, error) {

	appCatalogCR := &applicationv1alpha1.AppCatalog{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AppCatalog",
			APIVersion: "v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: metav1.NamespaceDefault,
			Labels:    map[string]string{},
		},
		Spec: applicationv1alpha1.AppCatalogSpec{
			Config: applicationv1alpha1.AppCatalogSpecConfig{
				ConfigMap: applicationv1alpha1.AppCatalogSpecConfigConfigMap{
					Name:      "cf",
					Namespace: "cf",
				},
				Secret: applicationv1alpha1.AppCatalogSpecConfigSecret{
					Name:      "sc",
					Namespace: "sc",
				},
			},
			Description: config.Description,
			Storage: applicationv1alpha1.AppCatalogSpecStorage{
				URL:  config.URL,
				Type: "helm",
			},
		},
	}

	return appCatalogCR, nil
}
