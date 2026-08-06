package creator

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

const repoPath = "/repo"

func Test_Create_WritesNoMetadataUnlessAsked(t *testing.T) {
	fs := afero.NewMemMapFs()

	c := NewCreator(CreatorConfig{
		Fs:        fs,
		FsObjects: []*FsObject{NewFsObject("demoorg", nil, 0)},
		Path:      repoPath,
		Stdout:    new(bytes.Buffer),
	})

	err := c.Create()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	exists, err := afero.Exists(fs, metadata.FilePath(repoPath))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if exists {
		t.Error("commands that record nothing must not create a metadata file")
	}
}

func Test_Create_WritesMetadata(t *testing.T) {
	fs := &afero.Afero{Fs: afero.NewMemMapFs()}

	c := NewCreator(CreatorConfig{
		Fs:        fs,
		FsObjects: []*FsObject{NewFsObject("demoorg", nil, 0)},
		MetadataLayer: &metadata.Layer{
			Kind: metadata.LayerOrganization,
			Path: "management-clusters/demomc/organizations/demoorg",
		},
		Path:   repoPath,
		Stdout: new(bytes.Buffer),
	})

	err := c.Create()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	md, err := metadata.Load(fs, repoPath)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(md.Layers) != 1 {
		t.Fatalf("expected 1 recorded layer, got %d", len(md.Layers))
	}
	if md.Layers[0].Kind != metadata.LayerOrganization {
		t.Errorf("expected kind %q, got %q", metadata.LayerOrganization, md.Layers[0].Kind)
	}
	if md.StructureVersion != metadata.StructureVersion {
		t.Errorf("expected structure version %d, got %d", metadata.StructureVersion, md.StructureVersion)
	}
}

// The generated structure is never overwritten, but the metadata file is ours,
// so a second run has to update it rather than leave a stale copy behind.
func Test_Create_RefreshesExistingMetadata(t *testing.T) {
	fs := &afero.Afero{Fs: afero.NewMemMapFs()}

	stale := metadata.New()
	stale.Upsert(metadata.Layer{
		Kind:             metadata.LayerManagementCluster,
		Path:             "management-clusters/demomc",
		StructureVersion: metadata.StructureVersion,
		GeneratedWith:    "kubectl-gs/0.0.1",
	})
	err := metadata.Save(fs, repoPath, stale)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	c := NewCreator(CreatorConfig{
		Fs:        fs,
		FsObjects: []*FsObject{NewFsObject("demoorg", nil, 0)},
		MetadataLayer: &metadata.Layer{
			Kind: metadata.LayerOrganization,
			Path: "management-clusters/demomc/organizations/demoorg",
		},
		Path:   repoPath,
		Stdout: new(bytes.Buffer),
	})

	err = c.Create()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	md, err := metadata.Load(fs, repoPath)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(md.Layers) != 2 {
		t.Fatalf("expected the pre-existing layer to be kept alongside the new one, got %d", len(md.Layers))
	}
	if md.GeneratedWith != metadata.GeneratedWith() {
		t.Errorf("expected generatedWith to be refreshed to %q, got %q", metadata.GeneratedWith(), md.GeneratedWith)
	}
}

func Test_Create_DryRunPrintsMetadataAndWritesNothing(t *testing.T) {
	fs := afero.NewMemMapFs()
	out := new(bytes.Buffer)

	c := NewCreator(CreatorConfig{
		DryRun:    true,
		Fs:        fs,
		FsObjects: []*FsObject{NewFsObject("demoorg", nil, 0)},
		MetadataLayer: &metadata.Layer{
			Kind: metadata.LayerOrganization,
			Path: "management-clusters/demomc/organizations/demoorg",
		},
		Path:   repoPath,
		Stdout: out,
	})

	err := c.Create()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !strings.Contains(out.String(), "## METADATA ##") {
		t.Errorf("expected a METADATA section in the dry-run output, got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "management-clusters/demomc/organizations/demoorg") {
		t.Errorf("expected the layer to appear in the dry-run output, got:\n%s", out.String())
	}

	exists, err := afero.Exists(fs, metadata.FilePath(repoPath))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if exists {
		t.Error("a dry run must not write the metadata file")
	}
}
