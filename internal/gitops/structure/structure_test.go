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

func Test_NewApp(t *testing.T) {
	testCases := []struct {
		name            string
		config          AppConfig
		expectedObjects []FsObjectExpected
	}{
		{
			name: "flawless",
			config: AppConfig{
				App:             "hello-world",
				Catalog:         "giantswarm",
				Name:            "hello-world",
				Namespace:       "default",
				Organization:    "demoorg",
				Version:         "0.3.0",
				WorkloadCluster: "demowc",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "apps",
				},
				{
					RelativePath: "apps/hello-world",
				},
				{
					RelativePath: "apps/hello-world/appcr.yaml",
					GoldenFile:   "testdata/expected/0-appcr.golden",
				},
			},
		},
		{
			name: "flawless from base",
			config: AppConfig{
				Base:            "base/apps/hello-world",
				Name:            "hello-world",
				Organization:    "demoorg",
				Version:         "0.3.0",
				WorkloadCluster: "demowc",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "apps",
				},
				{
					RelativePath: "apps/hello-world",
				},
				{
					RelativePath: "apps/hello-world/kustomization.yaml",
					GoldenFile:   "testdata/expected/1-hello_world_kustomization.golden",
				},
			},
		},
		{
			name: "flawless with cm configuration",
			config: AppConfig{
				App:          "hello-world",
				Catalog:      "giantswarm",
				Name:         "hello-world",
				Namespace:    "default",
				Organization: "demoorg",
				UserValuesConfigMap: string([]byte(`testKey: testValue
topKey:
  netedKey: nestedValue`)),
				Version:         "0.3.0",
				WorkloadCluster: "demowc",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "apps",
				},
				{
					RelativePath: "apps/hello-world",
				},
				{
					RelativePath: "apps/hello-world/appcr.yaml",
					GoldenFile:   "testdata/expected/2-appcr.golden",
				},
				{
					RelativePath: "apps/hello-world/configmap.yaml",
					GoldenFile:   "testdata/expected/2-configmap.golden",
				},
			},
		},
		{
			name: "flawless with secret configuration",
			config: AppConfig{
				App:          "hello-world",
				Catalog:      "giantswarm",
				Name:         "hello-world",
				Namespace:    "default",
				Organization: "demoorg",
				UserValuesSecret: string([]byte(`testKey: testValue
topKey:
  netedKey: nestedValue`)),
				Version:         "0.3.0",
				WorkloadCluster: "demowc",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "apps",
				},
				{
					RelativePath: "apps/hello-world",
				},
				{
					RelativePath: "apps/hello-world/appcr.yaml",
					GoldenFile:   "testdata/expected/3-appcr.golden",
				},
				{
					RelativePath: "apps/hello-world/secret.yaml",
					GoldenFile:   "testdata/expected/3-secret.golden",
				},
			},
		},
		{
			name: "flawless from base with cm configuration",
			config: AppConfig{
				Base:         "base/apps/hello-world",
				Name:         "hello-world",
				Organization: "demoorg",
				UserValuesConfigMap: string([]byte(`testKey: testValue
topKey:
  netedKey: nestedValue`)),
				WorkloadCluster: "demowc",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "apps",
				},
				{
					RelativePath: "apps/hello-world",
				},
				{
					RelativePath: "apps/hello-world/configmap.yaml",
					GoldenFile:   "testdata/expected/2-configmap.golden",
				},
				{
					RelativePath: "apps/hello-world/kustomization.yaml",
					GoldenFile:   "testdata/expected/4-hello_world_kustomization.golden",
				},
				{
					RelativePath: "apps/hello-world/patch_app_userconfig.yaml",
					GoldenFile:   "testdata/expected/4-patch_app_userconfig.golden",
				},
			},
		},
		{
			name: "flawless from base with secret configuration",
			config: AppConfig{
				Base:         "base/apps/hello-world",
				Name:         "hello-world",
				Organization: "demoorg",
				UserValuesSecret: string([]byte(`testKey: testValue
topKey:
  netedKey: nestedValue`)),
				WorkloadCluster: "demowc",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "apps",
				},
				{
					RelativePath: "apps/hello-world",
				},
				{
					RelativePath: "apps/hello-world/secret.yaml",
					GoldenFile:   "testdata/expected/3-secret.golden",
				},
				{
					RelativePath: "apps/hello-world/kustomization.yaml",
					GoldenFile:   "testdata/expected/5-hello_world_kustomization.golden",
				},
				{
					RelativePath: "apps/hello-world/patch_app_userconfig.yaml",
					GoldenFile:   "testdata/expected/5-patch_app_userconfig.golden",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			fsObjects, _, err := NewApp(tc.config)
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

func Test_NewAutomaticUpdate(t *testing.T) {
	testCases := []struct {
		name            string
		config          AutomaticUpdateConfig
		expectedObjects []FsObjectExpected
	}{
		{
			name: "flawless",
			config: AutomaticUpdateConfig{
				App:               "hello-world",
				ManagementCluster: "demomc",
				Organization:      "demoorg",
				Repository:        "gitops-demo",
				VersionRepository: "quay.io/giantswarm/hello-world",
				WorkloadCluster:   "demowc",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "automatic-updates",
				},
				{
					RelativePath: "automatic-updates/imageupdate.yaml",
					GoldenFile:   "testdata/expected/0-imageupdate.golden",
				},
				{
					RelativePath: "apps/hello-world/imagepolicy.yaml",
					GoldenFile:   "testdata/expected/0-imagepolicy.golden",
				},
				{
					RelativePath: "apps/hello-world/imagerepository.yaml",
					GoldenFile:   "testdata/expected/0-imagerepository.golden",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			fsObjects, _, err := NewAutomaticUpdate(tc.config)
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
				{
					RelativePath: "workload-clusters/demowc/apps/kustomization.yaml",
					GoldenFile:   "testdata/expected/0-apps_kustomization.golden",
				},
				{
					RelativePath: "workload-clusters/demowc/apps/patch_cluster_config.yaml",
					GoldenFile:   "testdata/expected/0-patch_cluster_config.golden",
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
					RelativePath: "workload-clusters/demowc/apps/kustomization.yaml",
					GoldenFile:   "testdata/expected/0-apps_kustomization.golden",
				},
				{
					RelativePath: "workload-clusters/demowc/apps/patch_cluster_config.yaml",
					GoldenFile:   "testdata/expected/0-patch_cluster_config.golden",
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
					RelativePath: "workload-clusters/demowc/apps/kustomization.yaml",
					GoldenFile:   "testdata/expected/0-apps_kustomization.golden",
				},
				{
					RelativePath: "workload-clusters/demowc/apps/patch_cluster_config.yaml",
					GoldenFile:   "testdata/expected/0-patch_cluster_config.golden",
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
