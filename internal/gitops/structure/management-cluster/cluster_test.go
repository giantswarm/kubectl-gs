package mcluster

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/giantswarm/kubectl-gs/v4/internal/gitops/encryption"
	"github.com/giantswarm/kubectl-gs/v4/internal/gitops/structure/common"
)

type FsObjectExpected struct {
	RelativePath string
	GoldenFile   string
}

func Test_NewManagementCluster(t *testing.T) {
	testCases := []struct {
		name            string
		config          common.StructureConfig
		expectedObjects []FsObjectExpected
	}{
		{
			name: "flawless",
			config: common.StructureConfig{
				ManagementCluster: "demomc",
				RepositoryName:    "gitops-demo",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc",
				},
				{
					RelativePath: "management-clusters/demomc/demomc.yaml",
					GoldenFile:   "testdata/expected/0-demomc.golden",
				},
				{
					RelativePath: "management-clusters/demomc/secrets",
				},
				{
					RelativePath: "management-clusters/demomc/secrets/demomc.gpgkey.enc.yaml",
					GoldenFile:   "testdata/expected/0-demomc.gpgkey.enc.golden",
				},
				{
					RelativePath: "management-clusters/demomc/.sops.keys",
				},
				{
					RelativePath: "management-clusters/demomc/organizations",
				},
			},
		},
		{
			name: "flawless",
			config: common.StructureConfig{
				EncryptionKeyPair: encryption.KeyPair{
					Fingerprint: "12345689ABCDEF",
					PrivateData: "private key material",
					PublicData:  "public key material",
				},
				ManagementCluster: "demomc",
				RepositoryName:    "gitops-demo",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc",
				},
				{
					RelativePath: "management-clusters/demomc/demomc.yaml",
					GoldenFile:   "testdata/expected/1-demomc.golden",
				},
				{
					RelativePath: "management-clusters/demomc/secrets",
				},
				{
					RelativePath: "management-clusters/demomc/secrets/demomc.gpgkey.enc.yaml",
					GoldenFile:   "testdata/expected/1-demomc.gpgkey.enc.golden",
				},
				{
					RelativePath: "management-clusters/demomc/.sops.keys",
				},
				{
					RelativePath: "management-clusters/demomc/.sops.keys/master.12345689ABCDEF.asc",
				},
				{
					RelativePath: "management-clusters/demomc/organizations",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			config, err := NewManagementCluster(tc.config)
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if len(config.FsObjects) != len(tc.expectedObjects) {
				t.Fatalf("expected %d, got: %d", len(tc.expectedObjects), len(config.FsObjects))
			}

			for i, e := range tc.expectedObjects {
				if e.RelativePath != config.FsObjects[i].RelativePath {
					t.Fatalf("expected path %s, got %s", e.RelativePath, config.FsObjects[i].RelativePath)
				}

				if e.GoldenFile == "" {
					continue
				}

				expected, err := os.ReadFile(e.GoldenFile)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				if !bytes.Equal(config.FsObjects[i].Data, expected) {
					t.Fatalf("want matching files \n%s\n", cmp.Diff(string(expected), string(config.FsObjects[i].Data)))
				}
			}
		})
	}
}
