package wcluster

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/giantswarm/kubectl-gs/v4/internal/gitops/structure/common"
)

type FsObjectExpected struct {
	RelativePath string
	GoldenFile   string
}

func Test_NewWorkloadCluster(t *testing.T) {
	testCases := []struct {
		name            string
		config          common.StructureConfig
		expectedObjects []FsObjectExpected
	}{
		{
			name: "flawless",
			config: common.StructureConfig{
				ManagementCluster: "demomc",
				WorkloadCluster:   "demowc",
				Organization:      "demoorg",
				SkipMAPI:          true,
				RepositoryName:    "gitops-demo",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/secrets/demowc.gpgkey.enc.yaml",
					GoldenFile:   "testdata/expected/0-secret.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc.yaml",
					GoldenFile:   "testdata/expected/0-demowc.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/kustomization.yaml",
					GoldenFile:   "testdata/expected/0-apps_kustomization.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/patch_cluster_config.yaml",
					GoldenFile:   "testdata/expected/0-patch_cluster_config.golden",
				},
			},
		},
		{
			name: "flawless with definition",
			config: common.StructureConfig{
				ClusterBase:       "bases/cluster/capo",
				ClusterRelease:    "0.13.0",
				ManagementCluster: "demomc",
				WorkloadCluster:   "demowc",
				Organization:      "demoorg",
				SkipMAPI:          true,
				RepositoryName:    "gitops-demo",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/secrets/demowc.gpgkey.enc.yaml",
					GoldenFile:   "testdata/expected/0-secret.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc.yaml",
					GoldenFile:   "testdata/expected/1-demowc.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/kustomization.yaml",
					GoldenFile:   "testdata/expected/0-apps_kustomization.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/patch_cluster_config.yaml",
					GoldenFile:   "testdata/expected/0-patch_cluster_config.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster/kustomization.yaml",
					GoldenFile:   "testdata/expected/0-kustomization.golden",
				},
			},
		},
		{
			name: "flawless with definition and configuration",
			config: common.StructureConfig{
				ClusterBase:    "bases/cluster/capo",
				ClusterRelease: "0.13.0",
				ClusterUserConfig: string([]byte(`testKey: testValue
topKey:
  nestedKey: nestedValue`)),
				ManagementCluster: "demomc",
				WorkloadCluster:   "demowc",
				Organization:      "demoorg",
				SkipMAPI:          true,
				RepositoryName:    "gitops-demo",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/secrets/demowc.gpgkey.enc.yaml",
					GoldenFile:   "testdata/expected/0-secret.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc.yaml",
					GoldenFile:   "testdata/expected/1-demowc.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/kustomization.yaml",
					GoldenFile:   "testdata/expected/0-apps_kustomization.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/patch_cluster_config.yaml",
					GoldenFile:   "testdata/expected/0-patch_cluster_config.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster/kustomization.yaml",
					GoldenFile:   "testdata/expected/1-kustomization.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster/cluster_userconfig.yaml",
					GoldenFile:   "testdata/expected/0-cluster_userconfig.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster/patch_cluster_userconfig.yaml",
					GoldenFile:   "testdata/expected/0-patch_cluster_userconfig.golden",
				},
			},
		},
		{
			name: "flawless with definition and full configuration",
			config: common.StructureConfig{
				ClusterBase:    "bases/cluster/capo",
				ClusterRelease: "0.13.0",
				ClusterUserConfig: string([]byte(`testKey: testValue
topKey:
  nestedKey: nestedValue`)),
				ManagementCluster: "demomc",
				WorkloadCluster:   "demowc",
				Organization:      "demoorg",
				SkipMAPI:          true,
				RepositoryName:    "gitops-demo",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/secrets/demowc.gpgkey.enc.yaml",
					GoldenFile:   "testdata/expected/0-secret.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc.yaml",
					GoldenFile:   "testdata/expected/1-demowc.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/kustomization.yaml",
					GoldenFile:   "testdata/expected/0-apps_kustomization.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/patch_cluster_config.yaml",
					GoldenFile:   "testdata/expected/0-patch_cluster_config.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster/kustomization.yaml",
					GoldenFile:   "testdata/expected/2-kustomization.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster/cluster_userconfig.yaml",
					GoldenFile:   "testdata/expected/0-cluster_userconfig.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster/patch_cluster_userconfig.yaml",
					GoldenFile:   "testdata/expected/0-patch_cluster_userconfig.golden",
				},
			},
		},
		{
			name: "flawless MAPI with definition and full configuration",
			config: common.StructureConfig{
				ClusterBase:    "bases/cluster/capo",
				ClusterRelease: "0.13.0",
				ClusterUserConfig: string([]byte(`testKey: testValue
topKey:
  nestedKey: nestedValue`)),
				ManagementCluster: "demomc",
				WorkloadCluster:   "demowc",
				Organization:      "demoorg",
				RepositoryName:    "gitops-demo",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/secrets/demowc.gpgkey.enc.yaml",
					GoldenFile:   "testdata/expected/0-secret.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc.yaml",
					GoldenFile:   "testdata/expected/2-demowc.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/apps",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/cluster",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/apps/kustomization.yaml",
					GoldenFile:   "testdata/expected/0-apps_kustomization.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/apps/patch_cluster_config.yaml",
					GoldenFile:   "testdata/expected/0-patch_cluster_config.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/cluster/kustomization.yaml",
					GoldenFile:   "testdata/expected/3-kustomization.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/cluster/cluster_userconfig.yaml",
					GoldenFile:   "testdata/expected/0-cluster_userconfig.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/cluster/patch_cluster_userconfig.yaml",
					GoldenFile:   "testdata/expected/0-patch_cluster_userconfig.golden",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			config, err := NewWorkloadCluster(tc.config)
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if len(config.FsObjects) != len(tc.expectedObjects) {
				t.Fatalf("expected %d objects, got: %d", len(tc.expectedObjects), len(config.FsObjects))
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
					t.Fatalf("want matching files for '%s', got \n\n%s\n", e.RelativePath, cmp.Diff(string(expected), string(config.FsObjects[i].Data)))
				}
			}
		})
	}
}
