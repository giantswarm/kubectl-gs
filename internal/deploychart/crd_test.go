package deploychart

import (
	"context"
	"strings"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	fakeapiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestDetectFluxCRDVersions(t *testing.T) {
	tests := []struct {
		name       string
		crds       []runtime.Object
		wantOCI    string
		wantHR     string
		wantErr    bool
		errContain string
	}{
		{
			name: "v1beta2 OCIRepository and v2 HelmRelease",
			crds: []runtime.Object{
				fakeCRD(OCIRepositoryCRDName, "source.toolkit.fluxcd.io", []crdVersion{
					{Name: "v1beta2", Storage: true, Served: true},
				}),
				fakeCRD(HelmReleaseCRDName, "helm.toolkit.fluxcd.io", []crdVersion{
					{Name: "v2", Storage: true, Served: true},
				}),
			},
			wantOCI: "source.toolkit.fluxcd.io/v1",
			wantHR:  "helm.toolkit.fluxcd.io/v2",
		},
		{
			name: "v1 OCIRepository storage with v1beta2 served",
			crds: []runtime.Object{
				fakeCRD(OCIRepositoryCRDName, "source.toolkit.fluxcd.io", []crdVersion{
					{Name: "v1beta2", Storage: false, Served: true},
					{Name: "v1", Storage: true, Served: true},
				}),
				fakeCRD(HelmReleaseCRDName, "helm.toolkit.fluxcd.io", []crdVersion{
					{Name: "v2", Storage: true, Served: true},
				}),
			},
			wantOCI: "source.toolkit.fluxcd.io/v1",
			wantHR:  "helm.toolkit.fluxcd.io/v2",
		},
		{
			name: "OCIRepository CRD missing",
			crds: []runtime.Object{
				fakeCRD(HelmReleaseCRDName, "helm.toolkit.fluxcd.io", []crdVersion{
					{Name: "v2", Storage: true, Served: true},
				}),
			},
			wantErr:    true,
			errContain: OCIRepositoryCRDName,
		},
		{
			name: "HelmRelease CRD missing",
			crds: []runtime.Object{
				fakeCRD(OCIRepositoryCRDName, "source.toolkit.fluxcd.io", []crdVersion{
					{Name: "v1beta2", Storage: true, Served: true},
				}),
			},
			wantErr:    true,
			errContain: HelmReleaseCRDName,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := fakeapiextensions.NewClientset(tc.crds...)

			result, err := DetectFluxCRDVersions(context.Background(), client)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errContain != "" && !strings.Contains(err.Error(), tc.errContain) {
					t.Errorf("error %q should contain %q", err.Error(), tc.errContain)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.OCIRepositoryAPIVersion != tc.wantOCI {
				t.Errorf("OCIRepositoryAPIVersion = %q, want %q", result.OCIRepositoryAPIVersion, tc.wantOCI)
			}
			if result.HelmReleaseAPIVersion != tc.wantHR {
				t.Errorf("HelmReleaseAPIVersion = %q, want %q", result.HelmReleaseAPIVersion, tc.wantHR)
			}
		})
	}
}

type crdVersion struct {
	Name    string
	Storage bool
	Served  bool
}

func fakeCRD(name, group string, versions []crdVersion) *apiextensionsv1.CustomResourceDefinition {
	var crdVersions []apiextensionsv1.CustomResourceDefinitionVersion
	for _, v := range versions {
		crdVersions = append(crdVersions, apiextensionsv1.CustomResourceDefinitionVersion{
			Name:    v.Name,
			Storage: v.Storage,
			Served:  v.Served,
		})
	}
	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group:    group,
			Versions: crdVersions,
		},
	}
}
