package encryption

import (
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"

	gitopsenc "github.com/giantswarm/kubectl-gs/v6/internal/gitops/encryption"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
)

// Test_NewEncryption_CreateOrganizationNamespaceTarget checks that an
// organisation's key for any directory in namespace/ sets up decryption on
// the namespace Kustomization, which applies it, and that only
// namespace/secrets is created and listed for it.
func Test_NewEncryption_CreateOrganizationNamespaceTarget(t *testing.T) {
	fs := seedCreate(t, createMasterRule, map[string]string{
		createRepo + "/management-clusters/demomc/demomc.yaml": strings.Replace(createKustomization, "%s", "demomc-gitops", 1),
	})

	config, err := NewEncryption(common.StructureConfig{
		EncryptionKeyPair: gitopsenc.KeyPair{Fingerprint: "ORGFINGERPRINT", PrivateData: "org key", PublicData: "public"},
		EncryptionTarget:  "namespace/shared-objects/",
		ManagementCluster: testMC,
		Organization:      testOrg,
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	config.Fs = fs
	config.Path = createRepo
	config.Stdout = io.Discard

	err = creator.NewCreator(*config).Create()
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	checkDecryption(t, fs, createOrgDir+"/namespace.yaml", "sops-gpg-master")

	var kustomization struct {
		Resources []string `json:"resources"`
	}
	readYAML(t, fs, createOrgDir+"/namespace/kustomization.yaml", &kustomization)
	if diff := cmp.Diff([]string{"../workload-clusters"}, kustomization.Resources); diff != "" {
		t.Errorf("namespace resources (-want +got):\n%s", diff)
	}

	data, err := afero.ReadFile(fs, createSopsFile)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	rule := `management-clusters/demomc/organizations/demoorg/namespace/shared-objects/.*\.enc\.yaml`
	if !strings.Contains(string(data), rule) {
		t.Errorf("expected a rule for %s, got:\n%s", rule, data)
	}
}

// Test_isNamespaceTarget checks which targets are in the organisation's
// namespace/, which its namespace Kustomization applies and decrypts.
func Test_isNamespaceTarget(t *testing.T) {
	testCases := map[string]bool{
		"namespace":                 true,
		"namespace/":                true,
		"namespace/secrets":         true,
		"namespace/secrets/":        true,
		"namespace/shared-objects/": true,
		"./namespace/a/b/":          true,
		"/namespace/a":              true,
		"secrets/":                  false,
		"namespaces/":               false,
		"namespace-objects/":        false,
		"namespace/../secrets":      false,
		"workload-clusters/":        false,
		"":                          false,
	}

	for target, want := range testCases {
		if got := isNamespaceTarget(target); got != want {
			t.Errorf("isNamespaceTarget(%q) = %t, want %t", target, got, want)
		}
	}
}
