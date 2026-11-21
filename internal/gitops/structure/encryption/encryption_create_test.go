package encryption

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"

	gitopsenc "github.com/giantswarm/kubectl-gs/v6/internal/gitops/encryption"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
)

const (
	testMC        = "demomc"
	testOrg       = "demoorg"
	testWC        = "demowc"
	testPublicKey = "public key material"
	testTarget    = "secrets/"

	testNamespaceTarget = "namespace/secrets/"

	createRepo     = "/repo"
	createOrgDir   = createRepo + "/management-clusters/demomc/organizations/demoorg"
	createKeysDir  = createOrgDir + "/namespace/secrets"
	createKeyFile  = createKeysDir + "/demowc.gpgkey.enc.yaml"
	createWCFile   = createOrgDir + "/workload-clusters/demowc.yaml"
	createAppsFile = createOrgDir + "/workload-clusters/demowc-apps.yaml"
	createSopsFile = createRepo + "/.sops.yaml"

	createMasterRule = `creation_rules:
  - encrypted_regex: ^(data|stringData)$
    path_regex: management-clusters/demomc/secrets/.*\.enc\.yaml
    pgp: MASTERFINGERPRINT
`
	createKustomization = "apiVersion: kustomize.toolkit.fluxcd.io/v1\nkind: Kustomization\nmetadata:\n  name: %s\nspec:\n  path: ./management-clusters/demomc/organizations/demoorg/workload-clusters/demowc\n"
)

// Test_NewEncryption_Create checks where a workload cluster's key goes: into
// the organisation's namespace/secrets, in the organisation's namespace,
// protected from deletion and encrypted with the master key, which the
// organisation's namespace Kustomization decrypts it with.
func Test_NewEncryption_Create(t *testing.T) {
	testCases := []struct {
		name         string
		seed         map[string]string
		expectedKeys []string
		expectApps   bool
	}{
		{
			name: "existing Secret keeps its keys",
			seed: map[string]string{
				createKeyFile:  "apiVersion: v1\ndata:\n  demowc.OLDFINGERPRINT.asc: b2xkIGtleQ==\nkind: Secret\nmetadata:\n  name: sops-gpg-demowc\n  namespace: org-demoorg\n",
				createAppsFile: strings.Replace(createKustomization, "%s", "demomc-clusters-demowc-apps", 1),
			},
			expectedKeys: []string{"demowc.OLDFINGERPRINT.asc", "demowc.NEWFINGERPRINT.asc"},
			expectApps:   true,
		},
		{
			name:         "cluster with a single Kustomization",
			expectedKeys: []string{"demowc.NEWFINGERPRINT.asc"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := seedCreate(t, createMasterRule, tc.seed)

			err := createWorkloadClusterKey(fs)
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			checkSecret(t, fs, tc.expectedKeys)
			checkListed(t, fs)
			checkDecryption(t, fs, createOrgDir+"/namespace.yaml", "sops-gpg-master")
			checkDecryption(t, fs, createWCFile, "sops-gpg-demowc")
			if tc.expectApps {
				checkDecryption(t, fs, createAppsFile, "sops-gpg-demowc")
			} else if ok, _ := afero.Exists(fs, createAppsFile); ok {
				t.Errorf("expected no %s", createAppsFile)
			}
			checkRules(t, fs)
		})
	}
}

// Test_NewEncryption_CreateWithoutMasterKey checks that a workload cluster
// key is refused when the management cluster has no master key to encrypt
// it with.
func Test_NewEncryption_CreateWithoutMasterKey(t *testing.T) {
	fs := seedCreate(t, "creation_rules: []\n", nil)

	err := createWorkloadClusterKey(fs)
	if !errors.Is(err, creator.ValidationError) {
		t.Fatalf("expected a validation error, got: %v", err)
	}
	if ok, _ := afero.Exists(fs, createKeyFile); ok {
		t.Errorf("expected no key to be written")
	}
}

func seedCreate(t *testing.T, sops string, extra map[string]string) afero.Fs {
	t.Helper()

	fs := afero.NewMemMapFs()
	seed := map[string]string{
		createSopsFile:                                 sops,
		createOrgDir + "/namespace.yaml":               "apiVersion: kustomize.toolkit.fluxcd.io/v1\nkind: Kustomization\nmetadata:\n  name: demomc-organization-demoorg\nspec:\n  targetNamespace: org-demoorg\n",
		createOrgDir + "/namespace/kustomization.yaml": "resources:\n  - ../workload-clusters\n",
		createWCFile:                                   strings.Replace(createKustomization, "%s", "demomc-clusters-demowc", 1),
	}
	for path, data := range extra {
		seed[path] = data
	}
	for path, data := range seed {
		err := afero.WriteFile(fs, path, []byte(data), 0600)
		if err != nil {
			t.Fatalf("unexpected error: %s", err.Error())
		}
	}

	return fs
}

func createWorkloadClusterKey(fs afero.Fs) error {
	config, err := NewEncryption(common.StructureConfig{
		EncryptionKeyPair: gitopsenc.KeyPair{
			Fingerprint: "NEWFINGERPRINT",
			PrivateData: "new key",
			PublicData:  testPublicKey,
		},
		EncryptionTarget:  testTarget,
		ManagementCluster: testMC,
		Organization:      testOrg,
		WorkloadCluster:   testWC,
	})
	if err != nil {
		return err
	}
	config.Fs = fs
	config.Path = createRepo
	config.Stdout = io.Discard

	return creator.NewCreator(*config).Create()
}

func checkSecret(t *testing.T, fs afero.Fs, keys []string) {
	t.Helper()

	var secret struct {
		Data     map[string]string `json:"data"`
		Metadata struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"metadata"`
	}
	readYAML(t, fs, createKeyFile, &secret)

	if secret.Metadata.Name != "sops-gpg-demowc" || secret.Metadata.Namespace != "org-demoorg" {
		t.Errorf("expected the Secret org-demoorg/sops-gpg-demowc, got %s/%s", secret.Metadata.Namespace, secret.Metadata.Name)
	}
	for _, k := range keys {
		if _, ok := secret.Data[k]; !ok {
			t.Errorf("expected key %q in the Secret", k)
		}
	}
}

func checkListed(t *testing.T, fs afero.Fs) {
	t.Helper()

	var kustomization struct {
		CommonAnnotations map[string]string `json:"commonAnnotations"`
		Resources         []string          `json:"resources"`
	}
	readYAML(t, fs, createKeysDir+"/kustomization.yaml", &kustomization)
	if diff := cmp.Diff([]string{"demowc.gpgkey.enc.yaml"}, kustomization.Resources); diff != "" {
		t.Errorf("namespace/secrets resources (-want +got):\n%s", diff)
	}
	if kustomization.CommonAnnotations["kustomize.toolkit.fluxcd.io/prune"] != "disabled" {
		t.Errorf("expected the keys to be kept by Flux, got annotations %v", kustomization.CommonAnnotations)
	}

	readYAML(t, fs, createOrgDir+"/namespace/kustomization.yaml", &kustomization)
	if diff := cmp.Diff([]string{"../workload-clusters", "secrets"}, kustomization.Resources); diff != "" {
		t.Errorf("namespace resources (-want +got):\n%s", diff)
	}
}

func checkDecryption(t *testing.T, fs afero.Fs, path, secret string) {
	t.Helper()

	var kustomization struct {
		Spec struct {
			Decryption struct {
				SecretRef struct {
					Name string `json:"name"`
				} `json:"secretRef"`
			} `json:"decryption"`
		} `json:"spec"`
	}
	readYAML(t, fs, path, &kustomization)
	if kustomization.Spec.Decryption.SecretRef.Name != secret {
		t.Errorf("expected %s to decrypt with %s, got %q", path, secret, kustomization.Spec.Decryption.SecretRef.Name)
	}
}

func checkRules(t *testing.T, fs afero.Fs) {
	t.Helper()

	var sops struct {
		CreationRules []struct {
			PathRegex string `json:"path_regex"`
			PGP       string `json:"pgp"`
		} `json:"creation_rules"`
	}
	readYAML(t, fs, createSopsFile, &sops)

	var got []string
	for _, r := range sops.CreationRules {
		got = append(got, r.PathRegex+" "+r.PGP)
	}
	expected := []string{
		`management-clusters/demomc/organizations/[^/]+/namespace/secrets/.*\.(gpg|age)key\.enc\.yaml MASTERFINGERPRINT`,
		`management-clusters/demomc/secrets/.*\.enc\.yaml MASTERFINGERPRINT`,
		`management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/secrets/.*\.enc\.yaml NEWFINGERPRINT`,
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf(".sops.yaml rules (-want +got):\n%s", diff)
	}
}

func readYAML(t *testing.T, fs afero.Fs, path string, into any) {
	t.Helper()

	data, err := afero.ReadFile(fs, path)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	err = yaml.Unmarshal(data, into)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
}

// Test_NewEncryption_CreateOrganization checks that an organisation's key
// encrypts what it keeps in namespace/secrets, which the organisation's
// namespace Kustomization decrypts with the master Secret it is added to.
func Test_NewEncryption_CreateOrganization(t *testing.T) {
	fs := seedCreate(t, createMasterRule, map[string]string{
		createRepo + "/management-clusters/demomc/demomc.yaml": strings.Replace(createKustomization, "%s", "demomc-gitops", 1),
	})

	config, err := NewEncryption(common.StructureConfig{
		EncryptionKeyPair: gitopsenc.KeyPair{Fingerprint: "ORGFINGERPRINT", PrivateData: "org key", PublicData: "public"},
		EncryptionTarget:  testNamespaceTarget,
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
	checkDecryption(t, fs, createRepo+"/management-clusters/demomc/demomc.yaml", "sops-gpg-master")

	var kustomization struct {
		Resources []string `json:"resources"`
	}
	readYAML(t, fs, createOrgDir+"/namespace/kustomization.yaml", &kustomization)
	if diff := cmp.Diff([]string{"../workload-clusters", "secrets"}, kustomization.Resources); diff != "" {
		t.Errorf("namespace resources (-want +got):\n%s", diff)
	}
	readYAML(t, fs, createRepo+"/management-clusters/demomc/secrets/kustomization.yaml", &kustomization)
	if diff := cmp.Diff([]string{"demomc.gpgkey.enc.yaml"}, kustomization.Resources); diff != "" {
		t.Errorf("secrets resources (-want +got):\n%s", diff)
	}

	data, err := afero.ReadFile(fs, createSopsFile)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	rule := `management-clusters/demomc/organizations/demoorg/namespace/secrets/.*\.enc\.yaml`
	if !strings.Contains(string(data), rule) {
		t.Errorf("expected a rule for %s, got:\n%s", rule, data)
	}
}
