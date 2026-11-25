package organization

import (
	"testing"

	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
)

// Test_NewOrganization_Build checks what the management cluster's
// Kustomization applies from the organisation's directory, once kustomize has
// run its replacements: the namespace Kustomization's targetNamespace and
// postBuild carry the Organization's name, hyphens included.
func Test_NewOrganization_Build(t *testing.T) {
	testCases := []struct {
		organization string
		namespace    string
	}{
		{organization: "demoorg", namespace: "org-demoorg"},
		{organization: "my-org", namespace: "org-my-org"},
		{organization: "a-b-c", namespace: "org-a-b-c"},
	}

	for _, tc := range testCases {
		t.Run(tc.organization, func(t *testing.T) {
			config, err := NewOrganization(common.StructureConfig{
				ManagementCluster: "demomc",
				Organization:      tc.organization,
				RepositoryName:    "gitops-demo",
			})
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			fs := filesys.MakeFsInMemory()
			for _, o := range config.FsObjects {
				if o.Data == nil {
					err = fs.MkdirAll(o.RelativePath)
				} else {
					err = fs.WriteFile(o.RelativePath, o.Data)
				}
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			}

			orgDir := key.BaseDirPath("demomc", tc.organization, "")
			resMap, err := krusty.MakeKustomizer(krusty.MakeDefaultOptions()).Run(fs, orgDir)
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			found := false
			for _, r := range resMap.Resources() {
				if r.GetKind() != "Kustomization" {
					continue
				}
				found = true

				got, err := r.GetString("spec.targetNamespace")
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
				if got != tc.namespace {
					t.Errorf("expected targetNamespace %q, got %q", tc.namespace, got)
				}

				got, err = r.GetString("spec.postBuild.substitute.organization")
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
				if got != tc.organization {
					t.Errorf("expected organization %q, got %q", tc.organization, got)
				}
			}
			if !found {
				t.Fatal("expected the organisation's namespace Kustomization")
			}
		})
	}
}
