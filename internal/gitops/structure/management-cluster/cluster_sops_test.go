package mcluster

import (
	"errors"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/encryption"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
)

const (
	sopsFingerprint = "12345689ABCDEF"
	sopsMC          = "demomc"
	sopsRepository  = "gitops-demo"
)

// Test_NewManagementCluster_SopsRules checks that the master key encrypts
// the management cluster's secrets, and the workload clusters' keys in the
// organisations' namespace/secrets.
func Test_NewManagementCluster_SopsRules(t *testing.T) {
	config, err := NewManagementCluster(common.StructureConfig{
		EncryptionKeyPair: encryption.KeyPair{
			Fingerprint: sopsFingerprint,
			PrivateData: "private key material",
			PublicData:  "public key material",
		},
		ManagementCluster: sopsMC,
		RepositoryName:    sopsRepository,
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	out, err := config.PostModifiers[key.SopsConfigFileName()].Execute([]byte("creation_rules: []\n"))
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	var sops struct {
		CreationRules []struct {
			PathRegex string `json:"path_regex"`
			PGP       string `json:"pgp"`
		} `json:"creation_rules"`
	}
	err = yaml.Unmarshal(out, &sops)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	var got []string
	for _, r := range sops.CreationRules {
		if r.PGP != sopsFingerprint {
			t.Errorf("rule %s encrypts with %s, expected the master key", r.PathRegex, r.PGP)
		}
		got = append(got, r.PathRegex)
	}

	expected := []string{
		`management-clusters/demomc/secrets/.*\.enc\.yaml`,
		`management-clusters/demomc/organizations/[^/]+/namespace/secrets/.*\.(gpg|age)key\.enc\.yaml`,
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("rules (-want +got):\n%s", diff)
	}
}

// Test_NewManagementCluster_RefusesIncompleteSecrets checks that running the
// command again on a management cluster whose secrets/ has no
// kustomization.yaml, but other keys, doesn't take them out of what Flux
// applies.
func Test_NewManagementCluster_RefusesIncompleteSecrets(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, "/repo/management-clusters/demomc/secrets/demowc.gpgkey.enc.yaml", []byte("kind: Secret\n"), 0600)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	config, err := NewManagementCluster(common.StructureConfig{ManagementCluster: sopsMC, RepositoryName: sopsRepository})
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	config.Fs = fs
	config.Path = "/repo"
	config.Stdout = io.Discard

	err = creator.NewCreator(*config).Create()
	if !errors.Is(err, creator.ValidationError) {
		t.Fatalf("expected a validation error, got: %v", err)
	}
	if ok, _ := afero.Exists(fs, "/repo/management-clusters/demomc/demomc.yaml"); ok {
		t.Errorf("expected nothing written")
	}
}
