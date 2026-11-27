package organization

import (
	"testing"

	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
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

			r := buildNamespaceKustomization(t, config.FsObjects, key.BaseDirPath("demomc", tc.organization, ""))
			checkField(t, r, "spec.targetNamespace", tc.namespace)
			checkField(t, r, "spec.postBuild.substitute.organization", tc.organization)
		})
	}
}

// buildNamespaceKustomization writes the objects to a filesystem, builds dir
// with kustomize, and returns the one Flux Kustomization it holds.
func buildNamespaceKustomization(t *testing.T, objects []*creator.FsObject, dir string) *resource.Resource {
	t.Helper()

	fs := filesys.MakeFsInMemory()
	for _, o := range objects {
		var err error
		if o.Data == nil {
			err = fs.MkdirAll(o.RelativePath)
		} else {
			err = fs.WriteFile(o.RelativePath, o.Data)
		}
		if err != nil {
			t.Fatalf("unexpected error: %s", err.Error())
		}
	}

	resMap, err := krusty.MakeKustomizer(krusty.MakeDefaultOptions()).Run(fs, dir)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	var found []*resource.Resource
	for _, r := range resMap.Resources() {
		if r.GetKind() == "Kustomization" {
			found = append(found, r)
		}
	}
	if len(found) != 1 {
		t.Fatalf("expected one Flux Kustomization, got %d", len(found))
	}

	return found[0]
}

// checkField checks the string at the field path of the resource.
func checkField(t *testing.T, r *resource.Resource, fieldPath, want string) {
	t.Helper()

	got, err := r.GetString(fieldPath)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	if got != want {
		t.Errorf("expected %s %q, got %q", fieldPath, want, got)
	}
}
