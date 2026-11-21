package encryption

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	gitopsenc "github.com/giantswarm/kubectl-gs/v6/internal/gitops/encryption"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
)

// sopsKeysDir holds the public keys.
const sopsKeysDir = "management-clusters/demomc/.sops.keys"

// expectedKeysKustomization is the kustomization.yaml of a directory of SOPS keys,
// before any is listed.
const expectedKeysKustomization = `apiVersion: kustomize.config.k8s.io/v1beta1
buildMetadata: [originAnnotations]
kind: Kustomization
# Flux decrypts with the keys here: deleting one stops GitOps for what it
# decrypts.
commonAnnotations:
  kustomize.toolkit.fluxcd.io/prune: disabled
labels:
  - pairs:
      giantswarm.io/managed-by: flux
      # nobody can delete them, Flux included, until this label is removed
      giantswarm.io/prevent-deletion: "true"
resources: []
`

type FsObjectExpected struct {
	RelativePath string
	Data         string
}

func Test_NewEncryption(t *testing.T) {
	keyPair := gitopsenc.KeyPair{
		Fingerprint: "12345689ABCDEF",
		PrivateData: "private key material",
		PublicData:  "public key material",
	}

	testCases := []struct {
		name            string
		config          common.StructureConfig
		expectedObjects []FsObjectExpected
	}{
		{
			name: "management cluster",
			config: common.StructureConfig{
				EncryptionKeyPair: keyPair,
				EncryptionTarget:  "secrets/",
				ManagementCluster: "demomc",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/secrets",
				},
				{
					RelativePath: "management-clusters/demomc/secrets/demomc.gpgkey.enc.yaml",
					Data:         "apiVersion: v1\nkind: Secret\nmetadata:\n    name: sops-gpg-master\n    namespace: default\n",
				},
				{
					RelativePath: "management-clusters/demomc/secrets/kustomization.yaml",
					Data:         expectedKeysKustomization,
				},
				{
					RelativePath: sopsKeysDir,
				},
				{
					RelativePath: "management-clusters/demomc/.sops.keys/demomc.12345689ABCDEF.asc",
					Data:         "public key material",
				},
			},
		},
		{
			name: "organization",
			config: common.StructureConfig{
				EncryptionKeyPair: keyPair,
				EncryptionTarget:  "namespace/secrets/",
				ManagementCluster: "demomc",
				Organization:      testOrg,
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/secrets",
				},
				{
					RelativePath: "management-clusters/demomc/secrets/demomc.gpgkey.enc.yaml",
					Data:         "apiVersion: v1\nkind: Secret\nmetadata:\n    name: sops-gpg-master\n    namespace: default\n",
				},
				{
					RelativePath: "management-clusters/demomc/secrets/kustomization.yaml",
					Data:         expectedKeysKustomization,
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/namespace/secrets",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/namespace/secrets/kustomization.yaml",
					Data:         expectedKeysKustomization,
				},
				{
					RelativePath: sopsKeysDir,
				},
				{
					RelativePath: "management-clusters/demomc/.sops.keys/demomc.12345689ABCDEF.asc",
					Data:         "public key material",
				},
			},
		},
		{
			name: "workload cluster",
			config: common.StructureConfig{
				EncryptionKeyPair: keyPair,
				EncryptionTarget:  "secrets/",
				ManagementCluster: "demomc",
				Organization:      testOrg,
				WorkloadCluster:   "demowc",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/namespace/secrets",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/namespace/secrets/demowc.gpgkey.enc.yaml",
					Data:         "apiVersion: v1\nkind: Secret\nmetadata:\n    name: sops-gpg-demowc\n    namespace: org-demoorg\n",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/namespace/secrets/kustomization.yaml",
					Data:         expectedKeysKustomization,
				},
				{
					RelativePath: sopsKeysDir,
				},
				{
					RelativePath: "management-clusters/demomc/.sops.keys/demowc.12345689ABCDEF.asc",
					Data:         "public key material",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			config, err := NewEncryption(tc.config)
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			got := make([]FsObjectExpected, 0, len(config.FsObjects))
			for _, o := range config.FsObjects {
				got = append(got, FsObjectExpected{RelativePath: o.RelativePath, Data: string(o.Data)})
			}

			if diff := cmp.Diff(tc.expectedObjects, got); diff != "" {
				t.Fatalf("unexpected file system objects (-want +got):\n%s", diff)
			}
		})
	}
}
