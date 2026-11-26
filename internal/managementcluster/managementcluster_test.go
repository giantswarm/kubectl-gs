package managementcluster

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func metadataConfigMap(data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: "flux-giantswarm", Name: "management-cluster-metadata"},
		Data:       data,
	}
}

func failingGet(err error) interceptor.Funcs {
	return interceptor.Funcs{
		Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
			return err
		},
	}
}

func Test_Name(t *testing.T) {
	testCases := []struct {
		name          string
		objects       []client.Object
		interceptor   interceptor.Funcs
		expectedName  string
		expectedError bool
	}{
		{
			name:         "case 0: the ConfigMap names the management cluster",
			objects:      []client.Object{metadataConfigMap(map[string]string{"NAME": "my-mc"})},
			expectedName: "my-mc",
		},
		{
			name: "case 1: no ConfigMap",
		},
		{
			name:    "case 2: the ConfigMap holds no name",
			objects: []client.Object{metadataConfigMap(nil)},
		},
		{
			name:        "case 3: the user may not read the ConfigMap",
			interceptor: failingGet(apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "management-cluster-metadata", errors.New("denied"))),
		},
		{
			name:          "case 4: the API server does not answer",
			interceptor:   failingGet(errors.New("connection refused")),
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrlClient := fake.NewClientBuilder().WithObjects(tc.objects...).WithInterceptorFuncs(tc.interceptor).Build()

			name, err := Name(context.Background(), ctrlClient)
			if tc.expectedError {
				if err == nil {
					t.Fatal("expected an error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if name != tc.expectedName {
				t.Fatalf("expected name %q, got %q", tc.expectedName, name)
			}
		})
	}
}
