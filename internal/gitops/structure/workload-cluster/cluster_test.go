package wcluster

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
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
				ManagementCluster: testMC,
				WorkloadCluster:   testWC,
				Organization:      testOrg,
				RepositoryName:    testRepository,
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: wcsDir,
				},
				{
					RelativePath: wcFile,
					GoldenFile:   "testdata/expected/0-demowc.golden",
				},
				{
					RelativePath: wcDir,
				},
				{
					RelativePath: wcKustomizationFile,
					GoldenFile:   wcKustomizationGolden,
				},
				{
					RelativePath: appsDir,
				},
				{
					RelativePath: clusterDir,
				},
				{
					RelativePath: appsKustomizationFile,
					GoldenFile:   appsKustomizationGolden,
				},
			},
		},
		{
			name: "flawless with definition",
			config: common.StructureConfig{
				ClusterBase:       testBase,
				Release:           testRelease,
				ManagementCluster: testMC,
				WorkloadCluster:   testWC,
				Organization:      testOrg,
				RepositoryName:    testRepository,
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: wcsDir,
				},
				{
					RelativePath: wcFile,
					GoldenFile:   splitGolden,
				},
				{
					RelativePath: appsFile,
					GoldenFile:   appsGolden,
				},
				{
					RelativePath: wcDir,
				},
				{
					RelativePath: wcKustomizationFile,
					GoldenFile:   wcKustomizationGolden,
				},
				{
					RelativePath: appsDir,
				},
				{
					RelativePath: clusterDir,
				},
				{
					RelativePath: appsKustomizationFile,
					GoldenFile:   appsKustomizationGolden,
				},
				{
					RelativePath: clusterKustomizationFile,
					GoldenFile:   "testdata/expected/4-kustomization.golden",
				},
			},
		},
		{
			name: "flawless with definition and configuration",
			config: common.StructureConfig{
				ClusterBase: testBase,
				Release:     testRelease,
				ClusterUserConfig: string([]byte(`testKey: testValue
topKey:
  nestedKey: nestedValue`)),
				ManagementCluster: testMC,
				WorkloadCluster:   testWC,
				Organization:      testOrg,
				RepositoryName:    testRepository,
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: wcsDir,
				},
				{
					RelativePath: wcFile,
					GoldenFile:   splitGolden,
				},
				{
					RelativePath: appsFile,
					GoldenFile:   appsGolden,
				},
				{
					RelativePath: wcDir,
				},
				{
					RelativePath: wcKustomizationFile,
					GoldenFile:   wcKustomizationGolden,
				},
				{
					RelativePath: appsDir,
				},
				{
					RelativePath: clusterDir,
				},
				{
					RelativePath: appsKustomizationFile,
					GoldenFile:   appsKustomizationGolden,
				},
				{
					RelativePath: clusterKustomizationFile,
					GoldenFile:   "testdata/expected/1-kustomization.golden",
				},
				{
					RelativePath: userConfigFile,
					GoldenFile:   userConfigGolden,
				},
				{
					RelativePath: patchFile,
					GoldenFile:   patchGolden,
				},
			},
		},
		{
			name: "flawless with definition and full configuration",
			config: common.StructureConfig{
				ClusterBase: testBase,
				Release:     testRelease,
				ClusterUserConfig: string([]byte(`testKey: testValue
topKey:
  nestedKey: nestedValue`)),
				ManagementCluster: testMC,
				WorkloadCluster:   testWC,
				Organization:      testOrg,
				RepositoryName:    testRepository,
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: wcsDir,
				},
				{
					RelativePath: wcFile,
					GoldenFile:   splitGolden,
				},
				{
					RelativePath: appsFile,
					GoldenFile:   appsGolden,
				},
				{
					RelativePath: wcDir,
				},
				{
					RelativePath: wcKustomizationFile,
					GoldenFile:   wcKustomizationGolden,
				},
				{
					RelativePath: appsDir,
				},
				{
					RelativePath: clusterDir,
				},
				{
					RelativePath: appsKustomizationFile,
					GoldenFile:   appsKustomizationGolden,
				},
				{
					RelativePath: clusterKustomizationFile,
					GoldenFile:   "testdata/expected/2-kustomization.golden",
				},
				{
					RelativePath: userConfigFile,
					GoldenFile:   userConfigGolden,
				},
				{
					RelativePath: patchFile,
					GoldenFile:   patchGolden,
				},
			},
		},
		{
			name: "definition added to a cluster with a single Kustomization",
			config: common.StructureConfig{
				ClusterBase: testBase,
				Release:     testRelease,
				ClusterUserConfig: string([]byte(`testKey: testValue
topKey:
  nestedKey: nestedValue`)),
				KeepSingleKustomization: true,
				ManagementCluster:       testMC,
				WorkloadCluster:         testWC,
				Organization:            testOrg,
				RepositoryName:          testRepository,
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: wcsDir,
				},
				{
					RelativePath: wcFile,
					GoldenFile:   "testdata/expected/2-demowc.golden",
				},
				{
					RelativePath: wcDir,
				},
				{
					RelativePath: wcKustomizationFile,
					GoldenFile:   wcKustomizationGolden,
				},
				{
					RelativePath: appsDir,
				},
				{
					RelativePath: clusterDir,
				},
				{
					RelativePath: appsKustomizationFile,
					GoldenFile:   appsKustomizationGolden,
				},
				{
					RelativePath: clusterKustomizationFile,
					GoldenFile:   "testdata/expected/2-kustomization.golden",
				},
				{
					RelativePath: userConfigFile,
					GoldenFile:   userConfigGolden,
				},
				{
					RelativePath: patchFile,
					GoldenFile:   patchGolden,
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
