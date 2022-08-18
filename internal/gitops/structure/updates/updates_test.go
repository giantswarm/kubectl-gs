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

func Test_NewAutomaticUpdate(t *testing.T) {
	testCases := []struct {
		name            string
		config          common.StructureConfig
		expectedObjects []FsObjectExpected
	}{
		{
			name: "flawless",
			config: common.StructureConfig{
				AppName:              "hello-world",
				ManagementCluster:    "demomc",
				Organization:         "demoorg",
				RepositoryName:       "gitops-demo",
				AppVersionRepository: "quay.io/giantswarm/hello-world",
				SkipMAPI:             true,
				WorkloadCluster:      "demowc",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/automatic-updates",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/automatic-updates/imageupdate.yaml",
					GoldenFile:   "testdata/expected/0-imageupdate.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/imagepolicy.yaml",
					GoldenFile:   "testdata/expected/0-imagepolicy.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/apps/hello-world/imagerepository.yaml",
					GoldenFile:   "testdata/expected/0-imagerepository.golden",
				},
			},
		},
		{
			name: "flawless MAPI",
			config: common.StructureConfig{
				AppName:              "hello-world",
				ManagementCluster:    "demomc",
				Organization:         "demoorg",
				RepositoryName:       "gitops-demo",
				AppVersionRepository: "quay.io/giantswarm/hello-world",
				WorkloadCluster:      "demowc",
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/automatic-updates",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/automatic-updates/imageupdate.yaml",
					GoldenFile:   "testdata/expected/0-imageupdate.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/apps/hello-world/imagepolicy.yaml",
					GoldenFile:   "testdata/expected/0-imagepolicy.golden",
				},
				{
					RelativePath: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/apps/hello-world/imagerepository.yaml",
					GoldenFile:   "testdata/expected/0-imagerepository.golden",
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
