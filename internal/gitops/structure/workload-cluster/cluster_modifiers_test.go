package wcluster

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	sigskusmod "github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/modifier/sigs-kustomization"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
)

const (
	testBase       = "bases/cluster/capa"
	testMC         = "demomc"
	testOrg        = "demoorg"
	testRelease    = "0.13.0"
	testRepository = "gitops-demo"
	testWC         = "demowc"

	// wcKustomization and appsKustomization are the files of the
	// Kustomizations applying the cluster and its apps.
	wcKustomization   = "demowc.yaml"
	appsKustomization = "demowc-apps.yaml"

	wcsDir                   = "management-clusters/demomc/organizations/demoorg/workload-clusters"
	wcsKustomizationFile     = wcsDir + "/kustomization.yaml"
	wcFile                   = wcsDir + "/" + wcKustomization
	appsFile                 = wcsDir + "/" + appsKustomization
	wcDir                    = wcsDir + "/demowc"
	wcKustomizationFile      = wcDir + "/kustomization.yaml"
	appsDir                  = wcDir + "/apps"
	appsKustomizationFile    = appsDir + "/kustomization.yaml"
	clusterDir               = wcDir + "/cluster"
	clusterKustomizationFile = clusterDir + "/kustomization.yaml"
	userConfigFile           = clusterDir + "/cluster_userconfig.yaml"
	patchFile                = clusterDir + "/patch_cluster_userconfig.yaml"

	wcKustomizationGolden   = "testdata/expected/0-wc_kustomization.golden"
	splitGolden             = "testdata/expected/1-demowc.golden"
	appsGolden              = "testdata/expected/1-demowc-apps.golden"
	appsKustomizationGolden = "testdata/expected/0-apps_kustomization.golden"
	userConfigGolden        = "testdata/expected/0-cluster_userconfig.golden"
	patchGolden             = "testdata/expected/0-patch_cluster_userconfig.golden"
)

// Test_NewWorkloadCluster_Modifiers checks what a run adds to the
// kustomization.yaml files that may exist already.
func Test_NewWorkloadCluster_Modifiers(t *testing.T) {
	testCases := []struct {
		name   string
		config common.StructureConfig
		// expected maps the kustomization.yaml files a run adds resources to
		// onto the resources they list afterwards.
		expected map[string][]string
	}{
		{
			name:     "without definition",
			config:   common.StructureConfig{},
			expected: map[string][]string{wcsKustomizationFile: {wcKustomization}},
		},
		{
			name:     "with definition",
			config:   common.StructureConfig{ClusterBase: testBase, Release: testRelease},
			expected: map[string][]string{wcsKustomizationFile: {wcKustomization, appsKustomization}},
		},
		{
			name:   "definition added to a cluster with a single Kustomization",
			config: common.StructureConfig{ClusterBase: testBase, Release: testRelease, KeepSingleKustomization: true},
			expected: map[string][]string{
				wcsKustomizationFile: {wcKustomization},
				wcKustomizationFile:  {"apps/", "cluster/"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.config.ManagementCluster = testMC
			tc.config.Organization = testOrg
			tc.config.RepositoryName = testRepository
			tc.config.WorkloadCluster = testWC

			config, err := NewWorkloadCluster(tc.config)
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			got := map[string][]string{}
			for path, m := range config.PostModifiers {
				if _, ok := m.(sigskusmod.KustomizationModifier); ok {
					got[path] = modifiedResources(t, config, path)
				}
			}

			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Fatalf("resources (-want +got):\n%s", diff)
			}
		})
	}
}

// modifiedResources runs the modifier for the file at path against what the
// run writes there, or an empty list, and returns the resources it lists.
func modifiedResources(t *testing.T, config *creator.CreatorConfig, path string) []string {
	t.Helper()

	input := []byte("resources: []\n")
	for _, o := range config.FsObjects {
		if o.RelativePath == path {
			input = o.Data
		}
	}

	out, err := config.PostModifiers[path].Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	var got struct {
		Resources []string `json:"resources"`
	}
	err = yaml.Unmarshal(out, &got)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	return got.Resources
}
