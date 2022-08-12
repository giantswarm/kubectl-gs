package structure

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/giantswarm/kubectl-gs/internal/gitops/encryption"
)

type FsObjectExpected struct {
	RelativePath string
	GoldenFile   string
}

func Test_NewApp(t *testing.T) {
	testCases := []struct {
		name            string
		config          StructureConfig
		expectedObjects []FsObjectExpected
	}{
		{
			name: "flawless",
			config: StructureConfig{
				App:               "hello-world",
				AppCatalog:        "giantswarm",
				ManagementCluster: "demomc",
				AppName:           "hello-world",
				AppNamespace:      "default",
				Organization:      "demoorg",
				AppVersion:        "0.3.0",
				WorkloadCluster:   "demowc",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/appcr.yaml",
					GoldenFile:   "testdata/expected/app/0-appcr.golden",
				},
			},
		},
		{
			name: "flawless from base",
			config: StructureConfig{
				AppBase:           "base/apps/hello-world",
				ManagementCluster: "demomc",
				AppName:           "hello-world",
				Organization:      "demoorg",
				AppVersion:        "0.3.0",
				WorkloadCluster:   "demowc",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/kustomization.yaml",
					GoldenFile:   "testdata/expected/app/0-hello_world_kustomization.golden",
				},
			},
		},
		{
			name: "flawless with cm configuration",
			config: StructureConfig{
				App:               "hello-world",
				AppCatalog:        "giantswarm",
				ManagementCluster: "demomc",
				AppName:           "hello-world",
				AppNamespace:      "default",
				Organization:      "demoorg",
				AppUserValuesConfigMap: string([]byte(`testKey: testValue
topKey:
  netedKey: nestedValue`)),
				AppVersion:      "0.3.0",
				WorkloadCluster: "demowc",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/appcr.yaml",
					GoldenFile:   "testdata/expected/app/1-appcr.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/configmap.yaml",
					GoldenFile:   "testdata/expected/app/0-configmap.golden",
				},
			},
		},
		{
			name: "flawless with secret configuration",
			config: StructureConfig{
				App:               "hello-world",
				AppCatalog:        "giantswarm",
				ManagementCluster: "demomc",
				AppName:           "hello-world",
				AppNamespace:      "default",
				Organization:      "demoorg",
				AppUserValuesSecret: string([]byte(`testKey: testValue
topKey:
  netedKey: nestedValue`)),
				AppVersion:      "0.3.0",
				WorkloadCluster: "demowc",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/appcr.yaml",
					GoldenFile:   "testdata/expected/app/2-appcr.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/secret.enc.yaml",
					GoldenFile:   "testdata/expected/app/0-secret.golden",
				},
			},
		},
		{
			name: "flawless from base with cm configuration",
			config: StructureConfig{
				AppBase:           "base/apps/hello-world",
				ManagementCluster: "demomc",
				AppName:           "hello-world",
				Organization:      "demoorg",
				AppUserValuesConfigMap: string([]byte(`testKey: testValue
topKey:
  netedKey: nestedValue`)),
				WorkloadCluster: "demowc",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/configmap.yaml",
					GoldenFile:   "testdata/expected/app/0-configmap.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/kustomization.yaml",
					GoldenFile:   "testdata/expected/app/1-hello_world_kustomization.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/patch_app_userconfig.yaml",
					GoldenFile:   "testdata/expected/app/0-patch_app_userconfig.golden",
				},
			},
		},
		{
			name: "flawless from base with secret configuration",
			config: StructureConfig{
				AppBase:           "base/apps/hello-world",
				ManagementCluster: "demomc",
				AppName:           "hello-world",
				Organization:      "demoorg",
				AppUserValuesSecret: string([]byte(`testKey: testValue
topKey:
  netedKey: nestedValue`)),
				WorkloadCluster: "demowc",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/secret.enc.yaml",
					GoldenFile:   "testdata/expected/app/0-secret.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/kustomization.yaml",
					GoldenFile:   "testdata/expected/app/2-hello_world_kustomization.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/patch_app_userconfig.yaml",
					GoldenFile:   "testdata/expected/app/1-patch_app_userconfig.golden",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			config, err := NewApp(tc.config)
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

				expected, err := ioutil.ReadFile(e.GoldenFile)
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

func Test_NewAutomaticUpdate(t *testing.T) {
	testCases := []struct {
		name            string
		config          StructureConfig
		expectedObjects []FsObjectExpected
	}{
		{
			name: "flawless",
			config: StructureConfig{
				AppName:              "hello-world",
				ManagementCluster:    "demomc",
				Organization:         "demoorg",
				RepositoryName:       "gitops-demo",
				AppVersionRepository: "quay.io/giantswarm/hello-world",
				WorkloadCluster:      "demowc",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/automatic-updates",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/automatic-updates/imageupdate.yaml",
					GoldenFile:   "testdata/expected/img/0-imageupdate.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/imagepolicy.yaml",
					GoldenFile:   "testdata/expected/img/0-imagepolicy.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/imagerepository.yaml",
					GoldenFile:   "testdata/expected/img/0-imagerepository.golden",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			config, err := NewAutomaticUpdate(tc.config)
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

				expected, err := ioutil.ReadFile(e.GoldenFile)
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

func Test_NewManagementCluster(t *testing.T) {
	testCases := []struct {
		name            string
		config          StructureConfig
		expectedObjects []FsObjectExpected
	}{
		{
			name: "flawless",
			config: StructureConfig{
				ManagementCluster: "demomc",
				RepositoryName:    "gitops-demo",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc",
				},
				{
					RelativePath: "management-clusters/demomc/demomc.yaml",
					GoldenFile:   "testdata/expected/mc/0-demomc.golden",
				},
				{
					RelativePath: "management-clusters/demomc/secrets",
				},
				{
					RelativePath: "management-clusters/demomc/secrets/demomc.gpgkey.enc.yaml",
					GoldenFile:   "testdata/expected/mc/0-demomc.gpgkey.enc.golden",
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
			config: StructureConfig{
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
					GoldenFile:   "testdata/expected/mc/1-demomc.golden",
				},
				{
					RelativePath: "management-clusters/demomc/secrets",
				},
				{
					RelativePath: "management-clusters/demomc/secrets/demomc.gpgkey.enc.yaml",
					GoldenFile:   "testdata/expected/mc/1-demomc.gpgkey.enc.golden",
				},
				{
					RelativePath: "management-clusters/demomc/.sops.keys",
				},
				{
					RelativePath: "management-clusters/demomc/.sops.keys/.sops.master.12345689ABCDEF.asc",
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

				expected, err := ioutil.ReadFile(e.GoldenFile)
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

func Test_NewOrganization(t *testing.T) {
	testCases := []struct {
		name            string
		config          StructureConfig
		expectedObjects []FsObjectExpected
	}{
		{
			name: "flawless",
			config: StructureConfig{
				ManagementCluster: "demomc",
				Organization:      "demoorg",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/demoorg.yaml",
					GoldenFile:   "testdata/expected/org/0-demoorg.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/kustomization.yaml",
					GoldenFile:   "testdata/expected/org/0-kustomization.golden",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			config, err := NewOrganization(tc.config)
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

				expected, err := ioutil.ReadFile(e.GoldenFile)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				if !bytes.Equal(config.FsObjects[i].Data, expected) {
					t.Fatalf("want matching files \n%s\n", cmp.Diff(string(config.FsObjects[i].Data), string(expected)))
				}
			}
		})
	}
}

func Test_NewWorkloadCluster(t *testing.T) {
	testCases := []struct {
		name            string
		config          StructureConfig
		expectedObjects []FsObjectExpected
	}{
		{
			name: "flawless",
			config: StructureConfig{
				ManagementCluster: "demomc",
				WorkloadCluster:   "demowc",
				Organization:      "demoorg",
				RepositoryName:    "gitops-demo",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/secrets/demowc.gpgkey.enc.yaml",
					GoldenFile:   "testdata/expected/wc/0-secret.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc.yaml",
					GoldenFile:   "testdata/expected/wc/0-demowc.golden",
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
					GoldenFile:   "testdata/expected/wc/0-apps_kustomization.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/patch_cluster_config.yaml",
					GoldenFile:   "testdata/expected/wc/0-patch_cluster_config.golden",
				},
			},
		},
		{
			name: "flawless with definition",
			config: StructureConfig{
				ClusterBase:        "bases/cluster/capo",
				ClusterRelease:     "0.13.0",
				DefaultAppsRelease: "0.6.0",
				ManagementCluster:  "demomc",
				WorkloadCluster:    "demowc",
				Organization:       "demoorg",
				RepositoryName:     "gitops-demo",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/secrets/demowc.gpgkey.enc.yaml",
					GoldenFile:   "testdata/expected/wc/0-secret.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc.yaml",
					GoldenFile:   "testdata/expected/wc/1-demowc.golden",
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
					GoldenFile:   "testdata/expected/wc/0-apps_kustomization.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/patch_cluster_config.yaml",
					GoldenFile:   "testdata/expected/wc/0-patch_cluster_config.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster/kustomization.yaml",
					GoldenFile:   "testdata/expected/wc/0-kustomization.golden",
				},
			},
		},
		{
			name: "flawless with definition and configuration",
			config: StructureConfig{
				ClusterBase:    "bases/cluster/capo",
				ClusterRelease: "0.13.0",
				ClusterUserConfig: string([]byte(`testKey: testValue
topKey:
  nestedKey: nestedValue`)),
				DefaultAppsRelease: "0.6.0",
				ManagementCluster:  "demomc",
				WorkloadCluster:    "demowc",
				Organization:       "demoorg",
				RepositoryName:     "gitops-demo",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/secrets/demowc.gpgkey.enc.yaml",
					GoldenFile:   "testdata/expected/wc/0-secret.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc.yaml",
					GoldenFile:   "testdata/expected/wc/1-demowc.golden",
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
					GoldenFile:   "testdata/expected/wc/0-apps_kustomization.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/patch_cluster_config.yaml",
					GoldenFile:   "testdata/expected/wc/0-patch_cluster_config.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster/kustomization.yaml",
					GoldenFile:   "testdata/expected/wc/1-kustomization.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster/cluster_userconfig.yaml",
					GoldenFile:   "testdata/expected/wc/0-cluster_userconfig.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster/patch_cluster_userconfig.yaml",
					GoldenFile:   "testdata/expected/wc/0-patch_cluster_userconfig.golden",
				},
			},
		},
		{
			name: "flawless with definition and full configuration",
			config: StructureConfig{
				ClusterBase:    "bases/cluster/capo",
				ClusterRelease: "0.13.0",
				ClusterUserConfig: string([]byte(`testKey: testValue
topKey:
  nestedKey: nestedValue`)),
				DefaultAppsRelease: "0.6.0",
				DefaultAppsUserConfig: string([]byte(`testKey: testValue
otherTopKey:
  nestedOtherKey: nestedOtherValue`)),
				ManagementCluster: "demomc",
				WorkloadCluster:   "demowc",
				Organization:      "demoorg",
				RepositoryName:    "gitops-demo",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/secrets/demowc.gpgkey.enc.yaml",
					GoldenFile:   "testdata/expected/wc/0-secret.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc.yaml",
					GoldenFile:   "testdata/expected/wc/1-demowc.golden",
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
					GoldenFile:   "testdata/expected/wc/0-apps_kustomization.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/patch_cluster_config.yaml",
					GoldenFile:   "testdata/expected/wc/0-patch_cluster_config.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster/kustomization.yaml",
					GoldenFile:   "testdata/expected/wc/2-kustomization.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster/cluster_userconfig.yaml",
					GoldenFile:   "testdata/expected/wc/0-cluster_userconfig.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster/default_apps_userconfig.yaml",
					GoldenFile:   "testdata/expected/wc/0-default_apps_userconfig.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster/patch_cluster_userconfig.yaml",
					GoldenFile:   "testdata/expected/wc/0-patch_cluster_userconfig.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster/patch_default_apps_userconfig.yaml",
					GoldenFile:   "testdata/expected/wc/0-patch_default_apps_userconfig.golden",
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

				expected, err := ioutil.ReadFile(e.GoldenFile)
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
