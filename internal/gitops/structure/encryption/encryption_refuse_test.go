package encryption

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/afero"

	gitopsenc "github.com/giantswarm/kubectl-gs/v6/internal/gitops/encryption"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
)

const refuseMCDir = createRepo + "/management-clusters/demomc"

// Test_NewEncryption_Refuses checks the cases that would change what Flux
// applies in a way that deletes something, or that can't work, and that
// nothing is written for them.
func Test_NewEncryption_Refuses(t *testing.T) {
	testCases := []struct {
		name    string
		config  common.StructureConfig
		remove  []string
		extra   map[string]string
		message string
	}{
		{
			// A kustomization.yaml listing only the new key would take every
			// other key out of the management cluster's build, and Flux would
			// delete them.
			name:    "management cluster secrets without a kustomization.yaml",
			config:  common.StructureConfig{ManagementCluster: testMC, EncryptionTarget: testTarget},
			extra:   map[string]string{refuseMCDir + "/secrets/demowc.gpgkey.enc.yaml": "kind: Secret\n"},
			message: "kustomization.yaml",
		},
		{
			name:    "organisation without a namespace Kustomization",
			config:  common.StructureConfig{ManagementCluster: testMC, Organization: testOrg, EncryptionTarget: testNamespaceTarget},
			remove:  []string{createOrgDir + "/namespace.yaml", createOrgDir + "/namespace/kustomization.yaml"},
			message: "namespace.yaml",
		},
		{
			name:    "workload cluster in an organisation without a namespace Kustomization",
			config:  common.StructureConfig{ManagementCluster: testMC, Organization: testOrg, WorkloadCluster: testWC, EncryptionTarget: testTarget},
			remove:  []string{createOrgDir + "/namespace.yaml", createOrgDir + "/namespace/kustomization.yaml"},
			message: "namespace.yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := seedCreate(t, createMasterRule, tc.extra)
			for _, path := range tc.remove {
				_ = fs.Remove(path)
			}
			before := listFiles(t, fs)

			err := createKey(fs, tc.config)
			if !errors.Is(err, creator.ValidationError) {
				t.Fatalf("expected a validation error, got: %v", err)
			}
			if !strings.Contains(err.Error(), tc.message) {
				t.Errorf("expected the error to name %s, got: %s", tc.message, err)
			}
			if after := listFiles(t, fs); strings.Join(after, "\n") != strings.Join(before, "\n") {
				t.Errorf("expected nothing written, got:\n%s", strings.Join(after, "\n"))
			}
		})
	}
}

func createKey(fs afero.Fs, config common.StructureConfig) error {
	config.EncryptionKeyPair = gitopsenc.KeyPair{Fingerprint: "NEWFINGERPRINT", PrivateData: "new key", PublicData: testPublicKey}

	creatorConfig, err := NewEncryption(config)
	if err != nil {
		return err
	}
	creatorConfig.Fs = fs
	creatorConfig.Path = createRepo
	creatorConfig.Stdout = io.Discard

	return creator.NewCreator(*creatorConfig).Create()
}

func listFiles(t *testing.T, fs afero.Fs) []string {
	t.Helper()

	var files []string
	err := afero.Walk(fs, createRepo, func(path string, _ os.FileInfo, err error) error {
		if err == nil {
			files = append(files, path)
		}
		return err
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	return files
}
