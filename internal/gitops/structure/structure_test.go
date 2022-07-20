package structure

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type FsObjectExpected struct {
	RelativePath string
	GoldenFile   string
}

func Test_NewManagementCluster(t *testing.T) {
	testCases := []struct {
		name            string
		config          McConfig
		expectedObjects []FsObjectExpected
	}{
		{
			name: "flawless",
			config: McConfig{
				Name:            "demomc",
				RefreshInterval: "1m",
				RefreshTimeout:  "2m",
				RepositoryName:  "gitops-demo",
				ServiceAccount:  "automation",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "demomc",
				},
				{
					RelativePath: "demomc/demomc.yaml",
					GoldenFile:   "testdata/expected/0-demomc.golden",
				},
				{
					RelativePath: "demomc/secrets",
				},
				{
					RelativePath: "demomc/.sops.keys",
				},
				{
					RelativePath: "demomc/organizations",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			fsObjects, err := NewManagementCluster(tc.config)
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if len(fsObjects) != len(tc.expectedObjects) {
				t.Fatalf("expected %d, got: %d", len(tc.expectedObjects), len(fsObjects))
			}

			for i, e := range tc.expectedObjects {
				if e.RelativePath != fsObjects[i].RelativePath {
					t.Fatalf("expected path %s, got %s", e.RelativePath, fsObjects[i].RelativePath)
				}

				if e.GoldenFile == "" {
					continue
				}

				expected, err := ioutil.ReadFile(e.GoldenFile)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				if !bytes.Equal(fsObjects[i].Data, expected) {
					t.Fatalf("want matching files \n%s\n", cmp.Diff(string(expected), string(fsObjects[i].Data)))
				}
			}
		})
	}
}

func Test_NewOrganization(t *testing.T) {
	testCases := []struct {
		name            string
		config          OrgConfig
		expectedObjects []FsObjectExpected
	}{
		{
			name: "flawless",
			config: OrgConfig{
				Name: "demoorg",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "demoorg",
				},
				{
					RelativePath: "demoorg/demoorg.yaml",
					GoldenFile:   "testdata/expected/0-demoorg.golden",
				},
				{
					RelativePath: "demoorg/workload-clusters",
				},
				{
					RelativePath: "demoorg/workload-clusters/kustomization.yaml",
					GoldenFile:   "testdata/expected/0-kustomization.golden",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			fsObjects, err := NewOrganization(tc.config)
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if len(fsObjects) != len(tc.expectedObjects) {
				t.Fatalf("expected %d, got: %d", len(tc.expectedObjects), len(fsObjects))
			}

			for i, e := range tc.expectedObjects {
				if e.RelativePath != fsObjects[i].RelativePath {
					t.Fatalf("expected path %s, got %s", e.RelativePath, fsObjects[i].RelativePath)
				}

				if e.GoldenFile == "" {
					continue
				}

				expected, err := ioutil.ReadFile(e.GoldenFile)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				if !bytes.Equal(fsObjects[i].Data, expected) {
					t.Fatalf("want matching files \n%s\n", cmp.Diff(string(expected), string(fsObjects[i].Data)))
				}
			}
		})
	}
}

func Test_NewWorkloadCluster(t *testing.T) {
	testCases := []struct {
		name            string
		config          WcConfig
		expectedObjects []FsObjectExpected
	}{
		{
			name: "flawless",
			config: WcConfig{
				ManagementCluster: "demomc",
				Name:              "demowc",
				Organization:      "demoorg",
				RepositoryName:    "gitops-demo",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "workload-clusters",
				},
				{
					RelativePath: "workload-clusters/demowc.yaml",
					GoldenFile:   "testdata/expected/0-demowc.golden",
				},
				{
					RelativePath: "workload-clusters/demowc",
				},
				{
					RelativePath: "workload-clusters/demowc/apps",
				},
				{
					RelativePath: "workload-clusters/demowc/cluster",
				},
			},
		},
		{
			name: "flawless with definition",
			config: WcConfig{
				Base:               "bases/cluster/capo",
				ClusterRelease:     "0.13.0",
				DefaultAppsRelease: "0.6.0",
				ManagementCluster:  "demomc",
				Name:               "demowc",
				Organization:       "demoorg",
				RepositoryName:     "gitops-demo",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "workload-clusters",
				},
				{
					RelativePath: "workload-clusters/demowc.yaml",
					GoldenFile:   "testdata/expected/1-demowc.golden",
				},
				{
					RelativePath: "workload-clusters/demowc",
				},
				{
					RelativePath: "workload-clusters/demowc/apps",
				},
				{
					RelativePath: "workload-clusters/demowc/cluster",
				},
				{
					RelativePath: "workload-clusters/demowc/cluster/kustomization.yaml",
					GoldenFile:   "testdata/expected/1-kustomization.golden",
				},
			},
		},
		{
			name: "flawless with definition and configuration",
			config: WcConfig{
				Base:           "bases/cluster/capo",
				ClusterRelease: "0.13.0",
				ClusterUserConfig: string([]byte(`testKey: testValue
topKey:
  netedKey: nestedValue`)),
				DefaultAppsRelease: "0.6.0",
				ManagementCluster:  "demomc",
				Name:               "demowc",
				Organization:       "demoorg",
				RepositoryName:     "gitops-demo",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "workload-clusters",
				},
				{
					RelativePath: "workload-clusters/demowc.yaml",
					GoldenFile:   "testdata/expected/1-demowc.golden",
				},
				{
					RelativePath: "workload-clusters/demowc",
				},
				{
					RelativePath: "workload-clusters/demowc/apps",
				},
				{
					RelativePath: "workload-clusters/demowc/cluster",
				},
				{
					RelativePath: "workload-clusters/demowc/cluster/kustomization.yaml",
					GoldenFile:   "testdata/expected/2-kustomization.golden",
				},
				{
					RelativePath: "workload-clusters/demowc/cluster/cluster_userconfig.yaml",
					GoldenFile:   "testdata/expected/2-cluster_userconfig.golden",
				},
				{
					RelativePath: "workload-clusters/demowc/cluster/patch_cluster_userconfig.yaml",
					GoldenFile:   "testdata/expected/2-patch_cluster_userconfig.golden",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			fsObjects, _, err := NewWorkloadCluster(tc.config)
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if len(fsObjects) != len(tc.expectedObjects) {
				t.Fatalf("expected %d objects, got: %d", len(tc.expectedObjects), len(fsObjects))
			}

			for i, e := range tc.expectedObjects {
				if e.RelativePath != fsObjects[i].RelativePath {
					t.Fatalf("expected path %s, got %s", e.RelativePath, fsObjects[i].RelativePath)
				}

				if e.GoldenFile == "" {
					continue
				}

				expected, err := ioutil.ReadFile(e.GoldenFile)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				if !bytes.Equal(fsObjects[i].Data, expected) {
					t.Fatalf("want matching files \n%s\n", cmp.Diff(string(expected), string(fsObjects[i].Data)))
				}
			}
		})
	}
}
