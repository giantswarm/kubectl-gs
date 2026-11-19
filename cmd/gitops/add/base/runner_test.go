package app

import (
	goflag "flag"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
	"github.com/giantswarm/kubectl-gs/v6/internal/key"
	"github.com/giantswarm/kubectl-gs/v6/test/goldenfile"
)

var update = goflag.Bool("update", false, "update .golden reference test files")

// Test_generateClusterBaseTemplates uses golden files.
//
// go test ./cmd/gitops/add/base -run Test_generateClusterBaseTemplates -update
func Test_generateClusterBaseTemplates(t *testing.T) {
	testCases := []struct {
		name     string
		config   common.StructureConfig
		resource string
		values   string
	}{
		{
			name:     "capa",
			config:   common.StructureConfig{Provider: key.ProviderCAPA, PreventDeletion: true},
			resource: "capa-cluster.golden",
			values:   "capa-cluster_config.golden",
		},
		{
			name: "capz",
			config: common.StructureConfig{
				Provider:            key.ProviderCAPZ,
				Region:              "westeurope",
				AzureSubscriptionID: "00000000-0000-0000-0000-000000000000",
				PreventDeletion:     true,
			},
			resource: "capz-cluster.golden",
			values:   "capz-cluster_config.golden",
		},
		{
			name:     "vsphere",
			config:   common.StructureConfig{Provider: key.ProviderVSphere, PreventDeletion: true},
			resource: "vsphere-cluster.golden",
			values:   "vsphere-cluster_config.golden",
		},
		{
			name:     "capa without deletion prevention",
			config:   common.StructureConfig{Provider: key.ProviderCAPA},
			resource: "capa-cluster-no-prevent-deletion.golden",
			values:   "capa-cluster_config-no-prevent-deletion.golden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			templates, err := generateClusterBaseTemplates(tc.config)
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			compareGolden(t, tc.resource, templates.ClusterResources)
			compareGolden(t, tc.values, templates.ClusterValues)
		})
	}
}

func compareGolden(t *testing.T, file, actual string) {
	t.Helper()

	gf := goldenfile.New("testdata", file)
	if *update {
		err := gf.Update([]byte(actual))
		if err != nil {
			t.Fatalf("unexpected error: %s", err.Error())
		}
		return
	}

	expected, err := gf.Read()
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	diff := cmp.Diff(string(expected), actual)
	if diff != "" {
		t.Fatalf("%s not expected, got:\n %s", file, diff)
	}
}

// Test_flagDefaults checks that a base prevents deletion unless the user asks
// otherwise.
func Test_flagDefaults(t *testing.T) {
	cmd := &cobra.Command{}
	f := &flag{}
	f.Init(cmd)

	if !f.PreventDeletion {
		t.Fatalf("--%s defaults to false, want true", flagPreventDeletion)
	}

	err := cmd.ParseFlags([]string{"--" + flagPreventDeletion + "=false"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	if f.PreventDeletion {
		t.Fatalf("--%s=false left it true", flagPreventDeletion)
	}
}
