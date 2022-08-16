package structure

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/giantswarm/kubectl-gs/internal/gitops/structure/common"
)

type FsObjectExpected struct {
	RelativePath string
	GoldenFile   string
}

func Test_NewApp(t *testing.T) {
	testCases := []struct {
		name            string
		config          common.StructureConfig
		expectedObjects []FsObjectExpected
	}{
		{
			name: "flawless",
			config: common.StructureConfig{
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
					GoldenFile:   "testdata/expected/0-appcr.golden",
				},
			},
		},
		{
			name: "flawless from base",
			config: common.StructureConfig{
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
					GoldenFile:   "testdata/expected/0-hello_world_kustomization.golden",
				},
			},
		},
		{
			name: "flawless with cm configuration",
			config: common.StructureConfig{
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
					GoldenFile:   "testdata/expected/1-appcr.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/configmap.yaml",
					GoldenFile:   "testdata/expected/0-configmap.golden",
				},
			},
		},
		{
			name: "flawless with secret configuration",
			config: common.StructureConfig{
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
					GoldenFile:   "testdata/expected/2-appcr.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/secret.enc.yaml",
					GoldenFile:   "testdata/expected/0-secret.golden",
				},
			},
		},
		{
			name: "flawless from base with cm configuration",
			config: common.StructureConfig{
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
					GoldenFile:   "testdata/expected/0-configmap.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/kustomization.yaml",
					GoldenFile:   "testdata/expected/1-hello_world_kustomization.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/patch_app_userconfig.yaml",
					GoldenFile:   "testdata/expected/0-patch_app_userconfig.golden",
				},
			},
		},
		{
			name: "flawless from base with secret configuration",
			config: common.StructureConfig{
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
					GoldenFile:   "testdata/expected/0-secret.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/kustomization.yaml",
					GoldenFile:   "testdata/expected/2-hello_world_kustomization.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/patch_app_userconfig.yaml",
					GoldenFile:   "testdata/expected/1-patch_app_userconfig.golden",
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
