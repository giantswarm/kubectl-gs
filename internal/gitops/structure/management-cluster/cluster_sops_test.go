package mcluster

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/encryption"
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
