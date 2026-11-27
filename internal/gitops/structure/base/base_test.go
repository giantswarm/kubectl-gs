package base

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
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
				Provider: "capa",
				ClusterBaseTemplates: common.ClusterBaseTemplates{
					ClusterResources: "foo:bar",
					ClusterValues:    "foo:bar",
				},
			},
			expectedObjects: []FsObjectExpected{
				{
					RelativePath: "bases",
				},
				{
					RelativePath: "bases/clusters",
				},
				{
					RelativePath: "bases/clusters/capa",
				},
				{
					RelativePath: "bases/clusters/capa/kustomization.yaml",
					GoldenFile:   "testdata/expected/0-capa-kustomization.golden",
				},
				{
					RelativePath: "bases/clusters/capa/cluster.yaml",
					GoldenFile:   "testdata/expected/foo-bar.golden",
				},
				{
					RelativePath: "bases/clusters/capa/cluster_config.yaml",
					GoldenFile:   "testdata/expected/foo-bar.golden",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			config, err := NewClusterBase(tc.config)
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

func Test_NewClusterBase_RecordsTheProviderDirectory(t *testing.T) {
	config, err := NewClusterBase(common.StructureConfig{Provider: "capa"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	if config.MetadataLayer.Path != "bases/clusters/capa" {
		t.Errorf("expected the base recorded at bases/clusters/capa, got %s", config.MetadataLayer.Path)
	}
}

// Test_NewClusterBase_RefusesLegacyBase checks that a provider whose base is
// still at bases/clusters/PROVIDER/template doesn't get a second base next to
// it, which adoption would not record.
func Test_NewClusterBase_RefusesLegacyBase(t *testing.T) {
	testCases := []struct {
		name      string
		existing  string
		expectErr bool
	}{
		{
			name:      "legacy base of the same provider",
			existing:  "/repo/bases/clusters/capa/template/kustomization.yaml",
			expectErr: true,
		},
		{
			name:     "legacy base of another provider",
			existing: "/repo/bases/clusters/capz/template/kustomization.yaml",
		},
		{
			name:     "no base yet",
			existing: "/repo/README.md",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs, err := addCapaBase(t, tc.existing)
			if tc.expectErr {
				assertLegacyBaseRefused(t, fs, err)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}

// addCapaBase adds a capa base to an in-memory repository at /repo that
// already holds the file existing.
func addCapaBase(t *testing.T, existing string) (afero.Fs, error) {
	t.Helper()

	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, existing, []byte("resources: []\n"), 0600)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	config, err := NewClusterBase(common.StructureConfig{
		Provider: "capa",
		ClusterBaseTemplates: common.ClusterBaseTemplates{
			ClusterResources: "foo: bar",
			ClusterValues:    "foo: bar",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	config.Fs = fs
	config.Path = "/repo"
	config.Stdout = io.Discard

	return fs, creator.NewCreator(*config).Create()
}

func assertLegacyBaseRefused(t *testing.T, fs afero.Fs, err error) {
	t.Helper()

	if !errors.Is(err, creator.ValidationError) || !strings.Contains(err.Error(), "bases/clusters/capa/template") {
		t.Fatalf("expected a validation error naming the legacy base, got: %v", err)
	}

	exists, err := afero.Exists(fs, "/repo/bases/clusters/capa/kustomization.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if exists {
		t.Fatalf("expected no base written next to the legacy one")
	}
}
